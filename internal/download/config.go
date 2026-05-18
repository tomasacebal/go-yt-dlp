package download

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	defaultListenAddr     = ":8080"
	defaultWorkerCount    = 2
	defaultQueueCapacity  = 20
	defaultDownloadDir    = "data/downloads"
	defaultYTDLPBin       = "yt-dlp"
	defaultJSRuntimes     = "deno,node"
	defaultFFmpegLocation = `C:\Shared\ffmpeg\bin`
	defaultCleanupEvery   = time.Hour
	defaultFileTTL        = 24 * time.Hour
	defaultMaxBodyBytes   = 1 * 1024 * 1024
	defaultWSWriteTimeout = 5 * time.Second
)

// Config representa la configuracion operativa del servicio.
type Config struct {
	ListenAddr     string
	WorkerCount    int
	QueueCapacity  int
	DownloadDir    string
	YTDLPBin       string
	JSRuntimes     string
	FFmpegLocation string
	CleanupEvery   time.Duration
	FileTTL        time.Duration
	MaxBodyBytes   int
	WSWriteTimeout time.Duration
}

// LoadConfig carga configuracion desde variables de entorno con defaults.
//
// Returns:
//
//	Config: configuracion final normalizada.
func LoadConfig() Config {
	_ = loadDotEnv(".env")
	_ = loadDotEnv(".env.local")

	cfg := Config{
		ListenAddr:     getEnv("LISTEN_ADDR", defaultListenAddr),
		WorkerCount:    getEnvInt("WORKER_COUNT", defaultWorkerCount),
		QueueCapacity:  getEnvInt("QUEUE_CAPACITY", defaultQueueCapacity),
		DownloadDir:    getEnv("DOWNLOAD_DIR", defaultDownloadDir),
		YTDLPBin:       getEnv("YTDLP_BIN", defaultYTDLPBin),
		JSRuntimes:     getEnv("YTDLP_JS_RUNTIMES", defaultJSRuntimes),
		FFmpegLocation: getEnv("FFMPEG_LOCATION", defaultFFmpegLocation),
		CleanupEvery:   getEnvDuration("CLEANUP_EVERY", defaultCleanupEvery),
		FileTTL:        getEnvDuration("FILE_TTL", defaultFileTTL),
		MaxBodyBytes:   getEnvInt("MAX_BODY_BYTES", defaultMaxBodyBytes),
		WSWriteTimeout: getEnvDuration("WS_WRITE_TIMEOUT", defaultWSWriteTimeout),
	}

	if cfg.WorkerCount <= 0 {
		cfg.WorkerCount = defaultWorkerCount
	}
	if cfg.QueueCapacity <= 0 {
		cfg.QueueCapacity = defaultQueueCapacity
	}
	if cfg.MaxBodyBytes <= 0 {
		cfg.MaxBodyBytes = defaultMaxBodyBytes
	}
	if cfg.CleanupEvery <= 0 {
		cfg.CleanupEvery = defaultCleanupEvery
	}
	if cfg.FileTTL <= 0 {
		cfg.FileTTL = defaultFileTTL
	}
	if cfg.WSWriteTimeout <= 0 {
		cfg.WSWriteTimeout = defaultWSWriteTimeout
	}
	if cfg.YTDLPBin == "" {
		cfg.YTDLPBin = defaultYTDLPBin
	}
	cfg.YTDLPBin = resolveYTDLPBin(cfg.YTDLPBin)
	cfg.JSRuntimes = strings.TrimSpace(cfg.JSRuntimes)
	cfg.FFmpegLocation = strings.TrimSpace(cfg.FFmpegLocation)
	if cfg.DownloadDir == "" {
		cfg.DownloadDir = defaultDownloadDir
	}
	return cfg
}

func resolveYTDLPBin(current string) string {
	trimmed := current
	if trimmed != "" && trimmed != defaultYTDLPBin {
		return trimmed
	}

	if runtime.GOOS == "windows" {
		localExe := filepath.Clean("yt-dlp.exe")
		if fileExists(localExe) {
			return localExe
		}
	}

	localBin := filepath.Clean("yt-dlp")
	if fileExists(localBin) {
		return localBin
	}

	return current
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return value
}
