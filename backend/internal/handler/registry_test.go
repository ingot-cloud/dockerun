package handler

import (
	"bytes"
	"dockerun/internal/database"
	"dockerun/internal/models"
	"dockerun/internal/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func registryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	t.Setenv("TEMPLATE_DIR", filepath.Join("..", "..", "..", "templates"))
	t.Setenv("CREDENTIALS_SECRET", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "registries.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Registry{}); err != nil {
		t.Fatal(err)
	}
	previous := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previous; connection, _ := db.DB(); connection.Close() })
	return db
}

func registryRequest(t *testing.T, handler gin.HandlerFunc, id uint, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(response)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(id)}}
	ctx.Request = httptest.NewRequest("POST", "/", bytes.NewReader(raw))
	ctx.Request.Header.Set("Content-Type", "application/json")
	handler(ctx)
	return response
}

func TestRegistryAuthenticationUpdates(t *testing.T) {
	db := registryTestDB(t)
	password, err := utils.SealCredential("test-secret")
	if err != nil {
		t.Fatal(err)
	}
	registry := models.Registry{Name: "test", Kind: "external", URL: "https://registry.test", Username: "deploy", Password: password, Email: "deploy@example.test", PullUsername: "agent", PullPassword: password}
	if err := db.Create(&registry).Error; err != nil {
		t.Fatal(err)
	}
	for _, payload := range []map[string]any{{"name": "legacy update"}, {"authentication": true, "username": "deploy", "password": ""}} {
		response := registryRequest(t, UpdateRegistry, registry.ID, payload)
		if response.Code != http.StatusOK {
			t.Fatalf("update: %d %s", response.Code, response.Body.String())
		}
		db.First(&registry, registry.ID)
		if registry.Password != password || registry.Username != "deploy" {
			t.Fatal("blank secret must preserve existing authenticated credentials")
		}
		if bytes.Contains(response.Body.Bytes(), []byte(password)) || bytes.Contains(response.Body.Bytes(), []byte("test-secret")) {
			t.Fatal("response exposed credentials")
		}
	}
	response := registryRequest(t, UpdateRegistry, registry.ID, map[string]any{"authentication": false, "username": "stale", "password": "stale", "pull_username": "agent", "email": ""})
	if response.Code != http.StatusOK {
		t.Fatal(response.Body.String())
	}
	db.First(&registry, registry.ID)
	if registry.Username != "" || registry.Password != "" {
		t.Fatal("turning authentication off must clear both credentials")
	}
	if registry.Email != "" {
		t.Fatal("explicit empty email must clear the optional field")
	}
	if registry.PullUsername != "agent" || registry.PullPassword != password {
		t.Fatal("agent credentials must be independent of management authentication")
	}
}

func TestRegistryConnectionUsesSavedCredentialsOnlyForSameDestination(t *testing.T) {
	db := registryTestDB(t)
	var authenticated atomic.Bool
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok && username == "deploy" && password == "test-secret" {
			authenticated.Store(true)
			w.WriteHeader(200)
			w.Write([]byte("{}"))
			return
		}
		w.Header().Set("WWW-Authenticate", `Basic realm="registry"`)
		w.WriteHeader(401)
	}))
	defer remote.Close()
	password, err := utils.SealCredential("test-secret")
	if err != nil {
		t.Fatal(err)
	}
	registry := models.Registry{Name: "test", Kind: "external", URL: remote.URL + "/", Username: "deploy", Password: password}
	if err := db.Create(&registry).Error; err != nil {
		t.Fatal(err)
	}
	response := registryRequest(t, TestRegistry, 0, map[string]any{"id": registry.ID, "url": remote.URL, "username": "deploy", "authentication": true})
	if response.Code != http.StatusOK || !authenticated.Load() {
		t.Fatalf("test saved credentials: %d %s", response.Code, response.Body.String())
	}
	for _, payload := range []map[string]any{
		{"id": registry.ID, "url": "http://other.test", "username": "deploy"},
		{"id": registry.ID, "url": remote.URL, "username": "different"},
	} {
		response := registryRequest(t, TestRegistry, 0, payload)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("changed destination must not receive saved secret: %d %s", response.Code, response.Body.String())
		}
	}
	authenticated.Store(false)
	response = registryRequest(t, TestRegistry, 0, map[string]any{"id": registry.ID, "url": remote.URL, "username": "deploy", "authentication": false})
	if response.Code == http.StatusOK || authenticated.Load() {
		t.Fatal("anonymous tests must not reuse saved credentials")
	}
}
