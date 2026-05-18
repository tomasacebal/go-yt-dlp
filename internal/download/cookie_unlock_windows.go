//go:build windows

package download

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	rmErrorSuccess  = 0
	rmErrorMoreData = 234

	cchRmSessionKey = 32
	rmForceShutdown = 1
)

var (
	rstrtmgrDLL = windows.NewLazySystemDLL("rstrtmgr.dll")
	rmStart     = rstrtmgrDLL.NewProc("RmStartSession")
	rmRegister  = rstrtmgrDLL.NewProc("RmRegisterResources")
	rmGetList   = rstrtmgrDLL.NewProc("RmGetList")
	rmShutdown  = rstrtmgrDLL.NewProc("RmShutdown")
	rmEnd       = rstrtmgrDLL.NewProc("RmEndSession")
)

func unlockChromiumCookiesIfNeeded(cfg Config) error {
	if !cfg.AutoUnlockBrowserCookies || !isChromiumBrowserCookieSource(cfg.CookiesBrowser) {
		return nil
	}

	dbPath := resolveChromiumCookieDBPath(cfg.CookiesBrowser)
	if dbPath == "" {
		return nil
	}
	if _, err := os.Stat(dbPath); err != nil {
		return nil
	}

	if err := unlockFileWithRestartManager(dbPath); err != nil {
		return fmt.Errorf("no se pudo desbloquear cookie db %s: %w", dbPath, err)
	}
	return nil
}

func resolveChromiumCookieDBPath(cookiesBrowser string) string {
	profile := "Default"
	parts := strings.SplitN(strings.TrimSpace(cookiesBrowser), ":", 2)
	browser := strings.ToLower(parts[0])
	if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
		profile = strings.TrimSpace(parts[1])
	}

	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return ""
	}

	var userDataRoot string
	switch browser {
	case "chrome":
		userDataRoot = filepath.Join(localAppData, "Google", "Chrome", "User Data")
	case "chromium":
		userDataRoot = filepath.Join(localAppData, "Chromium", "User Data")
	case "edge":
		userDataRoot = filepath.Join(localAppData, "Microsoft", "Edge", "User Data")
	case "brave":
		userDataRoot = filepath.Join(localAppData, "BraveSoftware", "Brave-Browser", "User Data")
	case "opera":
		userDataRoot = filepath.Join(localAppData, "Opera Software", "Opera Stable")
	case "vivaldi":
		userDataRoot = filepath.Join(localAppData, "Vivaldi", "User Data")
	default:
		return ""
	}

	candidates := []string{
		filepath.Join(userDataRoot, profile, "Network", "Cookies"),
		filepath.Join(userDataRoot, profile, "Cookies"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return candidates[0]
}

func unlockFileWithRestartManager(filePath string) error {
	if err := rstrtmgrDLL.Load(); err != nil {
		return err
	}

	var sessionHandle uint32
	sessionKey := make([]uint16, cchRmSessionKey+1)
	result, _, _ := rmStart.Call(
		uintptr(unsafe.Pointer(&sessionHandle)),
		0,
		uintptr(unsafe.Pointer(&sessionKey[0])),
	)
	if result != rmErrorSuccess {
		return fmt.Errorf("RmStartSession fallo: %d", result)
	}
	defer rmEnd.Call(uintptr(sessionHandle))

	pathPtr, err := windows.UTF16PtrFromString(filePath)
	if err != nil {
		return err
	}
	files := []*uint16{pathPtr}
	result, _, _ = rmRegister.Call(
		uintptr(sessionHandle),
		uintptr(len(files)),
		uintptr(unsafe.Pointer(&files[0])),
		0,
		0,
		0,
		0,
	)
	if result != rmErrorSuccess {
		return fmt.Errorf("RmRegisterResources fallo: %d", result)
	}

	var procInfoNeeded uint32
	var procInfo uint32
	var rebootReasons uint32
	result, _, _ = rmGetList.Call(
		uintptr(sessionHandle),
		uintptr(unsafe.Pointer(&procInfoNeeded)),
		uintptr(unsafe.Pointer(&procInfo)),
		0,
		uintptr(unsafe.Pointer(&rebootReasons)),
	)
	if result != rmErrorSuccess && result != rmErrorMoreData {
		return fmt.Errorf("RmGetList fallo: %d", result)
	}
	if procInfoNeeded == 0 {
		return nil
	}

	result, _, _ = rmShutdown.Call(uintptr(sessionHandle), rmForceShutdown, 0)
	if result != rmErrorSuccess {
		return fmt.Errorf("RmShutdown fallo: %d", result)
	}
	return nil
}
