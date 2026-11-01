// Package agent executes only tasks leased to this node and keeps a local journal.
package agent

import (
	"bufio"
	"bytes"
	"context"
	"dockerun/internal/bundle"
	"dockerun/internal/docker"
	"dockerun/internal/identity"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"encoding/json"
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Client struct {
	URL, Token, Dir string
	InstanceID      string
	HTTP            *http.Client
}

func (c *Client) request(ctx context.Context, method, path string, data any) (*http.Response, error) {
	var body io.Reader
	if data != nil {
		b, _ := json.Marshal(data)
		body = bytes.NewReader(b)
	}
	r, err := http.NewRequestWithContext(ctx, method, c.URL+path, body)
	if err != nil {
		return nil, err
	}
	r.Header.Set("Authorization", "Bearer "+c.Token)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Dockerun-Agent-ID", c.InstanceID)
	resp, err := c.HTTP.Do(r)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, fmt.Errorf("控制台 HTTP %d", resp.StatusCode)
	}
	return resp, nil
}
func (c *Client) send(ctx context.Context, path string, data any) error {
	r, e := c.request(ctx, "POST", path, data)
	if e == nil {
		r.Body.Close()
	}
	return e
}
func Run(ctx context.Context, base, token, dir string) error {
	u, err := url.Parse(base)
	if err != nil || u.Host == "" || u.Scheme != "https" && u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1" {
		return fmt.Errorf("Agent 要求 DOCKERUN_URL 使用 HTTPS（本机测试可用 HTTP）")
	}
	if token == "" {
		return fmt.Errorf("请设置 DOCKERUN_AGENT_TOKEN")
	}
	if err = os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	lock, err := os.OpenFile(filepath.Join(dir, "agent.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err = unix.Flock(int(lock.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return fmt.Errorf("此 Agent 数据目录已被另一个进程使用")
	}
	defer unix.Flock(int(lock.Fd()), unix.LOCK_UN)
	if err = docker.Init(); err != nil {
		return err
	}
	instancePath := filepath.Join(dir, "instance-id")
	instance, e := os.ReadFile(instancePath)
	if os.IsNotExist(e) {
		instance = []byte(identity.Random())
		if err := os.WriteFile(instancePath, instance, 0600); err != nil {
			return err
		}
	} else if e != nil {
		return e
	}
	c := &Client{InstanceID: string(instance), URL: base, Token: token, Dir: dir, HTTP: &http.Client{Timeout: 40 * time.Second}}
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		platform, _ := release.Platform(ctx)
		resp, err := c.request(ctx, "GET", "/agent/poll?platform="+url.QueryEscape(platform)+"&scripts=1&execution=1&command_images=1&containers=1&release_policy=2", nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(5 * time.Second):
			}
			continue
		}
		if resp.StatusCode == 204 {
			resp.Body.Close()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Second):
			}
			continue
		}
		var q struct {
			Deployment   models.Deployment `json:"deployment"`
			Snapshot     release.Snapshot  `json:"snapshot"`
			RegistryAuth string            `json:"registry_auth"`
			Previous     *release.Snapshot `json:"previous_snapshot"`
			PreviousAuth string            `json:"previous_auth"`
		}
		err = json.NewDecoder(resp.Body).Decode(&q)
		resp.Body.Close()
		if err != nil {
			continue
		}
		c.execute(ctx, q.Deployment, q.Snapshot, q.RegistryAuth, q.Previous, q.PreviousAuth)
	}
}

type result struct {
	Containers       []models.RuntimeContainer `json:"containers"`
	ScriptsCompleted bool                      `json:"scripts_completed"`
	RecoveryStatus   string                    `json:"recovery_status"`
	Status           string                    `json:"status"`
	Result           string                    `json:"result"`
	ActualImages     string                    `json:"actual_images"`
	Health           string                    `json:"health"`
	Images           []release.Image           `json:"images"`
}
type logEntry struct {
	Sequence int    `json:"sequence"`
	Stage    string `json:"stage"`
	Message  string `json:"message"`
}

func (c *Client) execute(parent context.Context, d models.Deployment, s release.Snapshot, auth string, previous *release.Snapshot, previousAuth string) {
	dir := filepath.Join(c.Dir, fmt.Sprint(d.ID))
	if err := os.MkdirAll(dir, 0700); err != nil {
		return
	}
	base := fmt.Sprintf("/agent/tasks/%d", d.ID)
	reportPath := filepath.Join(dir, "result.json")
	if b, err := os.ReadFile(reportPath); err == nil {
		var r result
		if json.Unmarshal(b, &r) == nil {
			c.send(parent, base+"/complete", r)
			return
		}
	}
	var mu sync.Mutex
	logs := []logEntry{}
	logPath := filepath.Join(dir, "logs.jsonl")
	if f, err := os.Open(logPath); err == nil {
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 4096), 2<<20)
		for scanner.Scan() {
			var l logEntry
			if json.Unmarshal(scanner.Bytes(), &l) == nil {
				logs = append(logs, l)
			}
		}
		f.Close()
	}
	log := func(stage, message string) {
		mu.Lock()
		defer mu.Unlock()
		l := logEntry{Sequence: len(logs) + 1, Stage: stage, Message: message}
		f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err == nil {
			b, _ := json.Marshal(l)
			f.Write(append(b, '\n'))
			f.Sync()
			f.Close()
			logs = append(logs, l)
		}
	}
	flush := func() {
		mu.Lock()
		copyLogs := append([]logEntry{}, logs...)
		mu.Unlock()
		for _, l := range copyLogs {
			if err := c.send(parent, base+"/logs", l); err != nil {
				return
			}
		}
	}
	ctx, cancel := context.WithTimeout(parent, release.ExecutionTimeout(s.Release, time.Hour))
	defer cancel()
	done := make(chan struct{})
	defer close(done)
	go func() {
		t := time.NewTicker(20 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				c.send(parent, base+"/heartbeat", nil)
				flush()
			}
		}
	}()
	r := result{Status: "failed"}
	journal := filepath.Join(dir, "snapshot.json")
	old, already := os.ReadFile(journal)
	if already == nil {
		// A started operation is reconciled, never blindly repeated after an Agent restart.
		if json.Unmarshal(old, &s) == nil && release.IsCommand(s.Release) && s.ScriptsCompleted {
			r.Status, r.Health = "success", "unchecked"
		} else if !release.IsCommand(s.Release) && len(s.Images) > 0 && (!(release.HasScripts(s.Scripts) || release.RequiresExecutionAgent(s.Release)) || s.ScriptsCompleted) {
			p, err := release.Render(ctx, s.Release, s.Environment, dir, s.Images)
			if err == nil {
				r.ActualImages, r.Health, err = release.Actual(ctx, p)
				if err == nil {
					r.Status = "success"
				}
			}
		}
		if r.Status != "success" {
			r.Status = "needs_attention"
			r.Result = "Agent 执行中断，实际状态无法确认，请人工核对"
		}
	} else {
		log("images", "节点开始核对全部交付材料")
		var err error
		if s.Release.BundleID != 0 {
			var resp *http.Response
			resp, err = c.request(ctx, "GET", base+"/bundle", nil)
			if err == nil {
				file := filepath.Join(dir, "release.tar.gz")
				out, e := os.Create(file)
				err = e
				if err == nil {
					_, err = io.Copy(out, io.LimitReader(resp.Body, bundle.MaxSize+1))
					out.Close()
				}
				resp.Body.Close()
				if err == nil {
					hash, e := bundle.HashFile(file)
					if e != nil || hash != resp.Header.Get("X-Archive-SHA256") {
						err = fmt.Errorf("节点下载发布包校验失败")
					}
				}
				if err == nil && s.Release.Source == "local" && resp.Header.Get("X-Archive-Kind") == "image" {
					err = release.LoadImageArchive(ctx, file, "", func(m string) { log("images", m) })
				} else if err == nil && s.Release.Source == "bundle" {
					materials := filepath.Join(dir, "materials")
					os.MkdirAll(materials, 0700)
					var m *bundle.Manifest
					m, err = bundle.Extract(file, materials)
					if err == nil {
						err = bundle.Load(ctx, materials, *m, func(m string) { log("images", m) })
					}
				} else if err == nil {
					err = fmt.Errorf("节点收到的离线材料类型与版本不匹配")
				}
			}
		}
		updated := false
		if err == nil {
			r.ActualImages, r.Health, updated, err = release.RunDeployment(ctx, &s, dir, auth, log, func() error {
				b, _ := json.Marshal(s)
				return os.WriteFile(journal, b, 0600)
			})
		}

		if err == nil {
			s.ScriptsCompleted = true
			b, _ := json.Marshal(s)
			if err = os.WriteFile(journal, b, 0600); err != nil {
				r.Status = "needs_attention"
				r.Result = "服务已更新但脚本完成状态保存失败，请核对发布结果"
			} else {
				r.Status = "success"
				r.Result = "节点部署完成"
			}
		} else {
			r.Result = err.Error()
			if updated && previous != nil && !release.IsCommand(s.Release) && !release.IsCommand(previous.Release) && s.Environment.Stateless && s.Environment.AutoRollback {
				r.RecoveryStatus = "failed"
				log("recovery", "恢复节点上次成功版本与配置")
				restore := previous.Release
				restore.Source = "bundle"
				restore.RegistryID = 0
				data, _ := json.Marshal(previous.Images)
				restore.Images = string(data)
				recoveryDir := filepath.Join(dir, "recovery")
				os.MkdirAll(recoveryDir, 0700)
				ims, e := release.PrepareWithAuth(ctx, &restore, previousAuth, func(m string) { log("recovery", m) })
				if e == nil {
					p, e2 := release.Render(ctx, previous.Release, previous.Environment, recoveryDir, ims)
					e = e2
					if e == nil {
						e = release.Up(ctx, p, recoveryDir, func(m string) { log("recovery", m) })
						if e == nil {
							_, _, e = release.Actual(ctx, p)
						}
					}
				}
				if e == nil {
					r.RecoveryStatus = "success"
				} else {
					log("recovery", e.Error())
				}
			}
		}
		r.Images = s.Images
	}
	r.Images = s.Images
	r.Containers = s.Containers
	r.ScriptsCompleted = s.ScriptsCompleted
	b, _ := json.Marshal(r)
	if err := os.WriteFile(reportPath, b, 0600); err != nil {
		return
	}
	flush()
	c.send(parent, base+"/complete", r)
}
