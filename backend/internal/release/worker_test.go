package release

import (
	"context"
	"dockerun/internal/builtin"
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/identity"
	"dockerun/internal/models"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEventsPermissionsAndSerialBatch(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	if err := config.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := database.Init(config.Load().DatabasePath); err != nil {
		t.Fatal(err)
	}
	admin := models.User{}
	database.DB.First(&admin, 1)
	admin.HashPassword("test-password")
	database.DB.Save(&admin)
	u := models.User{Username: "publisher", Enabled: true, RegistryAccess: "read_write", AuthVersion: 1}
	u.HashPassword("test-password")
	database.DB.Create(&u)
	app := models.Application{Name: "events", Scripts: models.DeploymentScripts{Before: "echo application"}}
	database.DB.Create(&app)
	env := models.Environment{ApplicationID: app.ID, Name: "test", RegistryID: 1, Repository: "team/app", TagPattern: "test-*", AutoDeploy: true, TargetID: 1, AllowedPublishers: "[]"}
	database.DB.Create(&env)
	oldDigest := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	ims, _ := json.Marshal([]Image{{Reference: "localhost:8080/team/app:test-old", Digest: oldDigest}})
	base := models.Release{ApplicationID: app.ID, Source: "registry", RegistryID: 1, Version: "base", Compose: "services:\n  app:\n    image: localhost:8080/team/app:test-old\n", Images: string(ims), Retained: true}
	database.DB.Create(&base)
	env.CurrentReleaseID = base.ID
	database.DB.Save(&env)
	// A draft for another environment must never become this environment's deployment definition.
	database.DB.Create(&models.Release{ApplicationID: app.ID, Source: "local", Version: "unrelated-draft", Compose: "invalid", Images: "[]"})
	event := func(key, ref string, uid uint) {
		database.DB.Create(&models.ImageEvent{RegistryID: 1, Repository: "team/app", Reference: ref, Digest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Key: key, Committed: true, UserID: uid})
	}
	event("ordinary", "test-1", u.ID)
	processEvents(context.Background())
	var n int64
	database.DB.Model(&models.Deployment{}).Count(&n)
	if n != 0 {
		t.Fatal("unlisted ordinary publisher triggered deployment")
	}
	processEvents(context.Background())
	var releases int64
	database.DB.Model(&models.Release{}).Count(&releases)
	if releases != 3 {
		t.Fatalf("event idempotency: %d versions", releases)
	}
	database.DB.Model(&env).Update("allowed_publishers", "["+jsonNumber(u.ID)+"]")
	event("allowed", "test-2", u.ID)
	processEvents(context.Background())
	var d models.Deployment
	if database.DB.Where("automatic = ?", true).First(&d).Error != nil {
		t.Fatal("listed publisher did not enqueue")
	}
	var automaticSnapshot Snapshot
	if json.Unmarshal([]byte(d.Snapshot), &automaticSnapshot) != nil || automaticSnapshot.Scripts.Before != "echo application" || automaticSnapshot.Release.Compose != base.Compose {
		t.Fatal("automatic deployment did not capture scripts and the environment's current definition")
	}
	database.DB.Model(&env).Update("allowed_publishers", "[]")
	execute(context.Background(), &d)
	database.DB.First(&d, d.ID)
	if d.Status != "cancelled" {
		t.Fatalf("revoked queued publisher executed: %s", d.Status)
	}
	node := models.Target{Name: "remote", Enabled: true}
	database.DB.Create(&node)
	env.AutoDeploy = false
	env.ScriptMode = "override"
	env.Scripts = models.DeploymentScripts{After: "echo environment"}
	env.TargetIDs = "[1," + jsonNumber(node.ID) + "]"
	database.DB.Save(&env)
	batch, err := Enqueue(env, base, admin.ID, "batch-test", false, 0)
	if err != nil {
		t.Fatal(err)
	}
	again, err := Enqueue(env, base, admin.ID, "batch-test", false, 0)
	if err != nil || again.ID != batch.ID {
		t.Fatal("duplicate task")
	}
	var nodes []models.Deployment
	database.DB.Where("batch = ?", "batch-test").Order("sequence").Find(&nodes)
	if len(nodes) != 2 || nodes[1].Status != "blocked" {
		t.Fatal("targets were not serialized")
	}
	for _, task := range nodes {
		var snap Snapshot
		json.Unmarshal([]byte(task.Snapshot), &snap)
		if snap.Scripts.After != "echo environment" || snap.Scripts.Before != "" {
			t.Fatal("local and remote targets did not preserve the environment script override")
		}
	}
	finish(batch, "failed", "fixture failure")
	database.DB.First(&nodes[1], nodes[1].ID)
	if nodes[1].Status != "paused" {
		t.Fatal("failure did not pause next target")
	}
	// The built-in manifest endpoint directly produces a deployable event without a WebhookConfig.
	env.TargetIDs = ""
	env.TargetID = 1
	env.AutoDeploy = true
	database.DB.Save(&env)
	previousEngine := builtin.Engine
	builtin.Engine = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(201) })
	t.Cleanup(func() { builtin.Engine = previousEngine })
	token, err := identity.Issue(&admin, "registry", []string{"repository:team/app:push"}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPut, "/v2/team/app/manifests/test-pushed", strings.NewReader(`{"schemaVersion":2}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
	w := httptest.NewRecorder()
	builtin.Serve(w, req)
	if w.Code != 201 {
		t.Fatal(w.Body.String())
	}
	var pushed models.ImageEvent
	if database.DB.Where("reference = ? AND committed = ?", "test-pushed", true).First(&pushed).Error != nil {
		t.Fatal("built-in push did not persist its event")
	}
	processEvents(context.Background())
	var automatic models.Deployment
	if database.DB.Where("event_id = ? AND automatic = ?", pushed.ID, true).First(&automatic).Error != nil {
		t.Fatal("built-in push did not enqueue automatic deployment")
	}
	var configs int64
	database.DB.Model(&models.WebhookConfig{}).Count(&configs)
	if configs != 0 {
		t.Fatal("built-in automatic deployment unexpectedly required a webhook")
	}
	interruptedSnapshot, _ := json.Marshal(Snapshot{Release: base, Environment: env, Scripts: models.DeploymentScripts{Before: "echo do-not-repeat"}, Images: []Image{{Reference: "localhost:8080/team/app:test-old", ID: "sha256:fixture"}}})
	interrupted := models.Deployment{EnvironmentID: env.ID, Status: "running", Snapshot: string(interruptedSnapshot), IdempotencyKey: "interrupted-script"}
	database.DB.Create(&interrupted)
	reconcile(context.Background())
	database.DB.First(&interrupted, interrupted.ID)
	if interrupted.Status != "needs_attention" || !strings.Contains(interrupted.Result, "不会自动重跑") {
		t.Fatal("an interrupted script was not held for manual review")
	}
}
func jsonNumber(v uint) string { b, _ := json.Marshal(v); return string(b) }
