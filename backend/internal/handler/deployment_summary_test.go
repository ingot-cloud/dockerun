package handler

import (
	"dockerun/internal/models"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDeploymentReadableNames(t *testing.T) {
	db := registryTestDB(t)
	if err := db.AutoMigrate(&models.Application{}, &models.Environment{}, &models.Release{}, &models.Target{}, &models.User{}, &models.DeployScript{}, &models.Deployment{}, &models.TaskLog{}); err != nil {
		t.Fatal(err)
	}
	app := models.Application{Name: "支付服务"}
	db.Create(&app)
	env := models.Environment{ApplicationID: app.ID, Name: "生产环境"}
	db.Create(&env)
	version := models.Release{ApplicationID: app.ID, Version: "v2.0.0"}
	db.Create(&version)
	target := models.Target{ID: 2, Name: "上海主机"}
	db.Create(&target)
	user := models.User{Username: "发布管理员", Password: "never-expose-password"}
	db.Create(&user)
	script := models.DeployScript{Name: "安装 Redis", MainScript: "never-expose-script"}
	db.Create(&script)
	tasks := []models.Deployment{
		{EnvironmentID: env.ID, ReleaseID: version.ID, TargetID: target.ID, UserID: user.ID, Batch: "test-batch", Sequence: 0, IdempotencyKey: "named-1", Status: "success"},
		{EnvironmentID: env.ID, ReleaseID: version.ID, TargetID: 1, UserID: user.ID, Batch: "test-batch", Sequence: 1, IdempotencyKey: "named-2", Status: "success"},
		{LegacyScriptID: script.ID, TargetID: 1, IdempotencyKey: "legacy", Status: "success"},
		{EnvironmentID: 999, ReleaseID: 888, TargetID: 1, IdempotencyKey: "archived", Snapshot: fmt.Sprintf(`{"environment":{"name":"归档环境","parameters":{"PASSWORD":"never-expose-snapshot"}},"release":{"version":"v0.1","application_id":%d}}`, app.ID)},
		{EnvironmentID: 777, ReleaseID: 666, IdempotencyKey: "unknown", Snapshot: "invalid old snapshot"},
	}
	if err := db.Create(&tasks).Error; err != nil {
		t.Fatal(err)
	}
	request := func(handler gin.HandlerFunc, path string, id uint) *httptest.ResponseRecorder {
		t.Helper()
		response := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(response)
		ctx.Request = httptest.NewRequest("GET", path, nil)
		ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(id)}}
		handler(ctx)
		if response.Code != 200 {
			t.Fatalf("%s: %d %s", path, response.Code, response.Body.String())
		}
		if strings.Contains(response.Body.String(), "never-expose") || strings.Contains(response.Body.String(), `"snapshot"`) {
			t.Fatal("deployment names exposed snapshot, script or account credentials")
		}
		return response
	}
	response := request(ListDeployments, "/deployments", 0)
	var rows []deploymentSummary
	if err := json.Unmarshal(response.Body.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != len(tasks) || rows[0].ID != tasks[4].ID {
		t.Fatal("missing deployment records or incorrect ordering")
	}
	byID := map[uint]deploymentSummary{}
	for _, row := range rows {
		byID[row.ID] = row
	}
	named := byID[tasks[0].ID]
	if named.Name != "支付服务 · 生产环境 · v2.0.0" || named.EnvironmentName != env.Name || named.ReleaseVersion != version.Version || named.Username != user.Username || named.TargetName != target.Name || named.ApplicationID != app.ID {
		t.Fatalf("names were not joined correctly: %+v", named)
	}
	if byID[tasks[2].ID].Name != "执行脚本 · 安装 Redis" || byID[tasks[2].ID].TargetName != "本机" {
		t.Fatal("legacy task names were lost")
	}
	archived := byID[tasks[3].ID]
	if archived.EnvironmentName != "归档环境" || archived.ReleaseVersion != "v0.1" || archived.ApplicationID != app.ID || archived.Name != "归档环境 · v0.1" {
		t.Fatal("deleted associations did not fall back to the saved snapshot")
	}
	if byID[tasks[4].ID].Name != fmt.Sprintf("部署任务 #%d", tasks[4].ID) {
		t.Fatal("unnamed historical task cannot be identified")
	}
	response = request(ListDeployments, fmt.Sprintf("/deployments?environment_id=%d", env.ID), 0)
	json.Unmarshal(response.Body.Bytes(), &rows)
	if len(rows) != 2 || rows[0].EnvironmentID != env.ID || rows[1].EnvironmentID != env.ID {
		t.Fatal("environment filter changed after joining names")
	}
	response = request(DeploymentDetail, "/deployments/1", tasks[0].ID)
	var detail struct {
		Deployment deploymentSummary   `json:"deployment"`
		Nodes      []deploymentSummary `json:"nodes"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &detail); err != nil {
		t.Fatal(err)
	}
	if detail.Deployment.Name != named.Name || len(detail.Nodes) != 2 || detail.Nodes[0].TargetName != target.Name || detail.Nodes[1].TargetName != "本机" {
		t.Fatal("detail and batch node names do not match the list")
	}
	response = request(ListDeployments, "/deployments?environment_id=123456", 0)
	if strings.TrimSpace(response.Body.String()) != "[]" {
		t.Fatal("empty result must remain an array")
	}
}
