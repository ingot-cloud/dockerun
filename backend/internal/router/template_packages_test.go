package router_test

import (
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/identity"
	"dockerun/internal/models"
	"dockerun/internal/router"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTemplatePackageRoutesRequireAdministrator(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	t.Setenv("TEMPLATE_DIR", t.TempDir())
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
	admin.Password = "initialized-fixture"
	database.DB.Save(&admin)
	reader := models.User{Username: "reader", Password: "initialized-fixture", Enabled: true, AuthVersion: 1}
	database.DB.Create(&reader)
	memberToken, err := identity.Issue(&reader, "web", nil, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	adminToken, err := identity.Issue(&admin, "web", nil, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	routes := router.Setup()
	for _, endpoint := range []struct {
		method, path string
		status       int
	}{{"POST", "/api/managed-templates/imports", 400}, {"POST", "/api/managed-templates/imports/invalid/commit", 400}, {"DELETE", "/api/managed-templates/imports/invalid", 400}, {"POST", "/api/managed-templates/export", 400}, {"GET", "/api/managed-templates/not-found/download", 404}} {
		for _, role := range []struct {
			token  string
			status int
		}{{"", 401}, {memberToken, 403}, {adminToken, endpoint.status}} {
			request := httptest.NewRequest(endpoint.method, endpoint.path, strings.NewReader("{}"))
			request.Header.Set("Content-Type", "application/json")
			if role.token != "" {
				request.Header.Set("Authorization", "Bearer "+role.token)
			}
			response := httptest.NewRecorder()
			routes.ServeHTTP(response, request)
			if response.Code != role.status {
				t.Fatalf("%s %s 状态 %d，预期 %d：%s", endpoint.method, endpoint.path, response.Code, role.status, response.Body.String())
			}
		}
	}
}
