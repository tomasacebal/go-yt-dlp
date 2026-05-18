package download

import "testing"

func TestParseProgressLine(t *testing.T) {
	event, ok := parseProgressLine("__PROGRESS__: 45.5%|2.5MiB/s|00:30")
	if !ok {
		t.Fatalf("se esperaba linea parseable")
	}
	if event.Status != JobStatusDownloading {
		t.Fatalf("status inesperado: %s", event.Status)
	}
	if event.Progress != 45.5 {
		t.Fatalf("progress inesperado: %f", event.Progress)
	}
	if event.Speed != "2.5MiB/s" {
		t.Fatalf("speed inesperada: %s", event.Speed)
	}
	if event.ETA != "00:30" {
		t.Fatalf("eta inesperada: %s", event.ETA)
	}
}

func TestParseProgressLineInvalid(t *testing.T) {
	_, ok := parseProgressLine("[download] 35%")
	if ok {
		t.Fatalf("no se esperaba parseo valido")
	}
}
