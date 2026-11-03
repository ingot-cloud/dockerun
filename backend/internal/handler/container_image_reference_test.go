package handler

import (
	"dockerun/internal/docker"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
)

func TestContainerImageNamesAreVerifiedAgainstActualIDs(t *testing.T) {
	db := registryTestDB(t)
	if err := db.AutoMigrate(&models.TargetState{}); err != nil {
		t.Fatal(err)
	}
	for _, state := range []models.TargetState{
		{EnvironmentID: 1, TargetID: 1, Containers: []models.RuntimeContainer{{ID: "old", Image: "registry.test/redis:8.2.0", ImageID: "sha256:old"}, {ID: "changed", Image: "registry.test/wrong:v1", ImageID: "sha256:before"}}},
		{EnvironmentID: 2, TargetID: 2, Containers: []models.RuntimeContainer{{ID: "remote", Image: "registry.test/remote:v1", ImageID: "sha256:remote"}}},
	} {
		if err := db.Create(&state).Error; err != nil {
			t.Fatal(err)
		}
	}
	rows := []types.Container{
		{ID: "new", Image: "sha256:new", ImageID: "sha256:new", Labels: map[string]string{release.ImageReferenceLabel: "registry.test/new:v1", release.ImageIDLabel: "sha256:new"}},
		{ID: "old", Image: "sha256:old", ImageID: "sha256:old"},
		{ID: "changed", Image: "sha256:changed", ImageID: "sha256:changed", Labels: map[string]string{release.ImageReferenceLabel: "registry.test/wrong:v1", release.ImageIDLabel: "sha256:before"}},
		{ID: "remote", Image: "sha256:remote", ImageID: "sha256:remote"},
		{ID: "unique", Image: "sha256:unique", ImageID: "sha256:unique"},
		{ID: "same", Image: "sha256:unique", ImageID: "sha256:unique"},
		{ID: "ambiguous", Image: "sha256:ambiguous", ImageID: "sha256:ambiguous"},
		{ID: "literal", Image: "redis:latest", ImageID: "sha256:literal"},
	}
	calls := map[string]int{}
	transport := containerTransport(func(request *http.Request) (*http.Response, error) {
		response := httptest.NewRecorder()
		response.Header().Set("Content-Type", "application/json")
		switch {
		case request.URL.Path == "/v1.41/containers/json":
			json.NewEncoder(response).Encode(rows)
		case strings.Contains(request.URL.Path, "/images/"):
			id := strings.TrimSuffix(strings.Split(request.URL.Path, "/images/")[1], "/json")
			calls[id]++
			switch id {
			case "sha256:unique":
				json.NewEncoder(response).Encode(types.ImageInspect{ID: id, RepoTags: []string{"registry.test/unique:v1"}})
			case "sha256:ambiguous":
				json.NewEncoder(response).Encode(types.ImageInspect{ID: id, RepoTags: []string{"first:v1", "second:v1"}})
			default:
				response.WriteHeader(http.StatusNotFound)
				fmt.Fprint(response, `{"message":"image unavailable"}`)
			}
		case request.URL.Path == "/v1.41/containers/new/json":
			json.NewEncoder(response).Encode(types.ContainerJSON{ContainerJSONBase: &types.ContainerJSONBase{ID: "new", Image: "sha256:new"}, Config: &container.Config{Image: "sha256:new", Labels: rows[0].Labels}})
		default:
			response.WriteHeader(http.StatusNotFound)
		}
		return response.Result(), nil
	})
	cli, err := client.NewClientWithOpts(client.WithHost("http://docker.test"), client.WithVersion("1.41"), client.WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatal(err)
	}
	previous := docker.Client
	docker.Client = cli
	t.Cleanup(func() { docker.Client = previous; cli.Close() })
	router := gin.New()
	router.GET("/containers", ListContainers)
	router.GET("/containers/:id", InspectContainer)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/containers", nil))
	var named []containerWithImageReference
	if err := json.Unmarshal(response.Body.Bytes(), &named); err != nil || response.Code != http.StatusOK || len(named) != len(rows) {
		t.Fatal(response.Body.String(), err)
	}
	want := map[string]string{"new": "registry.test/new:v1", "old": "registry.test/redis:8.2.0", "unique": "registry.test/unique:v1", "same": "registry.test/unique:v1"}
	for i, row := range named {
		if row.ImageReference != want[row.ID] || row.Image != rows[i].Image || row.ImageID != rows[i].ImageID {
			t.Fatalf("镜像名称错误或原始字段被改写：%+v", row)
		}
	}
	if calls["sha256:unique"] != 1 || calls["sha256:new"] != 0 || calls["sha256:old"] != 0 || calls["sha256:literal"] != 0 {
		t.Fatal("镜像查询没有复用缓存或重复查询已核对的名称", calls)
	}
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/containers/new", nil))
	var detail struct {
		types.ContainerJSON
		ImageReference string
	}
	if err := json.Unmarshal(response.Body.Bytes(), &detail); err != nil || response.Code != http.StatusOK || detail.ImageReference != "registry.test/new:v1" || detail.Config.Image != "sha256:new" || detail.Image != "sha256:new" {
		t.Fatal("详情名称与实际 Image ID 不一致", response.Body.String(), err)
	}
}
