package handler

import (
	"context"
	"dockerun/internal/models"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRegistryImagesOnlyReadsCurrentPageAndIsolatesTagFailures(t *testing.T) {
	db := registryTestDB(t)
	var mutex sync.Mutex
	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mutex.Lock()
		requests[r.URL.Path]++
		attempt := requests[r.URL.Path]
		mutex.Unlock()
		switch r.URL.Path {
		case "/v2/":
			fmt.Fprint(w, `{}`)
		case "/v2/_catalog":
			fmt.Fprint(w, `{"repositories":["app-1","app-2","lucaslorentz/caddy-docker-proxy","team/app-4","app-5"]}`)
		case "/v2/lucaslorentz/caddy-docker-proxy/tags/list":
			if attempt == 1 {
				w.WriteHeader(502)
				fmt.Fprint(w, `upstream unavailable`)
			} else {
				fmt.Fprint(w, `{"tags":["latest"]}`)
			}
		default:
			fmt.Fprint(w, `{"tags":["v1","v2"]}`)
		}
	}))
	defer server.Close()
	registry := models.Registry{Name: "external", Kind: "external", URL: server.URL}
	if err := db.Create(&registry).Error; err != nil {
		t.Fatal(err)
	}
	if response := registryRequest(t, TestRegistry, 0, map[string]any{"url": server.URL}); response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	response := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(response)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(registry.ID)}}
	ctx.Request = httptest.NewRequest("GET", "/images?page=2&page_size=2", nil)
	ListRegistryImages(ctx)
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	var result struct {
		Items []RegistryImage `json:"items"`
		Total int             `json:"total"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Total != 5 || len(result.Items) != 2 || result.Items[0].Name != "lucaslorentz/caddy-docker-proxy" || result.Items[1].Name != "team/app-4" {
		t.Fatalf("wrong page: %+v", result)
	}
	if result.Items[0].TagsError == "" || len(result.Items[0].Tags) != 0 || result.Items[1].TagsError != "" || len(result.Items[1].Tags) != 2 {
		t.Fatalf("single failure must not discard successful rows: %+v", result.Items)
	}
	mutex.Lock()
	for _, name := range []string{"app-1", "app-2", "app-5"} {
		if requests["/v2/"+name+"/tags/list"] != 0 {
			t.Errorf("fetched off-page tags for %s", name)
		}
	}
	mutex.Unlock()
	// The failed repository can be retried independently through its existing endpoint.
	retry := httptest.NewRecorder()
	retryCtx, _ := gin.CreateTestContext(retry)
	retryCtx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(registry.ID)}, {Key: "name", Value: "/lucaslorentz/caddy-docker-proxy/tags"}}
	retryCtx.Request = httptest.NewRequest("GET", "/images/lucaslorentz/caddy-docker-proxy/tags", nil)
	GetRegistryImageTags(retryCtx)
	if retry.Code != 200 || !strings.Contains(retry.Body.String(), "latest") {
		t.Fatalf("tag retry failed: %s", retry.Body.String())
	}
	for _, page := range []string{"4", "9223372036854775807"} {
		out := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(out)
		c.Params = ctx.Params
		c.Request = httptest.NewRequest("GET", "/images?page="+page+"&page_size=2", nil)
		ListRegistryImages(c)
		if out.Code != 200 || !strings.Contains(out.Body.String(), `"items":[]`) {
			t.Fatalf("out-of-range page must be empty: %s", out.Body.String())
		}
	}
}

func TestRegistryCatalogAndTagsFollowPaginationLinks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/_catalog" {
			if r.URL.Query().Get("last") == "" {
				w.Header().Set("Link", `</v2/_catalog?n=2&last=team%2Fapp-2>; rel="next"`)
				fmt.Fprint(w, `{"repositories":["team/app-1","team/app-2"]}`)
			} else {
				fmt.Fprint(w, `{"repositories":["team/app-3"]}`)
			}
			return
		}
		if r.URL.Query().Get("last") == "" {
			w.Header().Set("Link", `<?n=2&last=v2>; rel="next"`)
			fmt.Fprint(w, `{"tags":["v1","v2"]}`)
		} else {
			fmt.Fprint(w, `{"tags":["v2","v3"]}`)
		}
	}))
	defer server.Close()
	registry := models.Registry{URL: server.URL}
	names, err := fetchRegistryRepositories(context.Background(), registry, " TEAM/APP ")
	if err != nil || len(names) != 3 {
		t.Fatalf("catalog pages truncated: %v %v", names, err)
	}
	tags, err := fetchImageTags(context.Background(), registry, "team/app-1")
	if err != nil || strings.Join(tags, ",") != "v1,v2,v3" {
		t.Fatalf("tag pages truncated or duplicated: %v %v", tags, err)
	}
	for _, link := range []string{`<https://other.test/v2/_catalog?n=2>; rel="next"`, `</v2/private?n=2>; rel="next"`} {
		_, err := nextRegistryPage(registry, "/v2/_catalog?n=2", http.Header{"Link": []string{link}})
		if err == nil {
			t.Fatalf("accepted unsafe pagination link %s", link)
		}
	}
}

func TestRegistryTagLoadingIsBoundedAndHonorsCancellation(t *testing.T) {
	var active, maximum atomic.Int32
	started := make(chan struct{}, 8)
	gate := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := active.Add(1)
		defer active.Add(-1)
		for previous := maximum.Load(); count > previous; previous = maximum.Load() {
			if maximum.CompareAndSwap(previous, count) {
				break
			}
		}
		started <- struct{}{}
		select {
		case <-gate:
			fmt.Fprint(w, `{"tags":["latest"]}`)
		case <-r.Context().Done():
		}
	}))
	defer server.Close()
	names := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	done := make(chan []RegistryImage, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { done <- fetchRegistryPageTags(ctx, models.Registry{URL: server.URL}, names) }()
	for i := 0; i < 4; i++ {
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatal("tag workers did not start")
		}
	}
	cancel()
	select {
	case items := <-done:
		for _, item := range items {
			if item.TagsError == "" {
				t.Errorf("cancelled item lost error state: %+v", item)
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("tag loading ignored request cancellation")
	}
	if maximum.Load() != 4 {
		t.Fatalf("expected at most four parallel reads, got %d", maximum.Load())
	}
	close(gate)
}
