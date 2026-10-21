package handler

import (
	"bufio"
	"dockerun/internal/docker"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func TestConsoleTickets(t *testing.T) {
	store := consoleTicketStore{tickets: map[string]consoleTicket{}}
	key, err := store.issue(consoleTicket{ContainerID: "one", Mode: "exec", Command: []string{"/bin/sh"}})
	if err != nil || len(key) < 40 {
		t.Fatal("failed to issue secure ticket")
	}
	if _, ok := store.consume(key, "two"); ok {
		t.Fatal("ticket accepted for another container")
	}
	if _, ok := store.consume(key, "one"); !ok {
		t.Fatal("fresh ticket rejected")
	}
	if _, ok := store.consume(key, "one"); ok {
		t.Fatal("ticket replay accepted")
	}
	store.tickets["expired"] = consoleTicket{ContainerID: "one", Expires: time.Now().Add(-time.Second)}
	if _, ok := store.consume("expired", "one"); ok {
		t.Fatal("expired ticket accepted")
	}
}
func TestConsoleOrigin(t *testing.T) {
	for _, test := range []struct {
		origin, host string
		allowed      bool
	}{
		{"http://localhost:5173", "127.0.0.1:8080", true}, {"https://app.example.com", "app.example.com", true},
		{"https://evil.example", "app.example.com", false}, {"https://app.example.com.evil.example", "app.example.com", false},
		{"null", "localhost:8080", false}, {"file:///tmp/page", "localhost:8080", false},
	} {
		request := httptest.NewRequest("GET", "http://"+test.host+"/", nil)
		request.Header.Set("Origin", test.origin)
		if consoleOriginAllowed(request) != test.allowed {
			t.Errorf("origin %s host %s", test.origin, test.host)
		}
	}
}
func TestConsoleRoundTrip(t *testing.T) {
	resize := make(chan string, 4)
	daemon := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(req.URL.Path, "/json"):
			io.WriteString(w, `{"Id":"unit","State":{"Running":true,"Status":"running"},"Config":{"Tty":false,"OpenStdin":true}}`)
		case strings.HasSuffix(req.URL.Path, "/exec"):
			var config struct {
				Cmd        []string
				User       string
				Privileged bool
				Tty        bool
			}
			if json.NewDecoder(req.Body).Decode(&config) != nil || len(config.Cmd) != 1 || config.Cmd[0] != "/bin/sh" || config.User != "root" || config.Privileged || !config.Tty {
				t.Error("invalid exec configuration")
			}
			io.WriteString(w, `{"Id":"exec-unit"}`)
		case strings.HasSuffix(req.URL.Path, "/resize"):
			resize <- req.URL.Query().Get("w") + "x" + req.URL.Query().Get("h")
			w.WriteHeader(200)
		case strings.HasSuffix(req.URL.Path, "/start"), strings.HasSuffix(req.URL.Path, "/attach"):
			connection, buffer, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Error(err)
				return
			}
			defer connection.Close()
			buffer.WriteString("HTTP/1.1 101 UPGRADED\r\nContent-Type: application/vnd.docker.raw-stream\r\nConnection: Upgrade\r\nUpgrade: tcp\r\n\r\n")
			buffer.Flush()
			var output io.Writer = connection
			if strings.HasSuffix(req.URL.Path, "/attach") {
				output = stdcopy.NewStdWriter(connection, stdcopy.Stdout)
			}
			io.WriteString(output, "终端已连接\r\n")
			input, err := bufio.NewReader(connection).ReadString('\r')
			if err == nil {
				io.WriteString(output, "收到："+input+"\n")
			}
		default:
			w.WriteHeader(404)
		}
	}))
	defer daemon.Close()
	cli, err := client.NewClientWithOpts(client.WithHost("tcp://"+strings.TrimPrefix(daemon.URL, "http://")), client.WithVersion("1.41"))
	if err != nil {
		t.Fatal(err)
	}
	previous := docker.Client
	docker.Client = cli
	defer func() { docker.Client = previous; cli.Close() }()
	router := gin.New()
	router.POST("/containers/:id/console", func(c *gin.Context) { c.Set("is_admin", true); CreateContainerConsole(c) })
	router.GET("/containers/:id/console/ws", ContainerConsoleSocket)
	app := httptest.NewServer(router)
	defer app.Close()
	for _, mode := range []string{"exec", "attach"} {
		t.Run(mode, func(t *testing.T) {
			response, err := http.Post(app.URL+"/containers/unit/console", "application/json", strings.NewReader(`{"mode":"`+mode+`","command":["/bin/sh"],"user":"root"}`))
			if err != nil {
				t.Fatal(err)
			}
			var ticket struct {
				Ticket string `json:"ticket"`
			}
			json.NewDecoder(response.Body).Decode(&ticket)
			response.Body.Close()
			if response.StatusCode != 200 || ticket.Ticket == "" {
				t.Fatal("console ticket not issued")
			}
			endpoint := "ws" + strings.TrimPrefix(app.URL, "http") + "/containers/unit/console/ws"
			dialer := websocket.Dialer{Subprotocols: []string{"dockerun-console", ticket.Ticket}, HandshakeTimeout: 5 * time.Second}
			if _, bad, err := dialer.Dial(endpoint, http.Header{"Origin": []string{"https://evil.example"}}); err == nil || bad.StatusCode != 403 {
				t.Fatal("foreign origin accepted")
			}
			ws, _, err := dialer.Dial(endpoint, http.Header{"Origin": []string{app.URL}})
			if err != nil {
				t.Fatal(err)
			}
			defer ws.Close()
			ws.SetReadDeadline(time.Now().Add(5 * time.Second))
			if ws.Subprotocol() != "dockerun-console" {
				t.Fatal("wrong subprotocol")
			}
			_, ready, err := ws.ReadMessage()
			if err != nil || !strings.Contains(string(ready), `"type":"ready"`) {
				t.Fatalf("missing ready: %s %v", ready, err)
			}
			kind, output, err := ws.ReadMessage()
			if err != nil || kind != websocket.BinaryMessage || string(output) != "终端已连接\r\n" {
				t.Fatalf("invalid terminal output %q %v", output, err)
			}
			if mode == "exec" {
				ws.WriteJSON(map[string]any{"type": "resize", "cols": 120, "rows": 40})
				select {
				case dimensions := <-resize:
					if dimensions != "120x40" {
						t.Fatal(dimensions)
					}
				case <-time.After(5 * time.Second):
					t.Fatal("terminal not resized")
				}
			}
			ws.WriteJSON(map[string]string{"type": "input", "data": "test-input\r"})
			_, echo, err := ws.ReadMessage()
			if err != nil || !strings.Contains(string(echo), "收到：test-input") {
				t.Fatalf("input not relayed %q %v", echo, err)
			}
			if _, replay, err := dialer.Dial(endpoint, nil); err == nil || replay.StatusCode != 401 {
				t.Fatal("ticket replay accepted")
			}
		})
	}
}
func TestConsoleRequiresAdmin(t *testing.T) {
	router := gin.New()
	router.POST("/containers/:id/console", CreateContainerConsole)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("POST", "/containers/unit/console", strings.NewReader(`{"mode":"attach"}`)))
	if response.Code != 403 {
		t.Fatalf("non-admin status %d", response.Code)
	}
}
