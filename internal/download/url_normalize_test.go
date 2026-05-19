package download

import "testing"

func TestNormalizeDownloadURLRemovesListFromYouTubeWatchURL(t *testing.T) {
	raw := "https://www.youtube.com/watch?v=abc123&list=PL123&t=10"

	got := normalizeDownloadURL(raw)

	if got != "https://www.youtube.com/watch?t=10&v=abc123" {
		t.Fatalf("url inesperada: %s", got)
	}
}

func TestNormalizeDownloadURLKeepsNonYouTubeURLUntouched(t *testing.T) {
	raw := "https://example.com/watch?v=abc123&list=PL123"

	got := normalizeDownloadURL(raw)

	if got != raw {
		t.Fatalf("la url no debia cambiar: %s", got)
	}
}

func TestNormalizeDownloadURLHandlesInvalidInput(t *testing.T) {
	raw := "no es una url"

	got := normalizeDownloadURL(raw)

	if got != raw {
		t.Fatalf("la url invalida no debia cambiar: %s", got)
	}
}
