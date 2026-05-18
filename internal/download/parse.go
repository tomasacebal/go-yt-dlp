package download

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	progressLinePattern = regexp.MustCompile(`__PROGRESS__:(.+)\|(.+)\|(.+)`)
	legacyProgressRegex = regexp.MustCompile(`(?i)\[download\]\s+([0-9]+(?:\.[0-9]+)?)%\s+.*?\sat\s+(.+?)\s+ETA\s+(.+)$`)
	ansiEscapeRegex     = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
)

func parseProgressLine(line string) (ProgressEvent, bool) {
	clean := sanitizeLogLine(line)

	match := progressLinePattern.FindStringSubmatch(clean)
	if len(match) != 4 {
		return parseLegacyProgressLine(clean)
	}

	percent := parsePercent(match[1])
	speed := normalizeProgressField(match[2])
	eta := normalizeProgressField(match[3])
	return makeProgressEvent(percent, speed, eta), true
}

func parseLegacyProgressLine(clean string) (ProgressEvent, bool) {
	match := legacyProgressRegex.FindStringSubmatch(clean)
	if len(match) != 4 {
		return ProgressEvent{}, false
	}

	percent := parsePercent(match[1] + "%")
	speed := normalizeProgressField(match[2])
	eta := normalizeProgressField(match[3])
	return makeProgressEvent(percent, speed, eta), true
}

func makeProgressEvent(percent float64, speed, eta string) ProgressEvent {
	return ProgressEvent{
		Status:   JobStatusDownloading,
		Progress: percent,
		Speed:    speed,
		ETA:      eta,
	}
}

func normalizeProgressField(raw string) string {
	value := strings.TrimSpace(raw)
	upper := strings.ToUpper(value)
	if upper == "NA" || upper == "UNKNOWN" {
		return ""
	}
	return value
}

func sanitizeLogLine(line string) string {
	clean := strings.ReplaceAll(line, "\r", "")
	clean = ansiEscapeRegex.ReplaceAllString(clean, "")
	return strings.TrimSpace(clean)
}

func parsePercent(raw string) float64 {
	clean := strings.TrimSpace(raw)
	clean = strings.TrimSuffix(clean, "%")
	clean = strings.TrimSpace(clean)
	if clean == "" || strings.EqualFold(clean, "NA") {
		return 0
	}

	value, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		return 0
	}
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}
