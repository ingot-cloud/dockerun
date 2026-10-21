package main

import (
	"bytes"
	"context"
	"dockerun/internal/agent"
	"dockerun/internal/backup"
	"dockerun/internal/bundle"
	"dockerun/internal/docker"
	"dockerun/internal/models"
	"encoding/json"
	"flag"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var client = &http.Client{Timeout: 2 * time.Minute}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "dockerun:", err)
		os.Exit(1)
	}
}
func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("commands: deploy, package, import, promote, rollback, status, restore, agent")
	}
	cmd := args[0]
	args = args[1:]
	if cmd == "agent" {
		fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
		dir := fs.String("data-dir", "/var/lib/dockerun-agent", "persistent Agent directory")
		if err := fs.Parse(args); err != nil {
			return err
		}
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()
		return agent.Run(ctx, os.Getenv("DOCKERUN_URL"), os.Getenv("DOCKERUN_AGENT_TOKEN"), *dir)
	}
	if cmd == "restore" {
		fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
		dir := fs.String("data-dir", "", "new empty destination directory")
		file := fs.String("file", "", "backup tar.gz")
		if err := fs.Parse(args); err != nil {
			return err
		}
		if *dir == "" || *file == "" {
			return fmt.Errorf("restore --file BACKUP --data-dir NEW_EMPTY_DIR")
		}
		return backup.Restore(*file, *dir)
	}
	name := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		name = args[0]
		args = args[1:]
	}
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	env := fs.String("env", "test", "deployment environment")
	to := fs.String("to", "prod", "promotion environment")
	compose := fs.String("file", "compose.yaml", "Compose definition")
	platform := fs.String("platform", "", "target platform")
	output := fs.String("output", "application-release.tar.gz", "output bundle")
	version := fs.String("version", time.Now().Format("20060102-150405"), "release version")
	wait := fs.Bool("wait", false, "wait for actual result")
	build := fs.Bool("build", false, "build and push with Docker Compose")
	noCache := fs.Bool("no-cache", false, "explicit full rebuild")
	releaseID := fs.Uint("release", 0, "existing immutable release ID")
	registryID := fs.Uint("registry", 0, "Registry connection ID")
	source := fs.String("source", "registry", "registry or local")
	paramsFile := fs.String("defaults", "", "JSON non-sensitive defaults for package interpolation")
	bundleFile := fs.String("bundle", "", "complete offline release tar.gz")
	buildScript := fs.String("build-script", "", "existing build script executable path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if cmd == "package" {
		if name == "" {
			return fmt.Errorf("package APPLICATION --file compose.yaml --platform linux/amd64 --output release.tar.gz")
		}
		if err := docker.Init(); err != nil {
			return err
		}
		var defaults map[string]string
		if *paramsFile != "" {
			b, err := os.ReadFile(*paramsFile)
			if err != nil {
				return err
			}
			if err = json.Unmarshal(b, &defaults); err != nil {
				return err
			}
		}
		git, _ := exec.Command("git", "rev-parse", "HEAD").Output()
		return bundle.Pack(context.Background(), *compose, *output, bundle.Manifest{Application: name, Version: *version, Platform: *platform, Defaults: defaults, GitCommit: strings.TrimSpace(string(git))})
	}
	if cmd == "import" {
		if name == "" {
			return fmt.Errorf("import FILE.tar.gz")
		}
		_, err := upload(name, *wait)
		return err
	}
	if cmd == "status" {
		path := "/api/deployments"
		if name != "" {
			path += "/" + name
		}
		var v any
		if err := call("GET", path, nil, &v); err != nil {
			return err
		}
		return printJSON(v)
	}
	if name == "" {
		return fmt.Errorf("请指定应用名称")
	}
	var apps []models.Application
	if err := call("GET", "/api/applications", nil, &apps); err != nil {
		return err
	}
	var app models.Application
	for _, a := range apps {
		if a.Name == name {
			app = a
			break
		}
	}
	if app.ID == 0 {
		return fmt.Errorf("应用不存在，请先在控制台创建应用和环境")
	}
	var detail struct {
		Environments []models.Environment `json:"environments"`
		Releases     []models.Release     `json:"releases"`
	}
	if err := call("GET", fmt.Sprintf("/api/applications/%d", app.ID), nil, &detail); err != nil {
		return err
	}
	envName := *env
	if cmd == "promote" {
		envName = *to
	}
	var target models.Environment
	for _, e := range detail.Environments {
		if e.Name == envName {
			target = e
			break
		}
	}
	if target.ID == 0 {
		return fmt.Errorf("环境不存在: %s", envName)
	}
	if *build && *platform == "" {
		var sources struct {
			Targets []models.Target `json:"targets"`
		}
		if err := call("GET", fmt.Sprintf("/api/applications/%d/sources", app.ID), nil, &sources); err != nil {
			return err
		}
		ids := []uint{target.TargetID}
		if target.TargetIDs != "" && target.TargetIDs != "[]" {
			if err := json.Unmarshal([]byte(target.TargetIDs), &ids); err != nil {
				return err
			}
		}
		platforms := []string{}
		seen := map[string]bool{}
		for _, id := range ids {
			if id == 0 {
				id = 1
			}
			found := ""
			for _, n := range sources.Targets {
				if n.ID == id {
					found = n.Platform
				}
			}
			if found == "" {
				return fmt.Errorf("目标主机平台尚未确认；请连接 Agent 或显式指定 --platform")
			}
			if !seen[found] {
				seen[found] = true
				platforms = append(platforms, found)
			}
		}
		*platform = strings.Join(platforms, ",")
	}
	if cmd == "rollback" {
		if *releaseID == 0 {
			return fmt.Errorf("恢复须显式选择 --release ID；请先查看控制台中的镜像及配置差异")
		}
	}
	if cmd != "deploy" && cmd != "promote" && cmd != "rollback" {
		return fmt.Errorf("unknown command: %s", cmd)
	}
	id := *releaseID
	if *bundleFile != "" {
		b, err := upload(*bundleFile, true)
		if err != nil {
			return err
		}
		var result struct {
			Release models.Release `json:"release"`
		}
		if err = call("POST", fmt.Sprintf("/api/bundles/%d/version", b.ID), map[string]any{"application_id": app.ID}, &result); err != nil {
			return err
		}
		id = result.Release.ID
	}
	if id == 0 {
		b, err := os.ReadFile(*compose)
		if err != nil {
			return err
		}
		content := string(b)
		if *buildScript != "" {
			s := exec.Command(*buildScript)
			s.Stdout = os.Stdout
			s.Stderr = os.Stderr
			if err = s.Run(); err != nil {
				return err
			}
		}
		if *build {
			args := []string{"buildx", "bake", "--file", *compose, "--push"}
			if *platform != "" {
				args = append(args, "--set", "*.platform="+*platform)
			}
			if *noCache {
				args = append(args, "--no-cache")
			}
			c := exec.Command("docker", args...)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			if err = c.Run(); err != nil {
				return err
			}
			c = exec.Command("docker", "compose", "-f", *compose, "config", "--format", "json", "--no-env-resolution")
			rendered, err := c.Output()
			if err != nil {
				return err
			}
			content, err = publicationDefinition(b, rendered)
			if err != nil {
				return err
			}
		}
		if *registryID == 0 && *source == "registry" {
			var sources struct {
				Registries []models.Registry `json:"registries"`
			}
			if err := call("GET", fmt.Sprintf("/api/applications/%d/sources", app.ID), nil, &sources); err != nil {
				return err
			}
			for _, r := range sources.Registries {
				if r.IsDefault {
					*registryID = r.ID
				}
			}
		}
		git, _ := exec.Command("git", "rev-parse", "HEAD").Output()
		configs, err := bundle.Configs(filepath.Dir(*compose))
		if err != nil {
			return err
		}
		configJSON, err := json.Marshal(configs)
		if err != nil {
			return err
		}
		r := models.Release{Configs: string(configJSON), Version: *version, Source: *source, RegistryID: *registryID, Compose: content, GitCommit: strings.TrimSpace(string(git))}
		if err := call("POST", fmt.Sprintf("/api/applications/%d/releases?environment_id=%d", app.ID, target.ID), r, &r); err != nil {
			return err
		}
		id = r.ID
	}
	var task models.Deployment
	if err := call("POST", "/api/deployments", map[string]any{"environment_id": target.ID, "release_id": id, "restore_configuration": cmd == "rollback", "idempotency_key": fmt.Sprintf("cli-%d", time.Now().UnixNano())}, &task); err != nil {
		return err
	}
	if *wait {
		return waitTask(task.ID)
	}
	return printJSON(task)
}
func request(method, path string, body io.Reader) (*http.Request, error) {
	base := strings.TrimSuffix(os.Getenv("DOCKERUN_URL"), "/")
	if base == "" {
		base = "http://localhost:8080"
	}
	req, err := http.NewRequest(method, base+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+os.Getenv("DOCKERUN_TOKEN"))
	return req, nil
}
func call(method, path string, data, out any) error {
	var body io.Reader
	if data != nil {
		b, err := json.Marshal(data)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := request(method, path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, b)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}
func printJSON(v any) error {
	e := json.NewEncoder(os.Stdout)
	e.SetIndent("", "  ")
	return e.Encode(v)
}
func waitTask(id uint) error {
	after := uint(0)
	for {
		var r struct {
			Deployment models.Deployment   `json:"deployment"`
			Logs       []models.TaskLog    `json:"logs"`
			Nodes      []models.Deployment `json:"nodes"`
		}
		if err := call("GET", fmt.Sprintf("/api/deployments/%d?after=%d", id, after), nil, &r); err != nil {
			return err
		}
		for _, l := range r.Logs {
			fmt.Printf("[%s] %s\n", l.Stage, l.Message)
			after = l.ID
		}
		switch r.Deployment.Status {
		case "success":
			pending := false
			for _, n := range r.Nodes {
				switch n.Status {
				case "queued", "running", "blocked":
					pending = true
				case "success":
				default:
					return fmt.Errorf("主机 #%d: %s %s", n.TargetID, n.Status, n.Result)
				}
			}
			if pending {
				time.Sleep(time.Second)
				continue
			}
			fmt.Println(r.Deployment.Result)
			return nil
		case "queued", "running", "blocked":
			time.Sleep(time.Second)
		default:
			return fmt.Errorf("部署 %s: %s", r.Deployment.Status, r.Deployment.Result)
		}
	}
}
func upload(path string, wait bool) (models.Bundle, error) {
	hash, err := bundle.HashFile(path)
	if err != nil {
		return models.Bundle{}, err
	}
	var state struct {
		Offset int64 `json:"offset"`
	}
	if err = call("GET", "/api/bundles/uploads/"+hash, nil, &state); err != nil {
		return models.Bundle{}, err
	}
	f, err := os.Open(path)
	if err != nil {
		return models.Bundle{}, err
	}
	defer f.Close()
	st, _ := f.Stat()
	for state.Offset < st.Size() {
		end := state.Offset + 8<<20
		if end > st.Size() {
			end = st.Size()
		}
		req, err := request("PUT", "/api/bundles/uploads/"+hash, io.NewSectionReader(f, state.Offset, end-state.Offset))
		if err != nil {
			return models.Bundle{}, err
		}
		req.ContentLength = end - state.Offset
		req.Header.Set("Content-Type", "application/octet-stream")
		req.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", state.Offset, end-1, st.Size()))
		resp, err := client.Do(req)
		if err != nil {
			return models.Bundle{}, err
		}
		err = json.NewDecoder(resp.Body).Decode(&state)
		resp.Body.Close()
		if err != nil || resp.StatusCode != 200 {
			return models.Bundle{}, fmt.Errorf("分块上传失败，可重新执行命令续传")
		}
		fmt.Fprintf(os.Stderr, "uploaded %d/%d\n", state.Offset, st.Size())
	}
	var b models.Bundle
	if err = call("POST", "/api/bundles/uploads/"+hash+"/complete", map[string]any{"name": filepath.Base(path)}, &b); err != nil {
		return models.Bundle{}, err
	}
	if !wait {
		err := printJSON(b)
		return b, err
	}
	for {
		var rows []models.Bundle
		if err = call("GET", "/api/bundles", nil, &rows); err != nil {
			return models.Bundle{}, err
		}
		for _, v := range rows {
			if v.ID == b.ID {
				switch v.Status {
				case "ready":
					err := printJSON(v)
					return v, err
				case "failed":
					return models.Bundle{}, fmt.Errorf("导入失败: %s", v.Error)
				}
			}
		}
		time.Sleep(time.Second)
	}
}

// Only build-time image names cross from local interpolation into the release.
// Runtime parameters stay unresolved for the selected server environment.
func publicationDefinition(original, resolved []byte) (string, error) {
	var doc, local map[string]any
	if err := yaml.Unmarshal(original, &doc); err != nil {
		return "", err
	}
	if err := json.Unmarshal(resolved, &local); err != nil {
		return "", err
	}
	services, _ := doc["services"].(map[string]any)
	built, _ := local["services"].(map[string]any)
	for name, value := range services {
		svc, ok := value.(map[string]any)
		if !ok {
			continue
		}
		delete(svc, "build")
		if actual, ok := built[name].(map[string]any); ok {
			if image, ok := actual["image"].(string); ok {
				svc["image"] = image
			}
		}
	}
	b, err := json.Marshal(doc)
	return string(b), err
}
