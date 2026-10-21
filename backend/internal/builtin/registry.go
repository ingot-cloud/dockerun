// Package builtin embeds Distribution; it does not reimplement blob or manifest storage.
package builtin

import (
	"bytes"
	"context"
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/identity"
	"dockerun/internal/models"
	"encoding/json"
	"fmt"
	"github.com/distribution/distribution/v3/configuration"
	"github.com/distribution/distribution/v3/registry/auth"
	"github.com/distribution/distribution/v3/registry/handlers"
	"github.com/distribution/distribution/v3/registry/storage"
	"github.com/distribution/distribution/v3/registry/storage/driver/filesystem"
	"github.com/golang-jwt/jwt/v5"
	digest "github.com/opencontainers/go-digest"
	"gorm.io/gorm/clause"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var ReferenceGate sync.RWMutex
var Engine http.Handler
var gate sync.RWMutex
var readonly bool
var maintenanceMu sync.Mutex

func init() {
	auth.Register("dockerun", func(options map[string]any) (auth.AccessController, error) { return controller{}, nil })
}
func Init() error {
	cfg := config.Load()
	if !cfg.RegistryEnabled {
		return nil
	}
	if cfg.RegistrySecret == "" {
		return fmt.Errorf("registry signing key unavailable")
	}
	c := &configuration.Configuration{Version: configuration.CurrentVersion, Storage: configuration.Storage{"filesystem": configuration.Parameters{"rootdirectory": filepath.Join(cfg.DataDir, "registry")}, "delete": configuration.Parameters{"enabled": true}, "maintenance": configuration.Parameters{"uploadpurging": map[any]any{"enabled": false}}}, Auth: configuration.Auth{"dockerun": configuration.Parameters{}}, HTTP: configuration.HTTP{Secret: config.RegistryHTTPSecret(), RelativeURLs: true}}
	Engine = handlers.NewApp(context.Background(), c)
	return nil
}

type challenge struct{ access []auth.Access }

func (challenge) Error() string { return "authentication required" }
func (c challenge) SetHeaders(r *http.Request, w http.ResponseWriter) {
	realm := strings.TrimSuffix(config.Load().PublicURL, "/") + "/registry/token"
	value := fmt.Sprintf("Bearer realm=%q,service=%q", realm, "dockerun-registry")
	group := map[string][]string{}
	for _, a := range c.access {
		k := a.Type + ":" + a.Name
		group[k] = append(group[k], a.Action)
	}
	scopes := []string{}
	for k, actions := range group {
		sort.Strings(actions)
		scopes = append(scopes, k+":"+strings.Join(actions, ","))
	}
	sort.Strings(scopes)
	if len(scopes) > 0 {
		value += fmt.Sprintf(",scope=%q", strings.Join(scopes, " "))
	}
	w.Header().Set("WWW-Authenticate", value)
}

type controller struct{}

func (controller) Authorized(r *http.Request, access ...auth.Access) (*auth.Grant, error) {
	raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	u, c, err := identity.Validate(raw, "registry")
	if err != nil {
		// System credentials are independent, short lived and restricted to exact repositories.
		var ic identity.Claims
		_, e := jwt.ParseWithClaims(raw, &ic, func(t *jwt.Token) (any, error) { return []byte(config.Load().RegistrySecret), nil }, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer("dockerun-internal"), jwt.WithAudience("dockerun-registry"), jwt.WithExpirationRequired())
		if e != nil || ic.Purpose != "registry-internal" {
			return nil, challenge{access}
		}
		for _, a := range access {
			if a.Type != "repository" || a.Action != "pull" || !identity.Has(ic.Scopes, "repository:"+a.Name+":pull") {
				return nil, challenge{access}
			}
		}
		return &auth.Grant{User: auth.UserInfo{Name: "dockerun-executor"}}, nil
	}
	if !u.CanRegistry("pull") {
		return nil, challenge{access}
	}
	resources := []auth.Resource{}
	for _, a := range access {
		scope := a.Type + ":" + a.Name + ":" + a.Action
		allowed := a.Type == "repository" && u.CanRegistry(a.Action) || a.Type == "registry" && a.Name == "catalog" && u.CanRegistry("pull")
		if a.Action == "delete" {
			allowed = u.IsAdmin
		}
		if !allowed || !identity.Has(c.Scopes, scope) {
			return nil, challenge{access}
		}
		resources = append(resources, a.Resource)
	}
	return &auth.Grant{User: auth.UserInfo{Name: u.Username}, Resources: resources}, nil
}

func Token(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != "GET" && r.Method != "POST" {
		w.WriteHeader(405)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_request"})
		return
	}
	if r.Method == "POST" && r.Form.Get("grant_type") != "password" {
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"error": "unsupported_grant_type"})
		return
	}
	if service := r.Form.Get("service"); service != "" && service != "dockerun-registry" {
		http.Error(w, "invalid service", 400)
		return
	}
	name, password, ok := r.BasicAuth()
	if !ok && r.Method == "POST" {
		name, password = r.Form.Get("username"), r.Form.Get("password")
		ok = name != "" && password != ""
	}
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Dockerun"`)
		http.Error(w, "credentials required", 401)
		return
	}
	u, err := identity.Authenticate(name, password)
	var limit []string
	if strings.HasPrefix(password, "dku_") {
		u, limit, err = identity.Automation(password)
		if err == nil && u.Username != name {
			err = fmt.Errorf("invalid user")
		}
	}
	if err != nil || !u.CanRegistry("pull") {
		http.Error(w, "invalid credentials or registry permission", 401)
		return
	}
	var scopes []string
	for _, param := range r.Form["scope"] {
		for _, s := range strings.Fields(param) {
			p := strings.SplitN(s, ":", 3)
			if len(p) != 3 {
				continue
			}
			for _, a := range strings.Split(p[2], ",") {
				allowed := p[0] == "repository" && u.CanRegistry(a) || p[0] == "registry" && p[1] == "catalog" && a == "*" && u.CanRegistry("pull") || p[0] == "repository" && a == "delete" && u.IsAdmin
				if strings.HasPrefix(password, "dku_") {
					allowed = allowed && identity.Has(limit, a)
				}
				if allowed {
					scopes = append(scopes, p[0]+":"+p[1]+":"+a)
				}
			}
		}
	}
	token, err := identity.Issue(u, "registry", scopes, 5*time.Minute)
	if err != nil {
		http.Error(w, "token unavailable", 503)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"token": token, "access_token": token, "expires_in": 300, "issued_at": time.Now().UTC().Format(time.RFC3339)})
}
func InternalToken(repositories []string) (string, error) {
	scopes := []string{}
	for _, r := range repositories {
		scopes = append(scopes, "repository:"+r+":pull")
	}
	c := identity.Claims{Purpose: "registry-internal", Scopes: scopes, RegisteredClaims: jwt.RegisteredClaims{Issuer: "dockerun-internal", Audience: jwt.ClaimStrings{"dockerun-registry"}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)), IssuedAt: jwt.NewNumericDate(time.Now())}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString([]byte(config.Load().RegistrySecret))
}

type response struct {
	http.ResponseWriter
	status int
}

func (w *response) WriteHeader(s int) { w.status = s; w.ResponseWriter.WriteHeader(s) }
func (w *response) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = 200
	}
	return w.ResponseWriter.Write(b)
}
func Serve(w http.ResponseWriter, r *http.Request) {
	if r.Method == "DELETE" {
		ReferenceGate.Lock()
		defer ReferenceGate.Unlock()
	}
	gate.RLock()
	defer gate.RUnlock()
	if Engine == nil {
		http.Error(w, "内置仓库未启用", 503)
		return
	}
	write := r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS"
	if readonly && write {
		http.Error(w, "仓库只读维护中", 503)
		return
	}
	var initialized int64
	database.DB.Model(&models.User{}).Where("is_admin = ? AND enabled = ? AND password <> ?", true, true, "").Count(&initialized)
	if initialized == 0 {
		http.Error(w, "请先初始化管理员", 503)
		return
	}
	manifest := strings.LastIndex(r.URL.Path, "/manifests/")
	if r.Method == "DELETE" {
		u, _, err := identity.Validate(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), "registry")
		if err != nil || !u.IsAdmin {
			challenge{}.SetHeaders(r, w)
			http.Error(w, "administrator required", 401)
			return
		}
		// Layer deletion cannot establish reference safety, and is only done by offline GC.
		if manifest < 0 {
			http.Error(w, "镜像层只能通过仓库维护清理", 403)
			return
		}
		if err := Protected(r.URL.Path[manifest+11:]); err != nil {
			http.Error(w, err.Error(), 409)
			return
		}
	}
	var intent *models.ImageEvent
	if r.Method == "PUT" && manifest > 4 {
		b, err := io.ReadAll(io.LimitReader(r.Body, 8<<20+1))
		if err != nil || len(b) > 8<<20 {
			http.Error(w, "manifest too large", 413)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(b))
		u, claims, err := identity.Validate(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), "registry")
		if err == nil {
			repo := strings.TrimPrefix(r.URL.Path[:manifest], "/v2/")
			ref, _ := url.PathUnescape(r.URL.Path[manifest+11:])
			var reg models.Registry
			database.DB.Where("kind = ?", "builtin").First(&reg)
			d := digest.FromBytes(b).String()
			key := identity.Hash(fmt.Sprintf("%d:%s:%s:%s:%d:%s", reg.ID, repo, ref, d, u.ID, claims.ID))
			e := models.ImageEvent{Imported: claims.Origin == "bundle", RegistryID: reg.ID, Repository: repo, Reference: ref, Digest: d, MediaType: r.Header.Get("Content-Type"), UserID: u.ID, Key: key}
			if err := database.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&e).Error; err != nil {
				http.Error(w, "事件持久化失败", 503)
				return
			}
			if e.ID == 0 {
				database.DB.Where("`key` = ?", key).First(&e)
			}
			intent = &e
		}
	}
	rw := &response{ResponseWriter: w}
	Engine.ServeHTTP(rw, r)
	if intent != nil && rw.status == 201 {
		database.DB.Model(intent).Update("committed", true)
		identity.Audit(intent.UserID, "registry.push", intent.Repository+"@"+intent.Digest)
	}
	if intent != nil && rw.status != 201 {
		database.DB.Where("id = ? AND committed = ?", intent.ID, false).Delete(&models.ImageEvent{})
	}
	if r.Method == "DELETE" && rw.status == 202 {
		u, _, _ := identity.Validate(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), "registry")
		if u != nil {
			identity.Audit(u.ID, "registry.delete", r.URL.Path)
		}
	}
}

func Protected(d string) error {
	if err := digest.Digest(d).Validate(); err != nil {
		return fmt.Errorf("必须使用有效的 manifest digest 删除")
	}
	var n int64
	database.DB.Model(&models.Release{}).Where("retained = ? AND images LIKE ?", true, "%"+d+"%").Count(&n)
	if n > 0 {
		return fmt.Errorf("版本仍被当前部署或恢复历史引用")
	}
	// Protect child manifests referenced by any stored index, even when untagged.
	root := filepath.Join(config.Load().DataDir, "registry", "docker", "registry", "v2", "blobs", "sha256")
	protected := false
	filepath.WalkDir(root, func(p string, e os.DirEntry, err error) error {
		if err != nil || e.IsDir() || e.Name() != "data" {
			return nil
		}
		info, _ := e.Info()
		if info.Size() > 8<<20 {
			return nil
		}
		b, _ := os.ReadFile(p)
		var m struct {
			Manifests []struct {
				Digest string `json:"digest"`
			} `json:"manifests"`
		}
		if json.Unmarshal(b, &m) == nil {
			for _, v := range m.Manifests {
				if v.Digest == d {
					protected = true
				}
			}
		}
		return nil
	})
	if protected {
		return fmt.Errorf("版本仍被多架构索引引用")
	}
	return nil
}
func SetReadOnly(v bool) { gate.Lock(); defer gate.Unlock(); readonly = v }
func ReadOnly() bool     { gate.RLock(); defer gate.RUnlock(); return readonly }
func GarbageCollect(ctx context.Context) error {
	maintenanceMu.Lock()
	defer maintenanceMu.Unlock()
	gate.Lock()
	defer gate.Unlock()
	if !readonly {
		return fmt.Errorf("请先开启只读维护窗口")
	}
	driver, err := filesystem.FromParameters(map[string]any{"rootdirectory": filepath.Join(config.Load().DataDir, "registry")})
	if err != nil {
		return err
	}
	ns, err := storage.NewRegistry(ctx, driver, storage.EnableDelete)
	if err != nil {
		return err
	}
	return storage.MarkAndSweep(ctx, driver, ns, storage.GCOpts{RemoveUntagged: false})
}
