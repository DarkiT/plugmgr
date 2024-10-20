//go:build !windows
// +build !windows

package pluginmanager

import (
	"os"
	"syscall"

	"github.com/pkg/errors"
)

// Enable 优化:
// - 使用 errors.Wrap 提供更详细的错误信息
func (s *ISandbox) Enable() error {
	var err error
	s.originalDir, err = os.Getwd()
	if err != nil {
		return errors.Wrap(err, "获取当前工作目录失败")
	}

	if err := os.MkdirAll(s.chrootDir, 0o755); err != nil {
		return errors.Wrapf(err, "创建沙箱目录失败: %s", s.chrootDir)
	}

	if err := syscall.Chroot(s.chrootDir); err != nil {
		return errors.Wrapf(err, "切换根目录失败: %s", s.chrootDir)
	}

	if err := os.Chdir("/"); err != nil {
		return errors.Wrap(err, "切换到新的根目录失败")
	}

	s.originalUmask = syscall.Umask(0)

	return nil
}

// Disable 优化:
// - 使用 errors.Wrap 提供更详细的错误信息
func (s *ISandbox) Disable() error {
	syscall.Umask(s.originalUmask)

	if err := syscall.Chroot("."); err != nil {
		return errors.Wrap(err, "恢复原始根目录失败")
	}

	if err := os.Chdir(s.originalDir); err != nil {
		return errors.Wrapf(err, "切换回原始工作目录失败: %s", s.originalDir)
	}

	return nil
}
