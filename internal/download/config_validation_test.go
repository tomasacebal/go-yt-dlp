package download

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateConfigCookiesFileMissing(t *testing.T) {
	cfg := LoadConfig()
	cfg.CookiesFile = filepath.Join(t.TempDir(), "no-existe.txt")

	if err := ValidateConfig(cfg); err == nil {
		t.Fatalf("se esperaba error por archivo de cookies inexistente")
	}
}

func TestValidateConfigCookiesFileInvalidFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cookies.txt")
	if err := os.WriteFile(path, []byte("esto no es netscape\n"), 0o644); err != nil {
		t.Fatalf("no se pudo crear archivo: %v", err)
	}

	cfg := LoadConfig()
	cfg.CookiesFile = path

	if err := ValidateConfig(cfg); err == nil {
		t.Fatalf("se esperaba error por formato invalido")
	}
}

func TestValidateConfigCookiesFileValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cookies.txt")
	content := "# Netscape HTTP Cookie File\n.youtube.com\tTRUE\t/\tTRUE\t2147483647\tSID\tvalor\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("no se pudo crear archivo: %v", err)
	}

	cfg := LoadConfig()
	cfg.CookiesFile = path

	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
}
