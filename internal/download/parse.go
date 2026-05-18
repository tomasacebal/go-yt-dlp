package download

import (
	"regexp"
	"strconv"
	"strings"
)

var progressLinePattern = regexp.MustCompile(`^__PROGRESS__:(.+)\|(.+)\|(.+)$`)

func parseProgressLine(line string) (ProgressEvent, bool) {
	match := progressLinePattern.FindStringSubmatch(strings.TrimSpace(line))
	if len(match) != 4 {
		return ProgressEvent{}, false
	}

	percent := parsePercent(match[1])
	speed := strings.TrimSpace(match[2])
	eta := strings.TrimSpace(match[3])
	if speed == "NA" {
		speed = ""
	}
	if eta == "NA" {
		eta = ""
	}

	return ProgressEvent{
		Status:   JobStatusDownloading,
		Progress: percent,
		Speed:    speed,
		ETA:      eta,
	}, true
}

func parsePercent(raw string) float64 {
	clean := strings.TrimSpace(raw)
	clean = strings.TrimSuffix(clean, "%")
	clean = strings.TrimSpace(clean)
	if clean == "" || clean == "NA" {
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
