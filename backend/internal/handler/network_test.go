package handler

import (
	"dockerun/internal/docker"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
)

// Exercise the actual handler -> Docker client contract without modifying host networks.
func TestNetworkWorkflow(t *testing.T) {
	var created struct {
		Name string
		types.NetworkCreate
	}
	var connected, disconnected bool
	transport := containerTransport(func(request *http.Request) (*http.Response, error) {
		response := httptest.NewRecorder()
		response.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == "POST" && strings.HasSuffix(request.URL.Path, "/networks/create"):
			if err := json.NewDecoder(request.Body).Decode(&created); err != nil {
				t.Fatal(err)
			}
			response.WriteHeader(201)
			io.WriteString(response, `{"Id":"test-id","Warning":""}`)
		case strings.HasSuffix(request.URL.Path, "/connect") || strings.HasSuffix(request.URL.Path, "/disconnect"):
			var payload struct {
				Container string
				Force     bool
			}
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload.Container != "test-container" || payload.Force {
				t.Errorf("unexpected connection payload: %+v", payload)
			}
			if strings.HasSuffix(request.URL.Path, "/disconnect") {
				disconnected = true
			} else {
				connected = true
			}
		case request.Method == "DELETE":
			response.WriteHeader(409)
			io.WriteString(response, `{"message":"network has active endpoints"}`)
		case strings.HasSuffix(request.URL.Path, "/test-id"):
			io.WriteString(response, `{"Id":"test-id","Name":"test-net","Driver":"bridge","IPAM":{"Driver":"default","Config":[{"Subnet":"172.30.0.0/16","Gateway":"172.30.0.1"}]},"Containers":{"test-container":{"Name":"web","IPv4Address":"172.30.0.2/16"}}}`)
		default:
			response.WriteHeader(404)
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
	router.POST("/networks", CreateNetwork)
	router.GET("/networks/:id", InspectNetwork)
	router.POST("/networks/:id/connect", ConnectNetwork)
	router.POST("/networks/:id/disconnect", DisconnectNetwork)
	router.DELETE("/networks/:id", RemoveNetwork)
	perform := func(method, path, body string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(method, path, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(response, request)
		return response
	}
	response := perform("POST", "/networks", `{"name":"test-net","driver":"bridge","internal":true,"attachable":true,"enable_ipv6":true,"options":{"com.docker.network.driver.mtu":"1400"},"labels":{"owner":"test"},"ipam":{"Driver":"default","Config":[{"Subnet":"172.30.0.0/16","Gateway":"172.30.0.1","IPRange":"172.30.1.0/24","AuxiliaryAddresses":{"reserved":"172.30.0.10"}},{"Subnet":"fd30::/64","Gateway":"fd30::1"}]}}`)
	if response.Code != 200 || !strings.Contains(response.Body.String(), `"Id":"test-id"`) {
		t.Fatalf("create: %d %s", response.Code, response.Body.String())
	}
	if created.Name != "test-net" || created.Driver != "bridge" || !created.Internal || !created.Attachable || !created.EnableIPv6 || created.Options["com.docker.network.driver.mtu"] != "1400" || created.Labels["owner"] != "test" {
		t.Fatalf("creation options lost: %+v", created)
	}
	if created.IPAM == nil || len(created.IPAM.Config) != 2 || created.IPAM.Config[0].Subnet != "172.30.0.0/16" || created.IPAM.Config[0].Gateway != "172.30.0.1" || created.IPAM.Config[0].IPRange != "172.30.1.0/24" || created.IPAM.Config[0].AuxAddress["reserved"] != "172.30.0.10" || created.IPAM.Config[1].Gateway != "fd30::1" {
		t.Fatalf("IPAM options lost: %+v", created.IPAM)
	}
	response = perform("GET", "/networks/test-id", "")
	if response.Code != 200 || !strings.Contains(response.Body.String(), `"IPv4Address":"172.30.0.2/16"`) {
		t.Fatalf("inspect: %d %s", response.Code, response.Body.String())
	}
	for _, action := range []string{"connect", "disconnect"} {
		response = perform("POST", "/networks/test-id/"+action, `{"container_id":"test-container"}`)
		if response.Code != 200 {
			t.Fatalf("%s: %d %s", action, response.Code, response.Body.String())
		}
	}
	if !connected || !disconnected {
		t.Fatal("connection requests did not reach Docker")
	}
	response = perform("DELETE", "/networks/test-id", "")
	if response.Code == 200 || !strings.Contains(response.Body.String(), "active endpoints") {
		t.Fatalf("active network removal must expose Docker error: %d %s", response.Code, response.Body.String())
	}
	response = perform("POST", "/networks", `{"driver":"bridge"}`)
	if response.Code != 400 {
		t.Fatalf("missing name must fail: %d", response.Code)
	}
}
