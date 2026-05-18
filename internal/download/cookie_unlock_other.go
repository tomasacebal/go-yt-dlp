//go:build !windows

package download

func unlockChromiumCookiesIfNeeded(_ Config) error {
	return nil
}
