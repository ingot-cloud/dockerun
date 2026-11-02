package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/client"
)

type pathBridge struct {
	paths      dataPaths
	endpoint   string
	server     *http.Server
	transport  *http.Transport
	runtimeDir string
}

var bridgeMu sync.RWMutex
var bridge *pathBridge

// 保留 Compose 和 Shell 的容器文件视图，只在 Docker API 边界转换绑定源。
func startPathBridge(ctx context.Context) error {
	paths, err := resolveDataPaths(ctx)
	if err != nil {
		return err
	}
	if paths.local == paths.host {
		return nil
	}
	// socket 只用于本进程，避免共享目录（如 Docker Desktop）不支持 socket 权限。
	dir, err := os.MkdirTemp("/tmp", "dockerun-api-")
	if err != nil {
		return err
	}
	ready := false
	defer func() {
		if !ready {
			os.RemoveAll(dir)
		}
	}()
	socket := filepath.Join(dir, "docker.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		return err
	}
	if err = os.Chmod(socket, 0600); err != nil {
		listener.Close()
		return err
	}
	dial := Client.Dialer()
	transport := &http.Transport{
		DialContext:        func(ctx context.Context, _, _ string) (net.Conn, error) { return dial(ctx) },
		DisableCompression: true,
		IdleConnTimeout:    30 * time.Second,
	}
	proxy := &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			r.URL.Scheme = "http"
			r.URL.Host = client.DummyHost
			r.Host = client.DummyHost
		},
		Transport:      &mountTransport{base: transport, paths: *paths},
		FlushInterval:  -1,
		ModifyResponse: func(r *http.Response) error { return translateInspectResponse(r, *paths) },
	}
	server := &http.Server{Handler: proxy, ReadHeaderTimeout: 15 * time.Second}
	b := &pathBridge{paths: *paths, endpoint: "unix://" + socket, server: server, transport: transport, runtimeDir: dir}
	// SDK 也使用同一转换规则；保留原连接、TLS 与 API 版本协商。
	httpClient := Client.HTTPClient()
	httpClient.Transport = &mountTransport{base: httpClient.Transport, paths: *paths}
	if err := client.WithHTTPClient(httpClient)(Client); err != nil {
		listener.Close()
		return err
	}
	bridgeMu.Lock()
	bridge = b
	bridgeMu.Unlock()
	ready = true
	go server.Serve(listener)
	return nil
}

func CommandEnvironment(env []string) []string {
	bridgeMu.RLock()
	defer bridgeMu.RUnlock()
	if bridge == nil {
		return env
	}
	out := make([]string, 0, len(env)+1)
	for _, entry := range env {
		key, _, _ := strings.Cut(entry, "=")
		switch key {
		case "DOCKER_HOST", "DOCKER_CONTEXT", "DOCKER_TLS_VERIFY", "DOCKER_CERT_PATH":
			continue
		}
		out = append(out, entry)
	}
	return append(out, "DOCKER_HOST="+bridge.endpoint)
}

func closePathBridge() {
	bridgeMu.Lock()
	b := bridge
	bridge = nil
	bridgeMu.Unlock()
	if b != nil {
		b.server.Close()
		b.transport.CloseIdleConnections()
		os.RemoveAll(b.runtimeDir)
	}
}

type mountTransport struct {
	base  http.RoundTripper
	paths dataPaths
}

func (t *mountTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Body != nil && r.Body != http.NoBody && r.Method == http.MethodPost && (strings.HasSuffix(r.URL.Path, "/containers/create") || strings.HasSuffix(r.URL.Path, "/volumes/create")) {
		content, err := io.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			return nil, err
		}
		content, err = translateCreate(content, strings.HasSuffix(r.URL.Path, "/volumes/create"), t.paths)
		if err != nil {
			return nil, err
		}
		r = r.Clone(r.Context())
		r.Body = io.NopCloser(bytes.NewReader(content))
		r.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(content)), nil }
		r.ContentLength = int64(len(content))
		r.TransferEncoding = nil
		r.Header.Del("Transfer-Encoding")
		r.Header.Set("Content-Length", strconv.Itoa(len(content)))
	}
	return t.base.RoundTrip(r)
}

func decodeDocument(content []byte) (map[string]any, error) {
	var doc map[string]any
	d := json.NewDecoder(bytes.NewReader(content))
	d.UseNumber()
	err := d.Decode(&doc)
	return doc, err
}

func translateCreate(content []byte, volume bool, paths dataPaths) ([]byte, error) {
	doc, err := decodeDocument(content)
	if err != nil {
		return nil, err
	}
	if volume {
		err = translateVolumeOptions(doc, "DriverOpts", paths.hostPath)
	} else if host, ok := doc["HostConfig"].(map[string]any); ok {
		err = translateHostConfig(host, paths.hostPath)
	}
	if err != nil {
		return nil, err
	}
	return json.Marshal(doc)
}

func translateHostConfig(host map[string]any, path func(string) (string, error)) error {
	if binds, ok := host["Binds"].([]any); ok {
		for i, raw := range binds {
			bind, ok := raw.(string)
			if !ok {
				continue
			}
			source, rest, found := strings.Cut(bind, ":")
			if found && filepath.IsAbs(source) {
				mapped, err := path(source)
				if err != nil {
					return err
				}
				binds[i] = mapped + ":" + rest
			}
		}
	}
	return translateMounts(host["Mounts"], path)
}

func translateMounts(raw any, path func(string) (string, error)) error {
	if mounts, ok := raw.([]any); ok {
		for _, raw := range mounts {
			mount, ok := raw.(map[string]any)
			if !ok || mount["Type"] != "bind" {
				continue
			}
			if source, ok := mount["Source"].(string); ok {
				mapped, err := path(source)
				if err != nil {
					return err
				}
				mount["Source"] = mapped
			}
		}
	}
	return nil
}

func translateVolumeOptions(doc map[string]any, optionsKey string, path func(string) (string, error)) error {
	if driver, _ := doc["Driver"].(string); driver != "" && driver != "local" {
		return nil
	}
	options, ok := doc[optionsKey].(map[string]any)
	if !ok {
		return nil
	}
	flags, _ := options["o"].(string)
	for _, flag := range strings.Split(flags, ",") {
		if flag == "bind" || flag == "rbind" {
			if device, ok := options["device"].(string); ok && filepath.IsAbs(device) {
				mapped, err := path(device)
				if err != nil {
					return err
				}
				options["device"] = mapped
			}
			break
		}
	}
	return nil
}

// Compose 检查结果使用容器视图，避免路径不同导致未变更服务反复重建。
func translateInspectResponse(r *http.Response, paths dataPaths) error {
	if r.StatusCode != http.StatusOK || r.Request == nil || r.Request.Method != http.MethodGet {
		return nil
	}
	path := r.Request.URL.Path
	containerList := strings.HasSuffix(path, "/containers/json")
	containerInspect := strings.Contains(path, "/containers/") && strings.HasSuffix(path, "/json")
	volumeInspect := strings.Contains(path, "/volumes/") && !strings.HasSuffix(path, "/volumes/prune")
	if !containerList && !containerInspect && !volumeInspect {
		return nil
	}
	content, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		return err
	}
	var doc any
	d := json.NewDecoder(bytes.NewReader(content))
	d.UseNumber()
	if err = d.Decode(&doc); err != nil {
		return err
	}
	toLocal := func(path string) (string, error) { return paths.localPath(path), nil }
	items := []any{doc}
	if containerList {
		items, _ = doc.([]any)
	}
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if volumeInspect {
			translateVolumeOptions(item, "Options", toLocal)
			continue
		}
		translateMounts(item["Mounts"], toLocal)
		if host, ok := item["HostConfig"].(map[string]any); ok {
			translateHostConfig(host, toLocal)
		}
	}
	content, err = json.Marshal(doc)
	if err != nil {
		return err
	}
	r.Body = io.NopCloser(bytes.NewReader(content))
	r.ContentLength = int64(len(content))
	r.Header.Set("Content-Length", strconv.Itoa(len(content)))
	return nil
}
