package handler

import (
	"bytes"
	"dockerun/internal/docker"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/gin-gonic/gin"
)

type containerTransport func(*http.Request) (*http.Response, error)

func (transport containerTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

type trackedContainerBody struct {
	io.Reader
	closed bool
}

func (body *trackedContainerBody) Close() error { body.closed = true; return nil }

func TestContainerQueries(t *testing.T) {
	var framed bytes.Buffer
	stdcopy.NewStdWriter(&framed, stdcopy.Stdout).Write([]byte("服务已启动\n"))
	stdcopy.NewStdWriter(&framed, stdcopy.Stderr).Write([]byte("警告信息\n"))
	var lastBody *trackedContainerBody
	transport := containerTransport(func(request *http.Request) (*http.Response, error) {
		response := httptest.NewRecorder()
		response.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(request.URL.Path, "/json"):
			if strings.Contains(request.URL.Path, "/tty/") {
				response.Write([]byte(`{"Id":"tty","Config":{"Tty":true}}`))
			} else {
				response.Write([]byte(`{"Id":"framed","Config":{"Tty":false}}`))
			}
		case strings.HasSuffix(request.URL.Path, "/logs"):
			if request.URL.Query().Get("tail") != "500" {
				t.Errorf("unexpected tail: %s", request.URL.RawQuery)
			}
			if strings.Contains(request.URL.Path, "/tty/") {
				response.Write([]byte("终端日志\n"))
			} else {
				response.Write(framed.Bytes())
			}
		case strings.HasSuffix(request.URL.Path, "/stats"):
			if request.URL.Query().Get("stream") != "0" {
				t.Error("stats must be a single snapshot")
			}
			if strings.Contains(request.URL.Path, "/invalid/") {
				response.Write([]byte(`invalid`))
			} else {
				response.Write([]byte(`{"read":"2026-10-21T10:00:00Z","cpu_stats":{"cpu_usage":{"total_usage":300},"system_cpu_usage":1000,"online_cpus":4},"memory_stats":{"usage":1000,"limit":2000},"networks":{"eth0":{"rx_bytes":100,"tx_bytes":200}}}`))
			}
		default:
			response.WriteHeader(404)
		}
		result := response.Result()
		lastBody = &trackedContainerBody{Reader: result.Body}
		result.Body = lastBody
		return result, nil
	})
	cli, err := client.NewClientWithOpts(client.WithHost("http://docker.test"), client.WithVersion("1.41"), client.WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatal(err)
	}
	previous := docker.Client
	docker.Client = cli
	t.Cleanup(func() { docker.Client = previous; cli.Close() })
	router := gin.New()
	router.GET("/containers/:id/logs", GetContainerLogs)
	router.GET("/containers/:id/stats", GetContainerStats)
	for _, test := range []struct {
		name, endpoint, expected string
		status                   int
	}{
		{"multiplexed logs have no binary headers", "/containers/framed/logs?tail=500", "服务已启动\n警告信息\n", 200},
		{"TTY logs remain plain text", "/containers/tty/logs?tail=500", "终端日志\n", 200},
		{"snapshot exposes decoded statistics", "/containers/stats/stats", "", 200},
		{"malformed snapshot is an error", "/containers/invalid/stats", "", 500},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest("GET", test.endpoint, nil))
			if response.Code != test.status {
				t.Fatalf("status %d: %s", response.Code, response.Body.String())
			}
			if test.expected != "" && response.Body.String() != test.expected {
				t.Fatalf("body %q", response.Body.String())
			}
			if !lastBody.closed {
				t.Error("Docker response body was not closed")
			}
			if test.name == "snapshot exposes decoded statistics" {
				var snapshot struct {
					CPU struct {
						Usage struct {
							Total uint64 `json:"total_usage"`
						} `json:"cpu_usage"`
					} `json:"cpu_stats"`
					Memory struct {
						Usage uint64 `json:"usage"`
					} `json:"memory_stats"`
				}
				if err := json.Unmarshal(response.Body.Bytes(), &snapshot); err != nil {
					t.Fatal(err)
				}
				if snapshot.CPU.Usage.Total != 300 || snapshot.Memory.Usage != 1000 {
					t.Fatalf("missing CPU/memory data: %s", response.Body.String())
				}
			}
		})
	}
}
