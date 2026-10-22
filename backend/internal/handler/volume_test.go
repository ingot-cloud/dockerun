package handler

import (
	"dockerun/internal/docker"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
)

func TestVolumeWorkflow(t *testing.T) {
	var creations []volume.CreateOptions
	transport := containerTransport(func(request *http.Request) (*http.Response, error) {
		response := httptest.NewRecorder()
		response.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == "POST" && strings.HasSuffix(request.URL.Path, "/volumes/create"):
			var options volume.CreateOptions
			if err := json.NewDecoder(request.Body).Decode(&options); err != nil {
				t.Fatal(err)
			}
			creations = append(creations, options)
			name := options.Name
			if name == "" {
				name = "generated-volume"
			}
			response.WriteHeader(201)
			json.NewEncoder(response).Encode(volume.Volume{Name: name, Driver: options.Driver, Options: options.DriverOpts, Labels: options.Labels})
		case request.Method == "DELETE":
			if force := request.URL.Query().Get("force"); force == "1" || force == "true" {
				t.Error("normal removal must not force deletion")
			}
			if strings.HasSuffix(request.URL.Path, "/busy") {
				response.WriteHeader(409)
				io.WriteString(response, `{"message":"volume is in use by stopped-container"}`)
			} else {
				response.WriteHeader(204)
			}
		case strings.HasSuffix(request.URL.Path, "/volumes"):
			io.WriteString(response, `{"Volumes":[{"Name":"test-data","Driver":"local","Mountpoint":"/data/test-data","Labels":{"owner":"test"}}],"Warnings":["driver notice"]}`)
		case strings.HasSuffix(request.URL.Path, "/test-data"):
			io.WriteString(response, `{"Name":"test-data","Driver":"local","Mountpoint":"/data/test-data","Options":{"type":"nfs4","device":":/export/data"}}`)
		case strings.HasSuffix(request.URL.Path, "/unavailable"):
			response.WriteHeader(500)
			io.WriteString(response, `{"message":"volume driver unavailable"}`)
		default:
			response.WriteHeader(404)
			io.WriteString(response, `{"message":"no such volume"}`)
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
	router.GET("/volumes", ListVolumes)
	router.GET("/volumes/:name", InspectVolume)
	router.POST("/volumes", CreateVolume)
	router.DELETE("/volumes/:name", RemoveVolume)
	perform := func(method, path, body string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(method, path, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(response, request)
		return response
	}
	response := perform("POST", "/volumes", `{"name":" test-data ","driver":"local","options":{"type":"nfs4","device":":/export/data","o":"addr=192.0.2.1,rw"},"labels":{"owner":"test"}}`)
	if response.Code != 200 || len(creations) != 1 {
		t.Fatalf("create: %d %s", response.Code, response.Body.String())
	}
	created := creations[0]
	if created.Name != "test-data" || created.Driver != "local" || created.DriverOpts["type"] != "nfs4" || created.DriverOpts["device"] != ":/export/data" || created.DriverOpts["o"] != "addr=192.0.2.1,rw" || created.Labels["owner"] != "test" {
		t.Fatalf("driver options or labels lost: %+v", created)
	}
	response = perform("POST", "/volumes", `{}`)
	if response.Code != 200 || !strings.Contains(response.Body.String(), "generated-volume") || creations[1].Name != "" || creations[1].Driver != "local" {
		t.Fatalf("automatic name/default driver: %d %s", response.Code, response.Body.String())
	}
	for _, test := range []struct {
		method, path, body, expected string
		status                       int
	}{
		{"GET", "/volumes", "", "driver notice", 200},
		{"GET", "/volumes/test-data", "", ":/export/data", 200},
		{"GET", "/volumes/missing", "", "卷不存在", 404},
		{"GET", "/volumes/unavailable", "", "driver unavailable", 500},
		{"DELETE", "/volumes/test-data?force=false", "", "卷删除成功", 200},
		{"DELETE", "/volumes/busy?force=false", "", "in use", 500},
		{"POST", "/volumes", "invalid-json", "请求参数错误", 400},
	} {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			response := perform(test.method, test.path, test.body)
			if response.Code != test.status || !strings.Contains(response.Body.String(), test.expected) {
				t.Fatalf("status %d: %s", response.Code, response.Body.String())
			}
		})
	}
}
