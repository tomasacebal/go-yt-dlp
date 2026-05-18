package download

import (
	"path/filepath"
	"testing"
)

func TestBuildYTDLPArgsVideo(t *testing.T) {
	req := DownloadRequest{
		URL: "https://www.youtube.com/watch?v=abc123",
		Flags: DownloadFlags{
			Format:    "best",
			AudioOnly: false,
			Quality:   "1080p",
		},
	}

	args, outputTemplate, err := buildYTDLPArgs("job-1", req, "data/downloads")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if outputTemplate != filepath.Join("data/downloads", "job-1", "%(title).200B [%(id)s].%(ext)s") {
		t.Fatalf("output template inesperado: %s", outputTemplate)
	}
	assertContains(t, args, "--no-colors")
	assertContains(t, args, "--merge-output-format")
	assertContains(t, args, "mkv")
	assertContains(t, args, "-f")
	assertContains(t, args, "bestvideo[height<=1080][ext=mp4]+bestaudio[ext=m4a]/best[height<=1080]")
}

func TestBuildYTDLPArgsAudioOnly(t *testing.T) {
	req := DownloadRequest{
		URL: "https://www.youtube.com/watch?v=abc123",
		Flags: DownloadFlags{
			Format:    "best",
			AudioOnly: true,
			Quality:   "best",
		},
	}

	args, _, err := buildYTDLPArgs("job-2", req, "data/downloads")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	assertContains(t, args, "-x")
	assertContains(t, args, "--audio-format")
	assertContains(t, args, "mp3")
}

func TestValidateAndNormalizeFlagsRejectsInvalidQuality(t *testing.T) {
	flags := DownloadFlags{
		Format:  "best",
		Quality: "4k",
	}

	if err := validateAndNormalizeFlags(&flags); err == nil {
		t.Fatalf("se esperaba error por quality invalida")
	}
}

func TestValidateAndNormalizeFlagsAudioOnlyForcesBestQuality(t *testing.T) {
	flags := DownloadFlags{
		Format:    "best",
		AudioOnly: true,
		Quality:   "1080p",
	}

	if err := validateAndNormalizeFlags(&flags); err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if flags.Quality != "best" {
		t.Fatalf("quality esperada best, recibida %s", flags.Quality)
	}
}

func assertContains(t *testing.T, values []string, target string) {
	t.Helper()
	for _, value := range values {
		if value == target {
			return
		}
	}
	t.Fatalf("no se encontro %q en args", target)
}
