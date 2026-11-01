package release

import (
	"context"
	"dockerun/internal/models"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCommandDeploymentRunsWithoutCompose(t *testing.T) {
	dir, bin := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "docker"), []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > arguments\necho deploy >> order\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
	s := Snapshot{Release: models.Release{Execution: models.DeploymentExecution{Mode: "command", Command: `docker run --name "$NAME" "$IMAGE"`, CheckCommand: "echo check >> order"}, Configs: `{"configs/app.json":"  exact $value\r\n"}`}, Environment: models.Environment{Parameters: models.StringMap{"NAME": "space ; $(false)", "IMAGE": "nginx:test"}}, Scripts: models.DeploymentScripts{Before: "echo before >> order", After: "echo after >> order"}}
	images, err := Prepare(context.Background(), &s.Release, func(string) {})
	if err != nil || len(images) != 0 {
		t.Fatalf("command unexpectedly required Docker/Compose images: %v", err)
	}
	checkpoint := false
	actual, health, updated, err := Apply(context.Background(), s, dir, func(string, string) {}, func() error { checkpoint = true; return nil })
	if err != nil || !updated || !checkpoint || health != "healthy" || actual != "" {
		t.Fatalf("command result: %q %q %t %v", actual, health, updated, err)
	}
	order, _ := os.ReadFile(filepath.Join(dir, "order"))
	if string(order) != "before\ndeploy\ncheck\nafter\n" {
		t.Fatalf("wrong order: %s", order)
	}
	args, _ := os.ReadFile(filepath.Join(dir, "arguments"))
	if !strings.Contains(string(args), "space ; $(false)\n") {
		t.Fatal("parameter was interpreted as shell source")
	}
	content, _ := os.ReadFile(filepath.Join(dir, "configs/app.json"))
	if string(content) != "  exact $value\r\n" {
		t.Fatalf("config changed: %q", content)
	}
}

func TestCommandFailureStopsLaterStagesAndPreservesResult(t *testing.T) {
	for _, stage := range []string{"before", "deploy", "check", "after", "none"} {
		t.Run(stage, func(t *testing.T) {
			dir := t.TempDir()
			s := Snapshot{Release: models.Release{Execution: models.DeploymentExecution{Mode: "command", Command: "echo deploy >> order", CheckCommand: "echo check >> order"}}, Scripts: models.DeploymentScripts{Before: "echo before >> order", After: "echo after >> order"}}
			switch stage {
			case "before":
				s.Scripts.Before = "exit 7"
			case "deploy":
				s.Release.Execution.Command = "false | true\ntouch forbidden"
			case "check":
				s.Release.Execution.CheckCommand = "exit 8"
			case "after":
				s.Scripts.After = "exit 9"
			case "none":
				s.Release.Execution.CheckCommand = ""
			}
			_, health, updated, err := Apply(context.Background(), s, dir, func(string, string) {}, nil)
			if stage == "none" {
				if err != nil || health != "unchecked" {
					t.Fatal("unchecked command reported healthy", health, err)
				}
				return
			}
			if err == nil || updated != (stage != "before") {
				t.Fatalf("failed stage incorrectly reported: %t %v", updated, err)
			}
			order, _ := os.ReadFile(filepath.Join(dir, "order"))
			if strings.Contains(string(order), "after") {
				t.Fatal("after script ran after failure")
			}
			if _, e := os.Stat(filepath.Join(dir, "forbidden")); !os.IsNotExist(e) {
				t.Fatal("command continued after failed pipeline")
			}
			if stage == "after" && (!strings.Contains(err.Error(), "部署命令已执行") || health != "healthy") {
				t.Fatal("post-script failure hid deployment result", err)
			}
		})
	}
}

func TestCommandValidationAndWorkingDirectory(t *testing.T) {
	dir, working := t.TempDir(), t.TempDir()
	s := Snapshot{Release: models.Release{Execution: models.DeploymentExecution{Mode: "command", WorkDir: "$CUSTOM", Command: `test "$PWD" -ef "$CUSTOM"; test -f "$WORK_DIR/configs/test"; echo done > finished`}, Configs: `{"configs/test":""}`}, Environment: models.Environment{Parameters: models.StringMap{"CUSTOM": working}}}
	if _, _, _, err := Apply(context.Background(), s, dir, func(string, string) {}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(working, "finished")); err != nil {
		t.Fatal(err)
	}
	for _, e := range []models.DeploymentExecution{{Mode: "unknown"}, {Mode: "command"}, {Mode: "command", Command: "if then"}, {Command: "echo \x00"}, {TimeoutSeconds: -1}, {TimeoutSeconds: 86401}, {WorkDir: "a\x00b"}} {
		if ValidateExecution(e) == nil {
			t.Fatalf("invalid config accepted: %+v", e)
		}
	}
	if ExecutionTimeout(models.Release{Execution: models.DeploymentExecution{TimeoutSeconds: 12}}, time.Hour) != 12*time.Second {
		t.Fatal("configured timeout ignored")
	}
	if err := ValidateExecution(models.DeploymentExecution{Mode: "command", Command: "touch '" + filepath.Join(dir, "not-executed") + "'"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "not-executed")); !os.IsNotExist(err) {
		t.Fatal("validation executed command")
	}
}

func TestDisabledAdditionalScriptsKeepMainCommandAndCheck(t *testing.T) {
	dir := t.TempDir()
	s := Snapshot{Release: models.Release{Execution: models.DeploymentExecution{Mode: "command", Command: "echo main >> order", CheckCommand: "echo check >> order"}, Scripts: models.DeploymentScripts{Before: "exit 11", Deploy: "exit 12", After: "exit 13"}}, Environment: models.Environment{ScriptMode: "disabled"}}
	s.Scripts = EffectiveScripts(s.Release, s.Environment)
	if _, _, _, err := Apply(context.Background(), s, dir, func(string, string) {}, nil); err != nil {
		t.Fatal(err)
	}
	order, err := os.ReadFile(filepath.Join(dir, "order"))
	if err != nil || string(order) != "main\ncheck\n" {
		t.Fatalf("additional script policy changed main deployment: %s %v", order, err)
	}
}
