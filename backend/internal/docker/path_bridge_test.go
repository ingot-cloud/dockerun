package docker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
)

func TestTranslateCreateOnlyChangesMountSources(t *testing.T) {
	root := t.TempDir()
	p := dataPaths{local: root, host: "/host/dockerun"}
	body := `{"Image":"fixture","Env":["PATH_VALUE=` + root + `"],"Cmd":["echo","` + root + `"],"HostConfig":{"NanoCpus":9007199254740993,"Binds":["` + root + `/jobs/1:/config:ro,z","business-data:/business","/opt/business:/opt/business"],"Mounts":[{"Type":"bind","Source":"` + root + `/jobs/1","Target":"` + root + `","ReadOnly":true},{"Type":"volume","Source":"business-data","Target":"/data"}]}}`
	out, err := translateCreate([]byte(body), false, p)
	if err != nil {
		t.Fatal(err)
	}
	doc, _ := decodeDocument(out)
	host := doc["HostConfig"].(map[string]any)
	binds := host["Binds"].([]any)
	if binds[0] != "/host/dockerun/jobs/1:/config:ro,z" || binds[1] != "business-data:/business" || binds[2] != "/opt/business:/opt/business" {
		t.Fatal("绑定源转换错误", string(out))
	}
	bind := host["Mounts"].([]any)[0].(map[string]any)
	if bind["Source"] != "/host/dockerun/jobs/1" || bind["Target"] != root || bind["ReadOnly"] != true {
		t.Fatal("改动了绑定目标或选项", string(out))
	}
	if !bytes.Contains(out, []byte(`9007199254740993`)) || doc["Cmd"].([]any)[1] != root || doc["Env"].([]any)[0] != "PATH_VALUE="+root {
		t.Fatal("改动了命令、环境变量或整数精度", string(out))
	}
}

func TestTranslateLocalDriverBindAndInspectView(t *testing.T) {
	root := t.TempDir()
	p := dataPaths{local: root, host: "/host/dockerun"}
	out, err := translateCreate([]byte(`{"Name":"business","Driver":"local","DriverOpts":{"type":"none","o":"ro,bind","device":"`+root+`/jobs/1"}}`), true, p)
	if err != nil || !bytes.Contains(out, []byte(`"device":"/host/dockerun/jobs/1"`)) {
		t.Fatalf("卷绑定路径未转换: %s %v", out, err)
	}
	for _, path := range []string{"/v1.44/containers/test/json", "/v1.44/containers/json"} {
		body := `{"Mounts":[{"Type":"bind","Source":"/host/dockerun/jobs/1","Destination":"/config"}],"HostConfig":{"Binds":["/host/dockerun/jobs/1:/config:ro"]}}`
		if strings.HasSuffix(path, "/containers/json") {
			body = "[" + body + "]"
		}
		r := &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body)), Request: httptest.NewRequest("GET", path, nil)}
		if err := translateInspectResponse(r, p); err != nil {
			t.Fatal(err)
		}
		content, _ := io.ReadAll(r.Body)
		if bytes.Contains(content, []byte("/host/dockerun")) || !bytes.Contains(content, []byte(root+"/jobs/1")) {
			t.Fatalf("Compose 检查结果未还原到容器视图: %s", content)
		}
	}
}

func setupBridge(t *testing.T, volume bool, handler http.HandlerFunc) (string, string, *http.Client) {
	t.Helper()
	// Darwin Unix socket 有路径长度限制，使用短临时目录。
	root, err := os.MkdirTemp("/tmp", "du-path-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(root) })
	host := "/engine/data/root"
	engine := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/_ping":
			w.Header().Set("API-Version", "1.44")
		case strings.HasSuffix(r.URL.Path, "/volumes/dockerun-data"):
			json.NewEncoder(w).Encode(map[string]any{"Name": "dockerun-data", "Driver": "local", "Mountpoint": host})
		default:
			handler(w, r)
		}
	}))
	t.Cleanup(engine.Close)
	t.Setenv("DATA_DIR", root)
	t.Setenv("HOST_DATA_DIR", host)
	t.Setenv("DATA_VOLUME", "")
	t.Setenv("DOCKER_HOST", "tcp://"+strings.TrimPrefix(engine.URL, "http://"))
	t.Setenv("DOCKER_TLS_VERIFY", "")
	t.Setenv("DOCKER_CERT_PATH", "")
	if volume {
		t.Setenv("HOST_DATA_DIR", "")
		t.Setenv("DATA_VOLUME", "dockerun-data")
	}
	previous := Client
	if err := Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { Close(); Client = previous })
	env := CommandEnvironment([]string{"DOCKER_CONTEXT=ignored", "DOCKER_TLS_VERIFY=1", "DOCKER_HOST=ignored", "CUSTOM=value"})
	if len(env) != 2 || env[0] != "CUSTOM=value" || !strings.HasPrefix(env[1], "DOCKER_HOST=unix://") {
		t.Fatal("Docker 命令环境未切换到路径代理", env)
	}
	socket := strings.TrimPrefix(env[1], "DOCKER_HOST=unix://")
	httpClient := &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}}
	t.Cleanup(httpClient.CloseIdleConnections)
	return root, host, httpClient
}

func TestBridgeNamedVolumeAndBindSupportSDKAndCLI(t *testing.T) {
	for _, volume := range []bool{false, true} {
		t.Run(map[bool]string{false: "bind", true: "volume"}[volume], func(t *testing.T) {
			creates := make(chan map[string]any, 2)
			root, host, cli := setupBridge(t, volume, func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "/containers/create") {
					content, _ := io.ReadAll(r.Body)
					doc, err := decodeDocument(content)
					if err != nil {
						t.Error(err)
					}
					creates <- doc
					w.WriteHeader(201)
					io.WriteString(w, `{"Id":"fixture","Warnings":[]}`)
					return
				}
				io.WriteString(w, `{"Id":"fixture","Mounts":[{"Type":"bind","Source":"/engine/data/root/jobs/1/configs/app.conf","Destination":"/app.conf"}]}`)
			})
			file := filepath.Join(root, "jobs/1/configs/app.conf")
			os.MkdirAll(filepath.Dir(file), 0700)
			os.WriteFile(file, []byte("fixture"), 0600)
			_, err := Client.ContainerCreate(context.Background(), &container.Config{Image: "fixture"}, &container.HostConfig{Mounts: []mount.Mount{{Type: mount.TypeBind, Source: file, Target: "/app.conf", ReadOnly: true}}}, nil, nil, "fixture")
			if err != nil {
				t.Fatal(err)
			}
			body := `{"Image":"fixture","HostConfig":{"Binds":["` + file + `:/app.conf:ro"]}}`
			resp, err := cli.Post("http://docker/v1.44/containers/create", "application/json", strings.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if resp.StatusCode != 201 {
				t.Fatal("CLI 请求创建失败", resp.StatusCode)
			}
			sdk := (<-creates)["HostConfig"].(map[string]any)
			if sdk["Mounts"].([]any)[0].(map[string]any)["Source"] != host+"/jobs/1/configs/app.conf" {
				t.Fatal("SDK 挂载源错误", sdk)
			}
			if got := (<-creates)["HostConfig"].(map[string]any)["Binds"].([]any)[0]; got != host+"/jobs/1/configs/app.conf:/app.conf:ro" {
				t.Fatal("CLI 挂载源错误", got)
			}
			resp, err = cli.Get("http://docker/v1.44/containers/fixture/json")
			if err != nil {
				t.Fatal(err)
			}
			content, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if !bytes.Contains(content, []byte(file)) {
				t.Fatal("CLI Inspect 未还原路径", string(content))
			}
			if LocalDataPath(host+"/jobs/1") != root+"/jobs/1" {
				t.Fatal("旧目录未转换到固定容器目录")
			}
		})
	}
}

func TestBridgePreservesStreamingAndConnectionUpgrade(t *testing.T) {
	_, _, cli := setupBridge(t, false, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") != "" {
			conn, rw, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Error(err)
				return
			}
			defer conn.Close()
			rw.WriteString("HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: tcp\r\n\r\n")
			rw.Flush()
			line, _ := rw.ReadString('\n')
			rw.WriteString("echo:" + line)
			rw.Flush()
			return
		}
		w.Header().Set("Content-Type", "application/x-json-stream")
		io.WriteString(w, "first\n")
		w.(http.Flusher).Flush()
		// 第一帧应立即返回，而非等待整个响应完成。
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	})
	resp, err := cli.Get("http://docker/v1.44/images/create")
	if err != nil {
		t.Fatal(err)
	}
	line, err := bufio.NewReader(resp.Body).ReadString('\n')
	resp.Body.Close()
	if err != nil || line != "first\n" {
		t.Fatalf("流式输出损坏: %q %v", line, err)
	}
	req, _ := http.NewRequest("POST", "http://docker/v1.44/exec/fixture/start", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "tcp")
	resp, err = cli.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	stream, ok := resp.Body.(io.ReadWriteCloser)
	if resp.StatusCode != 101 || !ok {
		t.Fatal("Docker exec 连接升级失败", resp.StatusCode)
	}
	stream.Write([]byte("ping\n"))
	line, err = bufio.NewReader(stream).ReadString('\n')
	if err != nil || line != "echo:ping\n" {
		t.Fatalf("双向数据流损坏: %q %v", line, err)
	}
}

func TestStorageMappingStartupValidation(t *testing.T) {
	for _, tc := range []struct {
		name, host, volume, mountpoint, wantError string
	}{
		{name: "native"},
		{name: "same-path", host: "SAME"},
		{name: "conflicting", host: "/host/data", volume: "fixture", wantError: "不能同时设置"},
		{name: "relative-host", host: "relative/data", wantError: "绝对路径"},
		{name: "root-host", host: "/", wantError: "非根目录"},
		{name: "missing-volume", volume: "missing", wantError: "读取数据命名卷"},
		{name: "empty-mountpoint", volume: "fixture", wantError: "没有可用的宿主挂载路径"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if tc.host == "SAME" {
				tc.host = root
			}
			engine := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/_ping" {
					w.Header().Set("API-Version", "1.44")
					return
				}
				if strings.HasSuffix(r.URL.Path, "/volumes/fixture") {
					json.NewEncoder(w).Encode(map[string]any{"Name": "fixture", "Mountpoint": tc.mountpoint})
					return
				}
				w.WriteHeader(404)
				io.WriteString(w, `{"message":"volume not found"}`)
			}))
			defer engine.Close()
			t.Setenv("DATA_DIR", root)
			t.Setenv("HOST_DATA_DIR", tc.host)
			t.Setenv("DATA_VOLUME", tc.volume)
			t.Setenv("DOCKER_HOST", "tcp://"+strings.TrimPrefix(engine.URL, "http://"))
			t.Setenv("DOCKER_TLS_VERIFY", "")
			t.Setenv("DOCKER_CERT_PATH", "")
			previous := Client
			defer func() { Close(); Client = previous }()
			err := Init()
			if tc.wantError == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("启动验证结果错误: %v，预期 %s", err, tc.wantError)
			}
			if got := CommandEnvironment([]string{"DOCKER_HOST=original"}); len(got) != 1 || got[0] != "DOCKER_HOST=original" {
				t.Fatal("原生模式或启动失败不应留下路径代理", got)
			}
		})
	}
}
