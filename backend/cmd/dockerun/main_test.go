package main

import (
	"strings"
	"testing"
)

func TestBuildKeepsServerParameters(t *testing.T) {
	source := []byte("services:\n  app:\n    image: ${IMAGE}\n    build: .\n    environment:\n      PASSWORD: ${PASSWORD:?required}\n")
	local := []byte(`{"services":{"app":{"image":"registry.example/team/app:commit","environment":{"PASSWORD":"LOCAL_SECRET"},"build":{"context":"/developer/path"}}}}`)
	out, err := publicationDefinition(source, local)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "LOCAL_SECRET") || strings.Contains(out, "/developer/path") || !strings.Contains(out, "${PASSWORD:?required}") || !strings.Contains(out, "registry.example/team/app:commit") {
		t.Fatal("local runtime configuration leaked into release:", out)
	}
}
