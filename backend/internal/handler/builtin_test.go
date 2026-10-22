package handler

import (
	"dockerun/internal/builtin"
	"dockerun/internal/database"
	"dockerun/internal/models"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func builtinTestUser(t *testing.T) models.User {
	t.Helper()
	db := registryTestDB(t)
	if err := db.AutoMigrate(&models.User{}, &models.ImageEvent{}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("REGISTRY_SECRET", strings.Repeat("a", 64))
	user := models.User{Username: "admin", Password: "initialized-test-password", IsAdmin: true, Enabled: true, AuthVersion: 1}
	if err := database.DB.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	previous := builtin.Engine
	t.Cleanup(func() { builtin.Engine = previous })
	return user
}

func builtinCatalogRequest(user *models.User) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(response)
	ctx.Set("user", user)
	ctx.Request = httptest.NewRequest("GET", "/api/builtin/catalog", nil)
	BuiltinCatalog(ctx)
	return response
}

func TestBuiltinCatalogCollectsAllPages(t *testing.T) {
	user := builtinTestUser(t)
	calls := 0
	builtin.Engine = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/v2/_catalog" || r.URL.Query().Get("n") != fmt.Sprint(builtin.CatalogPageSize) {
			t.Errorf("unexpected request: %s", r.URL)
		}
		repositories := make([]string, builtin.CatalogPageSize)
		for i := range repositories {
			repositories[i] = fmt.Sprintf("team/app%04d", i)
		}
		switch calls {
		case 1:
			if r.URL.Query().Get("last") != "" {
				t.Error("unexpected initial cursor")
			}
			w.Header().Set("Link", "</v2/_catalog?last=team/app0999>; rel=\"next\"")
			json.NewEncoder(w).Encode(map[string]any{"repositories": repositories})
		case 2:
			if r.URL.Query().Get("last") != repositories[len(repositories)-1] {
				t.Error("missing continuation cursor")
			}
			json.NewEncoder(w).Encode(map[string]any{"repositories": []string{"team/final"}})
		default:
			t.Fatal("unexpected extra page")
		}
	})
	response := builtinCatalogRequest(&user)
	var result struct {
		Catalog struct {
			Repositories []string `json:"repositories"`
		} `json:"catalog"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if response.Code != 200 || calls != 2 || len(result.Catalog.Repositories) != builtin.CatalogPageSize+1 || result.Catalog.Repositories[builtin.CatalogPageSize] != "team/final" {
		t.Fatalf("incomplete catalog: %d, calls=%d, entries=%d", response.Code, calls, len(result.Catalog.Repositories))
	}
}

func TestBuiltinCatalogErrorsRemainActionable(t *testing.T) {
	user := builtinTestUser(t)
	for _, test := range []struct {
		name    string
		handler http.Handler
		status  int
		message string
	}{
		{"disabled", nil, 503, "内置仓库未启用"},
		{"upstream", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(400)
			w.Write([]byte(`{"errors":[{"code":"PAGINATION_NUMBER_INVALID","message":"invalid pagination number"}]}`))
		}), 400, "invalid pagination number"},
		{"malformed", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`not json`)) }), 502, "无效的镜像列表"},
		{"stalled pagination", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Link", "<next>")
			w.Write([]byte(`{"repositories":[]}`))
		}), 502, "分页未能继续"},
	} {
		t.Run(test.name, func(t *testing.T) {
			builtin.Engine = test.handler
			response := builtinCatalogRequest(&user)
			if response.Code != test.status || !strings.Contains(response.Body.String(), test.message) || !strings.Contains(response.Body.String(), `"error"`) {
				t.Fatalf("%d %s", response.Code, response.Body.String())
			}
		})
	}
}
