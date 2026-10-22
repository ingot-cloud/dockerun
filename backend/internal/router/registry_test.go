package router_test

import (
	"bytes"
	"dockerun/internal/builtin"
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/identity"
	"dockerun/internal/models"
	"dockerun/internal/router"
	"encoding/json"
	"fmt"
	digest "github.com/opencontainers/go-digest"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIntegratedRegistryAndPermissions(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	t.Setenv("GIN_MODE", "test")
	t.Setenv("PUBLIC_URL", "https://registry.example.test")
	if err := config.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := database.Init(config.Load().DatabasePath); err != nil {
		t.Fatal(err)
	}
	if err := builtin.Init(); err != nil {
		t.Fatal(err)
	}
	r := router.Setup()
	users := map[string]models.User{}
	for _, name := range []string{"admin", "none", "read_only", "read_write"} {
		var u models.User
		if name == "admin" {
			database.DB.First(&u, 1)
		} else {
			u = models.User{Username: name, Enabled: true, RegistryAccess: name, AuthVersion: 1}
		}
		u.HashPassword("test-password")
		if name == "admin" {
			database.DB.Save(&u)
		} else {
			database.DB.Create(&u)
		}
		users[name] = u
	}
	req := func(method, path, token string, body []byte) *httptest.ResponseRecorder {
		q := httptest.NewRequest(method, path, bytes.NewReader(body))
		if token != "" {
			q.Header.Set("Authorization", "Bearer "+token)
		}
		q.Header.Set("Content-Type", "application/json")
		q.Header.Set("Accept", "application/vnd.oci.image.manifest.v1+json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, q)
		return w
	}
	token := func(name, scope string) string {
		q := httptest.NewRequest("GET", "/registry/token?service=dockerun-registry&scope="+scope, nil)
		q.SetBasicAuth(name, "test-password")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, q)
		if w.Code != 200 {
			t.Fatalf("token %s: %d %s", name, w.Code, w.Body.String())
		}
		var v struct {
			Token string `json:"token"`
		}
		json.Unmarshal(w.Body.Bytes(), &v)
		return v.Token
	}
	form := url.Values{"grant_type": {"password"}, "username": {"read_write"}, "password": {"test-password"}, "service": {"dockerun-registry"}, "scope": {"repository:demo/app:pull,push"}}
	oauth := httptest.NewRequest("POST", "/registry/token", strings.NewReader(form.Encode()))
	oauth.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ow := httptest.NewRecorder()
	r.ServeHTTP(ow, oauth)
	if ow.Code != 200 {
		t.Fatalf("BuildKit OAuth: %d %s", ow.Code, ow.Body.String())
	}
	var ot struct {
		Token string `json:"access_token"`
	}
	json.Unmarshal(ow.Body.Bytes(), &ot)
	if w := req("POST", "/v2/demo/app/blobs/uploads/", ot.Token, nil); w.Code != 202 {
		t.Fatalf("OAuth token scope: %d %s", w.Code, w.Body.String())
	}
	if w := req("GET", "/v2/", "", nil); w.Code != 401 || !strings.Contains(w.Header().Get("WWW-Authenticate"), "/registry/token") {
		t.Fatalf("challenge %d %s", w.Code, w.Body.String())
	}
	for name, u := range users {
		web, _ := identity.Issue(&u, "web", nil, time.Hour)
		w := req("GET", "/api/users", web, nil)
		if name != "admin" && w.Code != 403 {
			t.Fatalf("ordinary Docker access %s: %d", name, w.Code)
		}
		if name != "admin" {
			for _, path := range []string{"/api/builtin/status", "/api/builtin/maintenance", "/api/system/backups", "/api/system/backups/1/download"} {
				if w := req("GET", path, web, nil); w.Code != 403 {
					t.Fatalf("ordinary maintenance access %s %s: %d", name, path, w.Code)
				}
			}
			if w := req("GET", "/api/images/sha256:test/history", web, nil); w.Code != 403 {
				t.Fatalf("ordinary image history access %s: %d", name, w.Code)
			}
		}
		if name == "none" {
			if w := req("GET", "/api/builtin/catalog", web, nil); w.Code != 403 {
				t.Fatalf("none catalog: %d", w.Code)
			}
			continue
		}
		if w := req("GET", "/api/builtin/catalog", web, nil); w.Code != 200 || !strings.Contains(w.Body.String(), `"repositories":[]`) {
			t.Fatalf("empty builtin catalog for %s: %d %s", name, w.Code, w.Body.String())
		}
		native := token(name, "repository:demo/app:pull,push,delete")
		if w := req("GET", "/api/auth/me", native, nil); w.Code != 401 {
			t.Fatalf("registry token accepted as web: %d", w.Code)
		}
		if w := req("GET", "/v2/", web, nil); w.Code != 401 {
			t.Fatalf("web token accepted as registry: %d", w.Code)
		}
		if name == "read_only" {
			if w := req("POST", "/v2/demo/app/blobs/uploads/", native, nil); w.Code != 401 {
				t.Fatalf("readonly push: %d %s", w.Code, w.Body.String())
			}
		}
	}
	no := users["none"]
	noWeb, _ := identity.Issue(&no, "web", nil, time.Hour)
	for _, route := range []struct{ method, path string }{{"POST", "/api/deployments"}, {"GET", "/api/registries"}, {"POST", "/api/images/load"}, {"POST", "/api/bundles/import-directory"}, {"GET", "/api/system/info"}} {
		if w := req(route.method, route.path, noWeb, []byte(`{}`)); w.Code != 403 {
			t.Fatalf("ordinary privilege %s %s: %d", route.method, route.path, w.Code)
		}
	}
	ro := token("read_only", "repository:demo/app:pull,push,delete")
	for _, route := range []struct{ method, path string }{{"PATCH", "/v2/demo/app/blobs/uploads/fake"}, {"PUT", "/v2/demo/app/blobs/uploads/fake"}, {"PUT", "/v2/demo/app/manifests/v1"}, {"DELETE", "/v2/demo/app/blobs/sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}} {
		if w := req(route.method, route.path, ro, nil); w.Code != 401 {
			t.Fatalf("read only mutation %s: %d", route.path, w.Code)
		}
	}
	// Native protocol: upload a config blob, submit a manifest, HEAD and pull it.
	rw := token("read_write", "repository:demo/app:pull,push,delete")
	configBody := []byte(`{"architecture":"amd64","os":"linux","rootfs":{"type":"layers","diff_ids":[]},"config":{}}`)
	cfgDigest := digest.FromBytes(configBody)
	q := httptest.NewRequest("POST", "/v2/demo/app/blobs/uploads/?digest="+cfgDigest.String(), bytes.NewReader(configBody))
	q.Header.Set("Authorization", "Bearer "+rw)
	q.Header.Set("Content-Type", "application/octet-stream")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, q)
	if w.Code == 202 {
		location, _ := url.Parse(w.Header().Get("Location"))
		values := location.Query()
		values.Set("digest", cfgDigest.String())
		location.RawQuery = values.Encode()
		q = httptest.NewRequest("PUT", location.RequestURI(), bytes.NewReader(configBody))
		q.Header.Set("Authorization", "Bearer "+rw)
		q.Header.Set("Content-Type", "application/octet-stream")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, q)
	}
	if w.Code != 201 {
		t.Fatalf("blob upload completion: %d %s", w.Code, w.Body.String())
	}
	manifest := []byte(fmt.Sprintf(`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{"mediaType":"application/vnd.oci.image.config.v1+json","digest":%q,"size":%d},"layers":[]}`, cfgDigest, len(configBody)))
	q = httptest.NewRequest("PUT", "/v2/demo/app/manifests/v1", bytes.NewReader(manifest))
	q.Header.Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
	q.Header.Set("Authorization", "Bearer "+rw)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, q)
	if w.Code != 201 {
		t.Fatalf("manifest push: %d %s", w.Code, w.Body.String())
	}
	d := digest.FromBytes(manifest).String()
	reader := users["read_only"]
	readerWeb, _ := identity.Issue(&reader, "web", nil, time.Hour)
	for _, path := range []string{"/api/builtin/catalog", "/api/builtin/tags?name=demo%2Fapp", "/api/builtin/tags?name=demo%2Fapp&reference=v1", "/api/builtin/tags?name=demo%2Fapp&reference=" + d} {
		if w := req("GET", path, readerWeb, nil); w.Code != 200 {
			t.Fatalf("builtin browser %s: %d %s", path, w.Code, w.Body.String())
		}
	}
	for _, path := range []string{"/api/builtin/tags", "/api/builtin/tags?name=../app", "/api/builtin/tags?name=demo/app&reference=bad/ref"} {
		if w := req("GET", path, readerWeb, nil); w.Code != 400 || !strings.Contains(w.Body.String(), `"error"`) {
			t.Fatalf("invalid builtin query %s: %d %s", path, w.Code, w.Body.String())
		}
	}
	if w := req("GET", "/api/builtin/tags?name=missing/app", readerWeb, nil); w.Code != 404 || !strings.Contains(w.Body.String(), `"error"`) {
		t.Fatalf("missing builtin repository: %d %s", w.Code, w.Body.String())
	}
	if w := req("HEAD", "/v2/demo/app/manifests/v1", rw, nil); w.Code != 200 || w.Header().Get("Docker-Content-Digest") != d {
		t.Fatalf("manifest HEAD: %d %s", w.Code, w.Body.String())
	}
	for _, path := range []string{"/v2/demo/app/tags/list", "/v2/demo/app/manifests/v1", "/v2/demo/app/blobs/" + cfgDigest.String()} {
		if w := req("GET", path, ro, nil); w.Code != 200 {
			t.Fatalf("read only read %s: %d %s", path, w.Code, w.Body.String())
		}
	}
	if w := req("GET", "/v2/another/app/manifests/v1", ro, nil); w.Code != 401 {
		t.Fatalf("cross repository token accepted: %d", w.Code)
	}
	var events int64
	database.DB.Model(&models.ImageEvent{}).Where("committed = ?", true).Count(&events)
	if events != 1 {
		t.Fatalf("expected one committed manifest event, got %d", events)
	}
	admin := token("admin", "repository:demo/app:pull,delete")
	database.DB.Create(&models.Release{Images: `[{"digest":"` + d + `"}]`, Retained: true})
	if w := req("DELETE", "/v2/demo/app/manifests/"+d, admin, nil); w.Code != 409 {
		t.Fatalf("protected delete %d %s", w.Code, w.Body.String())
	}
	adminUser := users["admin"]
	adminWeb, _ := identity.Issue(&adminUser, "web", nil, time.Hour)
	deleteBody := []byte(`{"name":"demo/app","digest":"` + d + `"}`)
	if w := req("DELETE", "/api/builtin/manifests", readerWeb, deleteBody); w.Code != 403 {
		t.Fatalf("reader builtin delete: %d %s", w.Code, w.Body.String())
	}
	if w := req("DELETE", "/api/builtin/manifests", adminWeb, []byte(`{"name":"demo/app","digest":"invalid"}`)); w.Code != 400 {
		t.Fatalf("invalid builtin digest: %d %s", w.Code, w.Body.String())
	}
	if w := req("DELETE", "/api/builtin/manifests", adminWeb, deleteBody); w.Code != 409 || !strings.Contains(w.Body.String(), `"error"`) {
		t.Fatalf("protected builtin delete: %d %s", w.Code, w.Body.String())
	}
	builtin.SetReadOnly(true)
	maintenanceDelete := req("DELETE", "/api/builtin/manifests", adminWeb, deleteBody)
	maintenanceCatalog := req("GET", "/api/builtin/catalog", readerWeb, nil)
	builtin.SetReadOnly(false)
	if maintenanceDelete.Code != 503 || maintenanceCatalog.Code != 200 {
		t.Fatalf("maintenance: delete=%d catalog=%d", maintenanceDelete.Code, maintenanceCatalog.Code)
	}
	// Every subsequent request rechecks authorization version and enabled state.
	u := users["read_write"]
	database.DB.Model(&u).Update("auth_version", u.AuthVersion+1)
	if w := req("HEAD", "/v2/demo/app/manifests/v1", rw, nil); w.Code != 401 {
		t.Fatalf("revoked token: %d", w.Code)
	}
	// Last admin guard and self-grant prevention at the API boundary.
	a := users["admin"]
	web, _ := identity.Issue(&a, "web", nil, time.Hour)
	if w := req("PATCH", "/api/users/1", web, []byte(`{"enabled":false}`)); w.Code != 409 {
		t.Fatalf("last admin: %d", w.Code)
	}
	n := users["none"]
	nw, _ := identity.Issue(&n, "web", nil, time.Hour)
	if w := req("PATCH", "/api/users/1", nw, []byte(`{"is_admin":true}`)); w.Code != 403 {
		t.Fatalf("self grant %d", w.Code)
	}
	if _, err := os.Stat(filepath.Join(config.Load().DataDir, "registry", "docker", "registry", "v2")); err != nil {
		t.Fatal(err)
	}
	database.DB.Model(&models.Release{}).Where("retained = ?", true).Update("retained", false)
	if w := req("DELETE", "/api/builtin/manifests", adminWeb, deleteBody); w.Code != 200 {
		t.Fatalf("unprotected builtin delete: %d %s", w.Code, w.Body.String())
	}
}
