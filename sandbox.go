// 版权所有 (C) 2024 Matt Dunleavy。保留所有权利。
// 本源代码的使用受 LICENSE 文件中的 MIT 许可证约束。

package pluginmanager

import (
	"os"
	"path/filepath"
	"syscall"
)

type Sandbox interface {
	Enable() error
	Disable() error
	VerifyPluginPath(path string) error
}

type LinuxSandbox struct {
	originalDir   string
	originalUmask int
	chrootDir     string
}

func NewLinuxSandbox(chrootDir string) *LinuxSandbox {
	if chrootDir == "" {
		chrootDir = "./sandbox"
	}
	return &LinuxSandbox{
		chrootDir: chrootDir,
	}
}

func (s *LinuxSandbox) Enable() error {
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

func (s *LinuxSandbox) Disable() error {
	syscall.Umask(s.originalUmask)

	if err := syscall.Chroot("."); err != nil {
		return err
	}

	if err := os.Chdir(s.originalDir); err != nil {
		return err
	}

	return nil
}

func (s *LinuxSandbox) VerifyPluginPath(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	if !filepath.HasPrefix(absPath, s.chrootDir) {
		return ErrPluginSandboxViolation
	}

	return nil
}
