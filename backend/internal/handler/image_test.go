package handler

import (
	"dockerun/internal/docker"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
)

type imageExportTransport func(*http.Request) (*http.Response, error)

func (transport imageExportTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestExportImages(t *testing.T) {
	var saved []string
	daemon := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/json") {
			w.Header().Set("Content-Type", "application/json")
			switch {
			case strings.Contains(r.URL.Path, "sha256:a"):
				w.Write([]byte(`{"Id":"sha256:a","RepoTags":["app:1","app:latest","app:1"]}`))
			case strings.Contains(r.URL.Path, "sha256:b"):
				w.Write([]byte(`{"Id":"sha256:b","RepoTags":["<none>:<none>"]}`))
			default:
				w.WriteHeader(404)
				w.Write([]byte(`{"message":"No such image"}`))
			}
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/images/get") {
			t.Errorf("unexpected Docker path: %s", r.URL.Path)
			w.WriteHeader(404)
			return
		}
		saved = r.URL.Query()["names"]
		if len(saved) == 1 && saved[0] == "missing" {
			w.WriteHeader(404)
			w.Write([]byte(`{"message":"No such image"}`))
			return
		}
		w.Header().Set("Content-Type", "application/x-tar")
		w.Write([]byte("test-image-archive"))
	})
	httpClient := &http.Client{Transport: imageExportTransport(func(request *http.Request) (*http.Response, error) {
		response := httptest.NewRecorder()
		daemon.ServeHTTP(response, request)
		return response.Result(), nil
	})}
	cli, err := client.NewClientWithOpts(client.WithHost("http://docker.test"), client.WithVersion("1.41"), client.WithHTTPClient(httpClient))
	if err != nil {
		t.Fatal(err)
	}
	previous := docker.Client
	docker.Client = cli
	t.Cleanup(func() { docker.Client = previous; cli.Close() })
	r := gin.New()
	r.POST("/images/export", ExportImages)
	r.GET("/images/:id/export", ExportImage)
	for _, test := range []struct {
		name, method, path, body string
		status                   int
		ids                      []string
	}{
		{"multiple preserves tags and untagged images", "POST", "/images/export", `{"image_ids":["sha256:a","sha256:b","sha256:a"]}`, 200, []string{"app:1", "app:latest", "sha256:b"}},
		{"single preserves every tag", "GET", "/images/sha256:a/export", "", 200, []string{"app:1", "app:latest"}},
		{"empty selection", "POST", "/images/export", `{"image_ids":[]}`, 400, nil},
		{"missing selection", "POST", "/images/export", `{}`, 400, nil},
		{"empty id", "POST", "/images/export", `{"image_ids":[" "]}`, 400, nil},
		{"daemon error", "POST", "/images/export", `{"image_ids":["missing"]}`, 500, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			saved = nil
			request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			r.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status %d: %s", response.Code, response.Body.String())
			}
			if !reflect.DeepEqual(saved, test.ids) {
				t.Fatalf("exported %v; want %v", saved, test.ids)
			}
			if test.status == 200 && (response.Body.String() != "test-image-archive" || response.Header().Get("Content-Type") != "application/x-tar" || !strings.Contains(response.Header().Get("Content-Disposition"), "attachment")) {
				t.Fatal("missing archive or download headers")
			}
		})
	}
}

func TestGetImageHistory(t *testing.T) {
	transport := imageExportTransport(func(request *http.Request) (*http.Response, error) {
		response := httptest.NewRecorder()
		response.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/v1.41/images/sha256:test/history":
			response.Write([]byte(`[{"Id":"sha256:test","Created":10,"CreatedBy":"/bin/sh -c echo hello","Size":123},{"Id":"<missing>","Created":1,"CreatedBy":"CMD nginx","Size":0}]`))
		case "/v1.41/images/empty/history":
			response.Write([]byte(`[]`))
		case "/v1.41/images/missing/history":
			response.WriteHeader(404)
			response.Write([]byte(`{"message":"No such image"}`))
		default:
			response.WriteHeader(500)
			response.Write([]byte(`{"message":"daemon unavailable"}`))
		}
		return response.Result(), nil
	})
	cli, err := client.NewClientWithOpts(client.WithHost("http://docker.test"), client.WithVersion("1.41"), client.WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatal(err)
	}
	previous := docker.Client
	docker.Client = cli
	t.Cleanup(func() { docker.Client = previous; cli.Close() })
	router := gin.New()
	router.GET("/images/:id/history", GetImageHistory)
	for _, test := range []struct {
		id     string
		status int
		body   string
	}{
		{"sha256:test", 200, `"CreatedBy":"/bin/sh -c echo hello"`},
		{"empty", 200, `[]`},
		{"missing", 404, "镜像不存在或已被删除"},
		{"unavailable", 500, "无法读取镜像层历史"},
	} {
		t.Run(test.id, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest("GET", "/images/"+test.id+"/history", nil))
			if response.Code != test.status || !strings.Contains(response.Body.String(), test.body) {
				t.Fatalf("%d %s", response.Code, response.Body.String())
			}
		})
	}
}
