//go:build !windows
// +build !windows

package pluginmanager

import (
	"os"
	"path/filepath"
	"syscall"
)

func (s *ISandbox) Enable() error {
	var err error
	s.originalDir, err = os.Getwd()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(s.chrootDir, 0o755); err != nil {
		return err
	}

	if err := syscall.Chroot(s.chrootDir); err != nil {
		return err
	}

	if err := os.Chdir("/"); err != nil {
		return err
	}

	s.originalUmask = syscall.Umask(0)

	return nil
}

func (s *ISandbox) Disable() error {
	syscall.Umask(s.originalUmask)

	if err := syscall.Chroot("."); err != nil {
		return err
	}

	if err := os.Chdir(s.originalDir); err != nil {
		return err
	}

	return nil
}

func (s *ISandbox) VerifyPluginPath(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	if !filepath.HasPrefix(absPath, s.chrootDir) {
		return ErrPluginSandboxViolation
	}

	return nil
}
