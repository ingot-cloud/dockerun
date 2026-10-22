package release

import (
	"context"
	"dockerun/internal/models"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
)

func resetRegistryConnection(t *testing.T, w http.ResponseWriter) {
	t.Helper()
	connection, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		t.Error(err)
		return
	}
	if tcp, ok := connection.(*net.TCPConn); ok {
		tcp.SetLinger(0)
	}
	connection.Close()
}

func TestRegistryRequestReusesConnectionsAcrossAuthentication(t *testing.T) {
	var mutex sync.Mutex
	connections := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mutex.Lock()
		connections[r.RemoteAddr] = true
		mutex.Unlock()
		username, password, _ := r.BasicAuth()
		if username != "deploy" || password != "test-secret" {
			w.Header().Set("WWW-Authenticate", `Basic realm="registry"`)
			w.WriteHeader(401)
			fmt.Fprint(w, "authentication required")
			return
		}
		fmt.Fprint(w, `{}`)
	}))
	defer server.Close()
	registry := models.Registry{URL: server.URL, Username: "deploy", Password: "test-secret"}
	for i := 0; i < 2; i++ {
		if _, _, err := RegistryRequest(context.Background(), registry, "GET", "/v2/"); err != nil {
			t.Fatal(err)
		}
	}
	mutex.Lock()
	defer mutex.Unlock()
	if len(connections) != 1 {
		t.Fatalf("expected one reusable connection, got %d", len(connections))
	}
}

func TestRegistryRequestRetriesTransientReadFailures(t *testing.T) {
	for _, failure := range []string{"reset", "partial-body"} {
		t.Run(failure, func(t *testing.T) {
			var attempts atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if attempts.Add(1) == 1 {
					if failure == "reset" {
						resetRegistryConnection(t, w)
					} else {
						w.Header().Set("Content-Length", "100")
						fmt.Fprint(w, `{"tags":`)
					}
					return
				}
				fmt.Fprint(w, `{"tags":["latest"]}`)
			}))
			defer server.Close()
			body, _, err := RegistryRequest(context.Background(), models.Registry{URL: server.URL}, "GET", "/v2/team/app/tags/list")
			if err != nil || string(body) != `{"tags":["latest"]}` {
				t.Fatalf("read retry failed: %s %v", body, err)
			}
			if attempts.Load() != 2 {
				t.Fatalf("expected two attempts, got %d", attempts.Load())
			}
		})
	}
}

func TestRegistryRequestBoundsRetriesAndDoesNotRetryMutationsOrHTTPFailures(t *testing.T) {
	for _, scenario := range []struct {
		name, method string
		status, want int
	}{{"bounded-read", "GET", 0, 3}, {"mutation", "DELETE", 0, 1}, {"forbidden", "GET", 403, 1}, {"unauthorized", "GET", 401, 1}, {"server-error", "GET", 500, 1}} {
		t.Run(scenario.name, func(t *testing.T) {
			var attempts atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempts.Add(1)
				if scenario.status == 0 {
					resetRegistryConnection(t, w)
					return
				}
				w.WriteHeader(scenario.status)
				fmt.Fprint(w, `{"error":"unavailable"}`)
			}))
			defer server.Close()
			_, _, err := RegistryRequest(context.Background(), models.Registry{URL: server.URL}, scenario.method, "/v2/team/app/manifests/latest")
			if err == nil {
				t.Fatal("expected error")
			}
			if got := attempts.Load(); got != int32(scenario.want) {
				t.Fatalf("expected %d attempts, got %d", scenario.want, got)
			}
		})
	}
}

func TestRegistryRequestHonorsCancellation(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); fmt.Fprint(w, `{}`) }))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := RegistryRequest(ctx, models.Registry{URL: server.URL}, "GET", "/v2/")
	if !errors.Is(err, context.Canceled) || calls.Load() != 0 {
		t.Fatalf("cancelled requests must not reach registry: %v, calls %d", err, calls.Load())
	}
}
