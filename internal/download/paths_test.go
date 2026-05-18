package download

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInferFinalPathFallbackSupportsLegacyAndCurrent(t *testing.T) {
	dir := t.TempDir()
	jobID := "job-123"

	currentDir := filepath.Join(dir, jobID)
	if err := os.MkdirAll(currentDir, 0o755); err != nil {
		t.Fatalf("mkdir fallo: %v", err)
	}

	currentFile := filepath.Join(currentDir, "video.mp4")
	if err := os.WriteFile(currentFile, []byte("ok"), 0o644); err != nil {
		t.Fatalf("write current fallo: %v", err)
	}

	got := inferFinalPathFallback(dir, jobID)
	if got != currentFile {
		t.Fatalf("ruta actual inesperada: %s", got)
	}

	legacyJob := "job-legacy"
	legacyFile := filepath.Join(dir, legacyJob+".mkv")
	if err := os.WriteFile(legacyFile, []byte("ok"), 0o644); err != nil {
		t.Fatalf("write legacy fallo: %v", err)
	}

	gotLegacy := inferFinalPathFallback(dir, legacyJob)
	if gotLegacy != legacyFile {
		t.Fatalf("ruta legacy inesperada: %s", gotLegacy)
	}
}

func TestResolveFinalPathPrefersHintedWhenExists(t *testing.T) {
	dir := t.TempDir()
	jobID := "job-999"
	hinted := filepath.Join(dir, "hinted.mp3")
	if err := os.WriteFile(hinted, []byte("ok"), 0o644); err != nil {
		t.Fatalf("write hinted fallo: %v", err)
	}

	got := resolveFinalPath(dir, jobID, hinted)
	if got != hinted {
		t.Fatalf("ruta hinted inesperada: %s", got)
	}
}

func TestPickNewestUsableFileSkipsTempFiles(t *testing.T) {
	dir := t.TempDir()
	part := filepath.Join(dir, "a.part")
	final := filepath.Join(dir, "b.mp4")
	if err := os.WriteFile(part, []byte("tmp"), 0o644); err != nil {
		t.Fatalf("write part fallo: %v", err)
	}
	time.Sleep(15 * time.Millisecond)
	if err := os.WriteFile(final, []byte("ok"), 0o644); err != nil {
		t.Fatalf("write final fallo: %v", err)
	}

	got := pickNewestUsableFile([]string{part, final})
	if got != final {
		t.Fatalf("archivo elegido inesperado: %s", got)
	}
}
