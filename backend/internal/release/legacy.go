package release

import (
	"context"
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/identity"
	"dockerun/internal/models"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func EnqueueLegacy(script models.DeployScript, uid uint) (*models.Deployment, error) {
	b, _ := json.Marshal(script)
	d := models.Deployment{LegacyScriptID: script.ID, UserID: uid, IdempotencyKey: identity.Random(), Status: "queued", Stage: "queued", Snapshot: string(b)}
	err := database.DB.Create(&d).Error
	return &d, err
}
func executeLegacy(parent context.Context, d *models.Deployment) {
	var u models.User
	if database.DB.First(&u, d.UserID).Error != nil || !u.Enabled || !u.IsAdmin {
		finish(d, "cancelled", "管理员授权已失效")
		return
	}
	var s models.DeployScript
	if json.Unmarshal([]byte(d.Snapshot), &s) != nil {
		finish(d, "failed", "脚本快照无效")
		return
	}
	database.DB.Model(d).Update("status", "running")
	ctx, cancel := context.WithTimeout(parent, Timeout())
	defer cancel()
	dir := s.WorkDir
	if dir == "" || dir == "/ingot-data/dockerun" {
		dir = filepath.Join(config.Load().DataDir, "jobs", fmt.Sprintf("legacy-%d", s.ID))
	}
	dir, _ = filepath.Abs(dir)
	err := os.MkdirAll(dir, 0700)
	if err == nil {
		for _, f := range s.Files {
			relative := f.FilePath
			if relative == "" {
				relative = f.FileName
			}
			p, e := SafePath(dir, relative)
			if e != nil {
				err = e
				break
			}
			if err = os.MkdirAll(filepath.Dir(p), 0700); err != nil {
				break
			}
			perm := os.FileMode(0600)
			if f.FileType == "script" {
				perm = 0700
			}
			if err = os.WriteFile(p, []byte(f.Content), perm); err != nil {
				break
			}
		}
	}
	if err != nil {
		finish(d, "failed", err.Error())
		return
	}
	path := filepath.Join(dir, "deploy.sh")
	if err = os.WriteFile(path, []byte(s.MainScript), 0700); err != nil {
		finish(d, "failed", err.Error())
		return
	}
	cmd := exec.CommandContext(ctx, "/bin/bash", path)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "WORK_DIR="+dir, "DOCKER_NETWORK="+config.Load().NetworkName)
	for k, v := range s.EnvVars {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	output := logWriter{func(m string) { Log(d, "legacy", m) }}
	cmd.Stdout = output
	cmd.Stderr = output
	database.DB.Model(&models.DeployScript{}).Where("id = ?", s.ID).Update("status", "running")
	err = cmd.Run()
	status := "success"
	message := "历史脚本执行完成；请自行确认服务健康"
	scriptStatus := "idle"
	if err != nil {
		status = "failed"
		scriptStatus = "failed"
		message = err.Error()
	}
	database.DB.Model(&models.DeployScript{}).Where("id = ?", s.ID).Update("status", scriptStatus)
	finish(d, status, message)
}
