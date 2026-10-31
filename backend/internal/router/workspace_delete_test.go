package router_test

import (
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/identity"
	"dockerun/internal/models"
	"dockerun/internal/router"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWorkspaceDeletionRequiresAdministrator(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	t.Setenv("GIN_MODE", "test")
	if err := config.Initialize(); err != nil {
		t.Fatal(err)
	}
	previous := database.DB
	if err := database.Init(config.Load().DatabasePath); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { connection, _ := database.DB.DB(); connection.Close(); database.DB = previous })
	var admin models.User
	database.DB.First(&admin, 1)
	admin.Password, admin.Enabled = "initialized-fixture", true
	database.DB.Save(&admin)
	member := models.User{Username: "reader", Password: "initialized-fixture", Enabled: true, AuthVersion: 1}
	database.DB.Create(&member)
	adminToken, err := identity.Issue(&admin, "web", nil, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	memberToken, err := identity.Issue(&member, "web", nil, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	app := models.Application{Name: "路由删除应用"}
	template := models.ApplicationTemplate{Name: "路由删除模板", Version: "v1"}
	bundle := models.Bundle{Name: "路由删除归档", SHA256: strings.Repeat("c", 64), Status: "failed"}
	task := models.Deployment{Status: "success", IdempotencyKey: "route-delete"}
	version := models.Release{Version: "路由删除版本"}
	environmentApp := models.Application{Name: "路由环境所属应用"}
	for _, row := range []any{&app, &template, &bundle, &task, &version, &environmentApp} {
		if err := database.DB.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	environment := models.Environment{ApplicationID: environmentApp.ID, Name: "路由删除环境"}
	if err := database.DB.Create(&environment).Error; err != nil {
		t.Fatal(err)
	}
	routes := router.Setup()
	for _, path := range []string{fmt.Sprintf("/api/applications/%d", app.ID), fmt.Sprintf("/api/managed-templates/custom-%d", template.ID), fmt.Sprintf("/api/bundles/%d", bundle.ID), fmt.Sprintf("/api/deployments/%d", task.ID), fmt.Sprintf("/api/releases/%d", version.ID), fmt.Sprintf("/api/applications/%d/environments/%d", environmentApp.ID, environment.ID)} {
		for _, role := range []struct {
			token  string
			status int
		}{{"", 401}, {memberToken, 403}, {adminToken, 200}} {
			request := httptest.NewRequest("DELETE", path, nil)
			if role.token != "" {
				request.Header.Set("Authorization", "Bearer "+role.token)
			}
			response := httptest.NewRecorder()
			routes.ServeHTTP(response, request)
			if response.Code != role.status {
				t.Fatalf("%s：状态 %d，预期 %d，响应 %s", path, response.Code, role.status, response.Body.String())
			}
		}
	}
}
