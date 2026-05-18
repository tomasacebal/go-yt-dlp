package download

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnvSetsMissingVar(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")

	content := "LISTEN_ADDR=:9090\n# comentario\n"
	if err := os.WriteFile(envPath, []byte(content), 0o644); err != nil {
		t.Fatalf("no se pudo escribir .env de test: %v", err)
	}

	t.Setenv("LISTEN_ADDR", "")
	_ = os.Unsetenv("LISTEN_ADDR")

	if err := loadDotEnv(envPath); err != nil {
		t.Fatalf("loadDotEnv fallo: %v", err)
	}

	if got := os.Getenv("LISTEN_ADDR"); got != ":9090" {
		t.Fatalf("LISTEN_ADDR inesperado: %s", got)
	}
}

func TestLoadDotEnvDoesNotOverrideExistingVar(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")

	content := "LISTEN_ADDR=:9090\n"
	if err := os.WriteFile(envPath, []byte(content), 0o644); err != nil {
		t.Fatalf("no se pudo escribir .env de test: %v", err)
	}

	t.Setenv("LISTEN_ADDR", ":7070")

	if err := loadDotEnv(envPath); err != nil {
		t.Fatalf("loadDotEnv fallo: %v", err)
	}

	if got := os.Getenv("LISTEN_ADDR"); got != ":7070" {
		t.Fatalf("LISTEN_ADDR fue sobreescrito: %s", got)
	}
}
