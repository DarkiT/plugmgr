//go:build !windows
// +build !windows

package plugmgr

import (
	"os"
	"syscall"
)

// Enable 激活沙箱
func (s *ISandbox) Enable() error {
	var err error
	s.originalDir, err = os.Getwd()
	if err != nil {
		return wrap(err, "获取当前工作目录失败")
	}

	if err := os.MkdirAll(s.chrootDir, 0o755); err != nil {
		return wrapf(err, "创建沙箱目录失败: %s", s.chrootDir)
	}

	if err := syscall.Chroot(s.chrootDir); err != nil {
		return wrapf(err, "切换根目录失败: %s", s.chrootDir)
	}

	if err := os.Chdir("/"); err != nil {
		return wrap(err, "切换到新的根目录失败")
	}

	s.originalUmask = syscall.Umask(0)

	return nil
}

// Disable 关闭沙箱
func (s *ISandbox) Disable() error {
	syscall.Umask(s.originalUmask)

	if err := syscall.Chroot("."); err != nil {
		return wrap(err, "恢复原始根目录失败")
	}

	if err := os.Chdir(s.originalDir); err != nil {
		return wrapf(err, "切换回原始工作目录失败: %s", s.originalDir)
	}

	return nil
}
