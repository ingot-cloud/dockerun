package release

import (
	"context"
	"crypto/tls"
	"dockerun/internal/builtin"
	"dockerun/internal/config"
	"dockerun/internal/models"
	"dockerun/internal/utils"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/distribution/reference"
	digest "github.com/opencontainers/go-digest"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const AcceptManifest = "application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json"

// Connection pools are shared, while authentication remains local to each request.
var registryHTTPClients = [2]*http.Client{newRegistryHTTPClient(false), newRegistryHTTPClient(true)}

func newRegistryHTTPClient(insecure bool) *http.Client {
	return &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		DialContext:         (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		TLSClientConfig:     &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: insecure},
		TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: 15 * time.Second,
		IdleConnTimeout: 90 * time.Second, MaxIdleConns: 100, MaxIdleConnsPerHost: 8,
		MaxConnsPerHost: 8, ForceAttemptHTTP2: true,
	}}
}

func RegistryRequest(ctx context.Context, reg models.Registry, method, path string) ([]byte, http.Header, error) {
	for attempt := 0; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		body, headers, err := registryRequest(ctx, reg, method, path)
		if err == nil || attempt >= 2 || reg.Kind == "builtin" || (method != http.MethodGet && method != http.MethodHead) || !retryableRegistryError(ctx, err) {
			return body, headers, err
		}
		timer := time.NewTimer(time.Duration(attempt+1) * 150 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func retryableRegistryError(ctx context.Context, err error) bool {
	if ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.ECONNABORTED) || errors.Is(err, syscall.EPIPE) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	var networkError net.Error
	return errors.As(err, &networkError) && networkError.Timeout()
}

func registryRequest(ctx context.Context, reg models.Registry, method, path string) ([]byte, http.Header, error) {
	var token string
	if reg.Kind == "builtin" {
		repo := ""
		p := strings.TrimPrefix(path, "/v2/")
		if i := strings.Index(p, "/manifests/"); i >= 0 {
			repo = p[:i]
		} else if i := strings.Index(p, "/tags/"); i >= 0 {
			repo = p[:i]
		}
		var err error
		token, err = builtin.InternalToken([]string{repo})
		if err != nil {
			return nil, nil, err
		}
		req := httptest.NewRequest(method, path, nil).WithContext(ctx)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", AcceptManifest)
		w := httptest.NewRecorder()
		builtin.Serve(w, req)
		if w.Code < 200 || w.Code >= 300 {
			return nil, nil, fmt.Errorf("registry returned %d: %s", w.Code, w.Body.String())
		}
		return w.Body.Bytes(), w.Header(), nil
	}
	base, err := url.Parse(reg.URL)
	if err != nil || base.Host == "" || (base.Scheme != "http" && base.Scheme != "https") {
		return nil, nil, fmt.Errorf("Registry URL 无效")
	}
	client := registryHTTPClients[0]
	if reg.Insecure {
		client = registryHTTPClients[1]
	}
	target := strings.TrimSuffix(reg.URL, "/") + path
	send := func(bearer string, basic bool) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, method, target, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", AcceptManifest)
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		if basic {
			p := reg.Password
			if v, e := utils.OpenCredential(p); e == nil {
				p = v
			}
			req.SetBasicAuth(reg.Username, p)
		}
		return client.Do(req)
	}
	resp, err := send("", false)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode == 401 {
		challenges := responseChallenges(resp)
		io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
		resp.Body.Close()
		authenticated := false
		for _, ch := range challenges {
			if strings.EqualFold(ch.Scheme, "basic") {
				authenticated = true
				resp, err = send("", true)
				break
			}
			if !strings.EqualFold(ch.Scheme, "bearer") {
				continue
			}
			realm, e := url.Parse(ch.Parameters["realm"])
			if e != nil || realm.Host == "" || (realm.Scheme != "https" && !(reg.Insecure && realm.Scheme == "http")) {
				return nil, nil, fmt.Errorf("Registry 返回不安全的令牌地址")
			}
			q := realm.Query()
			q.Set("service", ch.Parameters["service"])
			q.Set("scope", ch.Parameters["scope"])
			realm.RawQuery = q.Encode()
			req, _ := http.NewRequestWithContext(ctx, "GET", realm.String(), nil)
			p := reg.Password
			if v, e := utils.OpenCredential(p); e == nil {
				p = v
			}
			if reg.Username != "" {
				req.SetBasicAuth(reg.Username, p)
			}
			tr, e := client.Do(req)
			if e != nil {
				return nil, nil, e
			}
			var t struct {
				Token       string `json:"token"`
				AccessToken string `json:"access_token"`
			}
			e = json.NewDecoder(io.LimitReader(tr.Body, 1<<20)).Decode(&t)
			tr.Body.Close()
			if e != nil || tr.StatusCode != 200 {
				return nil, nil, fmt.Errorf("Registry 认证失败")
			}
			if t.Token == "" {
				t.Token = t.AccessToken
			}
			if t.Token == "" {
				return nil, nil, fmt.Errorf("Registry 认证失败：令牌为空")
			}
			authenticated = true
			resp, err = send(t.Token, false)
			break
		}
		if !authenticated {
			return nil, nil, fmt.Errorf("Registry HTTP 401：未提供支持的认证方式")
		}
	}
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("Registry HTTP %d: %s", resp.StatusCode, string(b))
	}
	return b, resp.Header, nil
}
func Resolve(ctx context.Context, reg models.Registry, ref string) (Image, error) {
	n, err := reference.ParseNormalizedNamed(ref)
	if err != nil {
		return Image{}, err
	}
	registryURL := reg.URL
	if reg.Kind == "builtin" {
		registryURL = config.Load().PublicURL
	}
	u, err := url.Parse(registryURL)
	if err != nil || u == nil || u.Host == "" {
		return Image{}, fmt.Errorf("Registry 地址无效")
	}
	if reference.Domain(n) != u.Host && !(reference.Domain(n) == "docker.io" && u.Host == "registry-1.docker.io") {
		return Image{}, fmt.Errorf("镜像 %s 必须包含所选仓库地址 %s", ref, u.Host)
	}
	name := reference.Path(n)
	tag := "latest"
	if t, ok := n.(reference.Tagged); ok {
		tag = t.Tag()
	}
	if d, ok := n.(reference.Digested); ok {
		tag = d.Digest().String()
	}
	b, h, err := RegistryRequest(ctx, reg, "GET", "/v2/"+name+"/manifests/"+tag)
	if err != nil {
		return Image{}, err
	}
	d := digest.FromBytes(b).String()
	if h.Get("Docker-Content-Digest") != "" && h.Get("Docker-Content-Digest") != d {
		return Image{}, fmt.Errorf("manifest digest 不匹配")
	}
	return Image{Reference: ref, Digest: d}, nil
}

type authChallenge struct {
	Scheme     string
	Parameters map[string]string
}

var challengeFields = regexp.MustCompile(`([a-zA-Z_]+)=("(?:[^"\\]|\\.)*"|[^, ]+)`)

func responseChallenges(resp *http.Response) []authChallenge {
	out := []authChallenge{}
	for _, h := range resp.Header.Values("WWW-Authenticate") {
		scheme, rest, ok := strings.Cut(h, " ")
		if !ok {
			continue
		}
		c := authChallenge{Scheme: scheme, Parameters: map[string]string{}}
		for _, m := range challengeFields.FindAllStringSubmatch(rest, -1) {
			v := m[2]
			if strings.HasPrefix(v, "\"") {
				v, _ = strconv.Unquote(v)
			}
			c.Parameters[strings.ToLower(m[1])] = v
		}
		out = append(out, c)
	}
	return out
}
