package backup_test

import (
	"context"
	"dockerun/internal/backup"
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"dockerun/internal/utils"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSQLiteCompleteBackupRestore(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DATA_DIR", dir)
	if err := config.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := database.Init(config.Load().DatabasePath); err != nil {
		t.Fatal(err)
	}
	sealed, err := utils.SealCredential("external-password")
	if err != nil {
		t.Fatal(err)
	}
	database.DB.Create(&models.Registry{Name: "external", Kind: "external", URL: "https://registry.example", Password: sealed})
	os.WriteFile(filepath.Join(dir, "registry", "fixture"), []byte("registry-material"), 0600)
	os.WriteFile(filepath.Join(dir, "uploads", "fixture.tar.gz"), []byte("offline-material"), 0600)
	path, err := backup.Create(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	restored := filepath.Join(t.TempDir(), "restored")
	if err = backup.Restore(path, restored); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"dockerun.db", "keys/credentials.key", "registry/fixture", "uploads/fixture.tar.gz", "backup.json"} {
		if _, err = os.Stat(filepath.Join(restored, name)); err != nil {
			t.Fatal(name, err)
		}
	}
	t.Setenv("DATA_DIR", restored)
	if plain, err := utils.OpenCredential(sealed); err != nil || plain != "external-password" {
		t.Fatalf("credential restore %v", err)
	}
	if err = backup.Restore(path, restored); err == nil {
		t.Fatal("restore overwrote existing data")
	}
}

func TestBackupCancellationWhileDeploymentRuns(t *testing.T) {
	release.ExecutionGate.RLock()
	defer release.ExecutionGate.RUnlock()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := backup.Create(ctx); err != context.DeadlineExceeded {
		t.Fatalf("waiting backup ignored cancellation: %v", err)
	}
}
