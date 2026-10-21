package handler

import (
	"dockerun/internal/docker"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
)

func TestContainerOperations(t *testing.T) {
	var calls []*http.Request
	var restartName string
	transport := containerTransport(func(request *http.Request) (*http.Response, error) {
		calls = append(calls, request)
		response := httptest.NewRecorder()
		response.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(request.URL.Path, "/update"):
			var config struct{ RestartPolicy struct{ Name string } }
			json.NewDecoder(request.Body).Decode(&config)
			restartName = config.RestartPolicy.Name
			response.WriteString(`{"Warnings":[]}`)
		case strings.HasSuffix(request.URL.Path, "/commit"):
			response.WriteString(`{"Id":"sha256:committed"}`)
		case strings.HasSuffix(request.URL.Path, "/top"):
			response.WriteString(`{"Titles":["UID","PID","CMD"],"Processes":[["root","1","nginx"]]}`)
		default:
			response.WriteHeader(204)
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
	router.POST("/containers/:id/kill", KillContainer)
	router.POST("/containers/:id/pause", PauseContainer)
	router.POST("/containers/:id/resume", ResumeContainer)
	router.POST("/containers/:id/rename", RenameContainer)
	router.PUT("/containers/:id/restart-policy", UpdateContainerRestart)
	router.POST("/containers/:id/commit", CommitContainer)
	router.GET("/containers/:id/top", GetContainerProcesses)
	for _, test := range []struct {
		name, method, endpoint, body, dockerPath string
		status                                   int
	}{
		{"kill", "POST", "/containers/unit/kill", "", "/v1.41/containers/unit/kill", 200},
		{"pause", "POST", "/containers/unit/pause", "", "/v1.41/containers/unit/pause", 200},
		{"resume", "POST", "/containers/unit/resume", "", "/v1.41/containers/unit/unpause", 200},
		{"rename", "POST", "/containers/unit/rename", `{"name":" new-name "}`, "/v1.41/containers/unit/rename", 200},
		{"restart policy", "PUT", "/containers/unit/restart-policy", `{"name":"unless-stopped"}`, "/v1.41/containers/unit/update", 200},
		{"commit default tag", "POST", "/containers/unit/commit", `{"image":"my-app"}`, "/v1.41/commit", 200},
		{"process list", "GET", "/containers/unit/top", "", "/v1.41/containers/unit/top", 200},
		{"empty rename rejected", "POST", "/containers/unit/rename", `{"name":" "}`, "", 400},
		{"invalid restart policy", "PUT", "/containers/unit/restart-policy", `{"name":"invalid"}`, "", 400},
		{"negative retries", "PUT", "/containers/unit/restart-policy", `{"name":"on-failure","maximum_retry_count":-1}`, "", 400},
		{"retries only on failure", "PUT", "/containers/unit/restart-policy", `{"name":"always","maximum_retry_count":2}`, "", 400},
		{"invalid image reference", "POST", "/containers/unit/commit", `{"image":"BAD IMAGE"}`, "", 400},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls = nil
			response := httptest.NewRecorder()
			request := httptest.NewRequest(test.method, test.endpoint, strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status %d: %s", response.Code, response.Body.String())
			}
			if test.dockerPath == "" {
				if len(calls) != 0 {
					t.Fatal("invalid input reached Docker")
				}
				return
			}
			if len(calls) != 1 || calls[0].URL.Path != test.dockerPath {
				t.Fatalf("unexpected Docker calls: %v", calls)
			}
			query := calls[0].URL.Query()
			switch test.name {
			case "kill":
				if query.Get("signal") != "SIGKILL" {
					t.Fatal("wrong kill signal")
				}
			case "rename":
				if query.Get("name") != "new-name" {
					t.Fatal("name was not trimmed")
				}
			case "restart policy":
				if restartName != "unless-stopped" {
					t.Fatal("restart policy not forwarded")
				}
			case "commit default tag":
				if query.Get("repo") != "my-app" || query.Get("tag") != "latest" || query.Get("pause") == "0" {
					t.Fatal("commit options not preserved")
				}
			case "process list":
				if !strings.Contains(response.Body.String(), "nginx") {
					t.Fatal("missing process details")
				}
			}
		})
	}
}
