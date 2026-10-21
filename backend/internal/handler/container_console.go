package handler

import (
	"context"
	"crypto/rand"
	"dockerun/internal/docker"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type consoleTicket struct {
	ContainerID string
	Mode        string   `json:"mode"`
	Command     []string `json:"command"`
	User        string   `json:"user"`
	Expires     time.Time
}
type consoleTicketStore struct {
	sync.Mutex
	tickets map[string]consoleTicket
}

var containerConsoleTickets = consoleTicketStore{tickets: make(map[string]consoleTicket)}
var consoleSlots = make(chan struct{}, 64)

func (store *consoleTicketStore) issue(ticket consoleTicket) (string, error) {
	store.Lock()
	defer store.Unlock()
	now := time.Now()
	for key, value := range store.tickets {
		if !now.Before(value.Expires) {
			delete(store.tickets, key)
		}
	}
	if len(store.tickets) >= 128 {
		return "", errors.New("控制台连接请求过多，请稍后重试")
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	key := base64.RawURLEncoding.EncodeToString(secret)
	ticket.Expires = now.Add(30 * time.Second)
	store.tickets[key] = ticket
	return key, nil
}
func (store *consoleTicketStore) consume(key, id string) (consoleTicket, bool) {
	store.Lock()
	defer store.Unlock()
	ticket, ok := store.tickets[key]
	if !ok || ticket.ContainerID != id {
		return consoleTicket{}, false
	}
	delete(store.tickets, key)
	return ticket, time.Now().Before(ticket.Expires)
}
func CreateContainerConsole(c *gin.Context) {
	if !c.GetBool("is_admin") {
		c.JSON(403, gin.H{"error": "此操作需要管理员权限"})
		return
	}
	var req consoleTicket
	if c.ShouldBindJSON(&req) != nil || (req.Mode != "exec" && req.Mode != "attach") {
		c.JSON(400, gin.H{"error": "控制台连接参数无效"})
		return
	}
	if req.Mode == "exec" {
		if len(req.Command) == 0 || len(req.Command) > 128 || strings.TrimSpace(req.Command[0]) == "" || len(strings.Join(req.Command, " ")) > 32768 || len(req.User) > 256 {
			c.JSON(400, gin.H{"error": "执行命令或用户无效"})
			return
		}
	}
	info, err := docker.Client.ContainerInspect(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(404, gin.H{"error": "容器不存在或已被删除"})
		return
	}
	if info.State == nil || !info.State.Running || info.State.Paused || info.State.Restarting {
		c.JSON(409, gin.H{"error": "仅运行中的容器可以连接控制台"})
		return
	}
	// 路由可能使用容器名称；票据绑定当前请求标识并在连接时再次解析。
	req.ContainerID = c.Param("id")
	key, err := containerConsoleTickets.issue(req)
	if err != nil {
		c.JSON(503, gin.H{"error": err.Error()})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(200, gin.H{"ticket": key})
}
func consoleOriginAllowed(req *http.Request) bool {
	raw := req.Header.Get("Origin")
	if raw == "" {
		return true
	}
	origin, err := url.Parse(raw)
	if err != nil || (origin.Scheme != "http" && origin.Scheme != "https") {
		return false
	}
	if strings.EqualFold(origin.Host, req.Host) {
		return true
	}
	// Vite 开发代理会改写 Host，仅在两端均为本机地址时允许该差异。
	host := req.Host
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		host = parsed
	}
	loopback := func(host string) bool {
		if host == "localhost" {
			return true
		}
		ip := net.ParseIP(host)
		return ip != nil && ip.IsLoopback()
	}
	return loopback(origin.Hostname()) && loopback(host)
}

type consoleSocketWriter struct {
	socket *websocket.Conn
	sync.Mutex
}

func (writer *consoleSocketWriter) send(kind int, value []byte) error {
	writer.Lock()
	defer writer.Unlock()
	writer.socket.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return writer.socket.WriteMessage(kind, value)
}
func (writer *consoleSocketWriter) Write(data []byte) (int, error) {
	if err := writer.send(websocket.BinaryMessage, data); err != nil {
		return 0, err
	}
	return len(data), nil
}
func (writer *consoleSocketWriter) message(kind, message string) {
	value, _ := json.Marshal(gin.H{"type": kind, "message": message})
	writer.send(websocket.TextMessage, value)
}
func ContainerConsoleSocket(c *gin.Context) {
	if !consoleOriginAllowed(c.Request) {
		c.AbortWithStatusJSON(403, gin.H{"error": "控制台来源无效"})
		return
	}
	protocols := websocket.Subprotocols(c.Request)
	if len(protocols) != 2 || protocols[0] != "dockerun-console" {
		c.AbortWithStatusJSON(401, gin.H{"error": "控制台连接票据无效"})
		return
	}
	ticket, ok := containerConsoleTickets.consume(protocols[1], c.Param("id"))
	if !ok {
		c.AbortWithStatusJSON(401, gin.H{"error": "控制台票据已过期或已使用，请重新连接"})
		return
	}
	select {
	case consoleSlots <- struct{}{}:
		defer func() { <-consoleSlots }()
	default:
		c.JSON(503, gin.H{"error": "控制台连接数已达上限"})
		return
	}
	upgrader := websocket.Upgrader{Subprotocols: []string{"dockerun-console"}, CheckOrigin: consoleOriginAllowed, HandshakeTimeout: 10 * time.Second}
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer ws.Close()
	ws.SetReadLimit(65536)
	writer := &consoleSocketWriter{socket: ws}
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	info, err := docker.Client.ContainerInspect(ctx, ticket.ContainerID)
	if err != nil || info.State == nil || !info.State.Running || info.State.Paused || info.State.Restarting {
		writer.message("error", "容器当前不可连接，请刷新容器状态")
		return
	}
	var stream types.HijackedResponse
	var execID string
	tty := info.Config != nil && info.Config.Tty
	if ticket.Mode == "exec" {
		exec, createErr := docker.Client.ContainerExecCreate(ctx, ticket.ContainerID, types.ExecConfig{User: ticket.User, Cmd: ticket.Command, Tty: true, AttachStdin: true, AttachStdout: true, AttachStderr: true})
		if createErr != nil {
			writer.message("error", createErr.Error())
			return
		}
		execID = exec.ID
		stream, err = docker.Client.ContainerExecAttach(ctx, execID, types.ExecStartCheck{Tty: true})
		tty = true
	} else {
		stream, err = docker.Client.ContainerAttach(ctx, ticket.ContainerID, container.AttachOptions{Stream: true, Stdin: info.Config != nil && info.Config.OpenStdin, Stdout: true, Stderr: true, DetachKeys: "ctrl-p,ctrl-q"})
	}
	if err != nil {
		writer.message("error", err.Error())
		return
	}
	defer stream.Close()
	stop := context.AfterFunc(ctx, func() { stream.Close(); ws.Close() })
	defer stop()
	writer.message("ready", "控制台已连接")
	ws.SetReadDeadline(time.Now().Add(90 * time.Second))
	ws.SetPongHandler(func(string) error { return ws.SetReadDeadline(time.Now().Add(90 * time.Second)) })
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if ws.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second)) != nil {
					cancel()
					return
				}
			}
		}
	}()
	go func() {
		if tty {
			io.Copy(writer, stream.Reader)
		} else {
			stdcopy.StdCopy(writer, writer, stream.Reader)
		}
		writer.message("exit", "控制台连接已结束")
		cancel()
	}()
	for {
		var input struct {
			Type string `json:"type"`
			Data string `json:"data"`
			Cols uint   `json:"cols"`
			Rows uint   `json:"rows"`
		}
		if ws.ReadJSON(&input) != nil {
			return
		}
		switch input.Type {
		case "input":
			if ticket.Mode == "attach" && (info.Config == nil || !info.Config.OpenStdin) {
				continue
			}
			stream.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if _, err := io.WriteString(stream.Conn, input.Data); err != nil {
				return
			}
		case "resize":
			if input.Cols == 0 || input.Rows == 0 || input.Cols > 1000 || input.Rows > 1000 {
				continue
			}
			resizeCtx, resizeCancel := context.WithTimeout(ctx, 5*time.Second)
			options := container.ResizeOptions{Width: input.Cols, Height: input.Rows}
			if execID != "" {
				docker.Client.ContainerExecResize(resizeCtx, execID, options)
			} else if tty {
				docker.Client.ContainerResize(resizeCtx, ticket.ContainerID, options)
			}
			resizeCancel()
		}
	}
}
