package release

import (
	"dockerun/internal/models"
	"strings"
	"testing"
)

func TestScriptLogRedactsSecretsAcrossWrites(t *testing.T) {
	s := Snapshot{Environment: models.Environment{Parameters: models.StringMap{"PASSWORD": "split-secret", "CUSTOM": "custom-credential", "PORT": "8080"}}, SensitiveParameters: []string{"CUSTOM"}}
	var output strings.Builder
	w := &scriptLogWriter{secrets: scriptSecrets(s), log: func(value string) { output.WriteString(value) }}
	for _, part := range []string{"value split-", "secret custom-", "credential\nport 8080\nlast split-", "secret"} {
		w.Write([]byte(part))
	}
	w.Flush()
	if output.String() != "value ****** ******\nport 8080\nlast ******" {
		t.Fatalf("sensitive value leaked or ordinary output changed: %q", output.String())
	}
	output.Reset()
	w.Write([]byte(strings.Repeat("x", 1<<20+1)))
	w.Write([]byte("split-secret\nnext\n"))
	w.Flush()
	if strings.Contains(output.String(), "split-secret") || !strings.HasSuffix(output.String(), "next\n") {
		t.Fatal("long log handling leaked content or lost following lines")
	}
}
