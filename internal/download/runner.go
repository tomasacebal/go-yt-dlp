package download

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var (
	allowedFormats = map[string]struct{}{
		"mp3":  {},
		"mp4":  {},
		"webm": {},
	}

	allowedQualities = map[string]struct{}{
		"best":  {},
		"1080p": {},
		"720p":  {},
	}
)

func buildYTDLPArgs(jobID string, req DownloadRequest, outputDir string) ([]string, string, error) {
	if req.URL == "" {
		return nil, "", errors.New("url vacia")
	}

	if err := validateAndNormalizeFlags(&req.Flags); err != nil {
		return nil, "", err
	}

	outputTemplate := filepath.Join(outputDir, jobID, "%(title).200B [%(id)s].%(ext)s")
	args := []string{
		req.URL,
		"--progress",
		"--newline",
		"--no-colors",
		"--progress-template",
		"download:__PROGRESS__:%(progress._percent_str)s|%(progress._speed_str)s|%(progress._eta_str)s",
		"--print",
		"after_move:__FILEPATH__:%(filepath)s",
		"-o",
		outputTemplate,
	}

	if req.Flags.AudioOnly {
		args = append(args, "-x", "--audio-format", "mp3")
	} else {
		args = append(args, "-f", formatSelectorForSelection(req.Flags.Format, req.Flags.Quality), "--merge-output-format", req.Flags.Format)
	}

	if req.Flags.EmbedSubs {
		args = append(args, "--write-subs", "--embed-subs")
	}

	return args, outputTemplate, nil
}

func validateAndNormalizeFlags(flags *DownloadFlags) error {
	flags.Format = strings.TrimSpace(strings.ToLower(flags.Format))
	flags.Quality = strings.TrimSpace(strings.ToLower(flags.Quality))

	if flags.Format == "best" {
		flags.Format = "mp4"
	}

	if flags.AudioOnly {
		flags.Format = "mp3"
		flags.Quality = "best"
	}

	if flags.Quality == "" {
		flags.Quality = "best"
	}
	if _, ok := allowedQualities[flags.Quality]; !ok {
		return fmt.Errorf("quality invalida: %s", flags.Quality)
	}

	if flags.Format == "" {
		flags.Format = "mp4"
	}
	if _, ok := allowedFormats[flags.Format]; !ok {
		return fmt.Errorf("format invalido: %s", flags.Format)
	}
	return nil
}

func formatSelectorForSelection(format string, quality string) string {
	if format == "mp4" {
		return mp4FormatSelector(quality)
	}

	audioExt := "m4a"
	if format == "webm" {
		audioExt = "webm"
	}

	switch quality {
	case "1080p":
		return fmt.Sprintf("bestvideo[height<=1080][ext=%s]+bestaudio[ext=%s]/best[height<=1080][ext=%s]/best[height<=1080]", format, audioExt, format)
	case "720p":
		return fmt.Sprintf("bestvideo[height<=720][ext=%s]+bestaudio[ext=%s]/best[height<=720][ext=%s]/best[height<=720]", format, audioExt, format)
	default:
		return fmt.Sprintf("bestvideo[ext=%s]+bestaudio[ext=%s]/best[ext=%s]/best", format, audioExt, format)
	}
}

func mp4FormatSelector(quality string) string {
	videoBase := "bestvideo[ext=mp4][vcodec^=avc1]"
	progressiveBase := "best[ext=mp4][vcodec^=avc1][acodec^=mp4a]"

	switch quality {
	case "1080p":
		return fmt.Sprintf(
			"%s[height<=1080]+bestaudio[ext=m4a]/%s[height<=1080]/bestvideo[height<=1080][ext=mp4]+bestaudio[ext=m4a]/best[height<=1080][ext=mp4]/best[height<=1080]",
			videoBase,
			progressiveBase,
		)
	case "720p":
		return fmt.Sprintf(
			"%s[height<=720]+bestaudio[ext=m4a]/%s[height<=720]/bestvideo[height<=720][ext=mp4]+bestaudio[ext=m4a]/best[height<=720][ext=mp4]/best[height<=720]",
			videoBase,
			progressiveBase,
		)
	default:
		return fmt.Sprintf(
			"%s+bestaudio[ext=m4a]/%s/bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best",
			videoBase,
			progressiveBase,
		)
	}
}

func runYTDLP(ctx context.Context, bin string, args []string, onLine func(string)) error {
	cmd := exec.CommandContext(ctx, bin, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go readLines(stdout, onLine, &wg)
	go readLines(stderr, onLine, &wg)
	wg.Wait()

	return cmd.Wait()
}

func appendRuntimeArgs(args []string, cfg Config) []string {
	if cfg.FFmpegLocation != "" {
		args = append(args, "--ffmpeg-location", cfg.FFmpegLocation)
	}
	if cfg.JSRuntimes != "" {
		args = append(args, "--js-runtimes", cfg.JSRuntimes)
	}
	if cfg.EnableChromeUnlockPlugin && isChromiumBrowserCookieSource(cfg.CookiesBrowser) && cfg.PluginDir != "" {
		args = append(args, "--plugin-dirs", cfg.PluginDir)
	}
	if cfg.CookiesBrowser != "" {
		args = append(args, "--cookies-from-browser", cfg.CookiesBrowser)
	} else if cfg.CookiesFile != "" {
		args = append(args, "--cookies", cfg.CookiesFile)
	}
	return args
}

func isChromiumBrowserCookieSource(source string) bool {
	if source == "" {
		return false
	}
	browser := strings.ToLower(strings.TrimSpace(strings.SplitN(source, ":", 2)[0]))
	switch browser {
	case "chrome", "chromium", "edge", "brave", "opera", "vivaldi":
		return true
	default:
		return false
	}
}

func readLines(reader io.Reader, onLine func(string), wg *sync.WaitGroup) {
	defer wg.Done()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		onLine(scanner.Text())
	}
}
