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

	outputTemplate := filepath.Join(outputDir, fmt.Sprintf("%s.%%(ext)s", jobID))
	args := []string{
		req.URL,
		"--newline",
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
		args = append(args, "-f", formatSelectorForQuality(req.Flags.Quality), "--merge-output-format", "mkv")
	}

	if req.Flags.EmbedSubs {
		args = append(args, "--write-subs", "--embed-subs")
	}

	return args, outputTemplate, nil
}

func validateAndNormalizeFlags(flags *DownloadFlags) error {
	flags.Format = strings.TrimSpace(strings.ToLower(flags.Format))
	flags.Quality = strings.TrimSpace(strings.ToLower(flags.Quality))

	if flags.Quality == "" {
		flags.Quality = "best"
	}
	if _, ok := allowedQualities[flags.Quality]; !ok {
		return fmt.Errorf("quality invalida: %s", flags.Quality)
	}

	if flags.Format == "" {
		flags.Format = "best"
	}
	if flags.Format != "best" {
		return fmt.Errorf("format invalido: %s", flags.Format)
	}
	return nil
}

func formatSelectorForQuality(quality string) string {
	switch quality {
	case "1080p":
		return "bestvideo[height<=1080][ext=mp4]+bestaudio[ext=m4a]/best[height<=1080]"
	case "720p":
		return "bestvideo[height<=720][ext=mp4]+bestaudio[ext=m4a]/best[height<=720]"
	default:
		return "bestvideo[ext=mp4]+bestaudio[ext=m4a]/best"
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

func readLines(reader io.Reader, onLine func(string), wg *sync.WaitGroup) {
	defer wg.Done()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		onLine(scanner.Text())
	}
}
