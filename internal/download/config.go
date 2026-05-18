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
	defaultPluginDir      = "."
	defaultCleanupEvery   = time.Hour
	defaultFileTTL        = 24 * time.Hour
	defaultMaxBodyBytes   = 1 * 1024 * 1024
	defaultWSWriteTimeout = 5 * time.Second
)

// Config representa la configuracion operativa del servicio.
type Config struct {
	ListenAddr               string
	WorkerCount              int
	QueueCapacity            int
	DownloadDir              string
	YTDLPBin                 string
	JSRuntimes               string
	CookiesFile              string
	CookiesBrowser           string
	PluginDir                string
	EnableChromeUnlockPlugin bool
	FFmpegLocation           string
	CleanupEvery             time.Duration
	FileTTL                  time.Duration
	MaxBodyBytes             int
	WSWriteTimeout           time.Duration
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
		ListenAddr:               getEnv("LISTEN_ADDR", defaultListenAddr),
		WorkerCount:              getEnvInt("WORKER_COUNT", defaultWorkerCount),
		QueueCapacity:            getEnvInt("QUEUE_CAPACITY", defaultQueueCapacity),
		DownloadDir:              getEnv("DOWNLOAD_DIR", defaultDownloadDir),
		YTDLPBin:                 getEnv("YTDLP_BIN", defaultYTDLPBin),
		JSRuntimes:               getEnv("YTDLP_JS_RUNTIMES", defaultJSRuntimes),
		CookiesFile:              getEnv("YTDLP_COOKIES_FILE", ""),
		CookiesBrowser:           getEnv("YTDLP_COOKIES_FROM_BROWSER", ""),
		PluginDir:                getEnv("YTDLP_PLUGIN_DIR", defaultPluginDir),
		EnableChromeUnlockPlugin: getEnvBool("YTDLP_CHROME_COOKIE_UNLOCK_PLUGIN", runtime.GOOS == "windows"),
		FFmpegLocation:           getEnv("FFMPEG_LOCATION", defaultFFmpegLocation),
		CleanupEvery:             getEnvDuration("CLEANUP_EVERY", defaultCleanupEvery),
		FileTTL:                  getEnvDuration("FILE_TTL", defaultFileTTL),
		MaxBodyBytes:             getEnvInt("MAX_BODY_BYTES", defaultMaxBodyBytes),
		WSWriteTimeout:           getEnvDuration("WS_WRITE_TIMEOUT", defaultWSWriteTimeout),
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
	cfg.CookiesFile = strings.TrimSpace(cfg.CookiesFile)
	cfg.CookiesBrowser = strings.TrimSpace(cfg.CookiesBrowser)
	cfg.PluginDir = strings.TrimSpace(cfg.PluginDir)
	cfg.FFmpegLocation = strings.TrimSpace(cfg.FFmpegLocation)
	cfg.PluginDir = resolvePluginDir(cfg.PluginDir)
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

func getEnvBool(key string, fallback bool) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if raw == "" {
		return fallback
	}
	switch raw {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func resolvePluginDir(path string) string {
	if path == "" {
		path = defaultPluginDir
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return filepath.Clean(path)
	}
	return filepath.Join(cwd, path)
}
