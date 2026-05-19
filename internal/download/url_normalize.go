package download

import (
	"net/url"
	"strings"
)

func normalizeDownloadURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	parsed, err := url.ParseRequestURI(trimmed)
	if err != nil {
		return trimmed
	}
	if !isYouTubeURLHost(parsed.Hostname()) {
		return trimmed
	}

	values := parsed.Query()
	values.Del("list")
	parsed.RawQuery = values.Encode()
	return parsed.String()
}

func isYouTubeURLHost(host string) bool {
	normalized := strings.ToLower(host)
	return normalized == "youtube.com" ||
		strings.HasSuffix(normalized, ".youtube.com") ||
		normalized == "youtu.be" ||
		strings.HasSuffix(normalized, ".youtu.be") ||
		normalized == "youtube-nocookie.com" ||
		strings.HasSuffix(normalized, ".youtube-nocookie.com")
}
