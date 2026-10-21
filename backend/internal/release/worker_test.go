package release

import (
	"context"
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/models"
	"encoding/json"
	"testing"
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
	app := models.Application{Name: "events"}
	database.DB.Create(&app)
	env := models.Environment{ApplicationID: app.ID, Name: "test", RegistryID: 1, Repository: "team/app", TagPattern: "test-*", AutoDeploy: true, TargetID: 1, AllowedPublishers: "[]"}
	database.DB.Create(&env)
	oldDigest := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	ims, _ := json.Marshal([]Image{{Reference: "localhost:8080/team/app:test-old", Digest: oldDigest}})
	base := models.Release{ApplicationID: app.ID, Source: "registry", RegistryID: 1, Version: "base", Compose: "services:\n  app:\n    image: localhost:8080/team/app:test-old\n", Images: string(ims), Retained: true}
	database.DB.Create(&base)
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
	if releases != 2 {
		t.Fatalf("event idempotency: %d versions", releases)
	}
	database.DB.Model(&env).Update("allowed_publishers", "["+jsonNumber(u.ID)+"]")
	event("allowed", "test-2", u.ID)
	processEvents(context.Background())
	var d models.Deployment
	if database.DB.Where("automatic = ?", true).First(&d).Error != nil {
		t.Fatal("listed publisher did not enqueue")
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
	finish(batch, "failed", "fixture failure")
	database.DB.First(&nodes[1], nodes[1].ID)
	if nodes[1].Status != "paused" {
		t.Fatal("failure did not pause next target")
	}
}
func jsonNumber(v uint) string { b, _ := json.Marshal(v); return string(b) }
