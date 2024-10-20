package pluginmanager

import (
	"path/filepath"

	"github.com/pkg/errors"
)

type Sandbox interface {
	Enable() error
	Disable() error
	VerifyPluginPath(path string) error
}

type ISandbox struct {
	originalDir   string
	originalUmask int
	chrootDir     string
}

func NewSandbox(chrootDir string) *ISandbox {
	if chrootDir == "" {
		chrootDir = "./sandbox"
	}
	return &ISandbox{
		chrootDir: chrootDir,
	}
}

// VerifyPluginPath 优化:
// - 使用 filepath.Clean 来规范化路径
// - 使用 errors.Wrap 提供更详细的错误信息
func (s *ISandbox) VerifyPluginPath(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return errors.Wrap(err, "获取插件绝对路径失败")
	}

	cleanPath := filepath.Clean(absPath)
	cleanChrootDir := filepath.Clean(s.chrootDir)

	if !filepath.HasPrefix(cleanPath, cleanChrootDir) {
		return errors.Wrapf(ErrPluginSandboxViolation, "插件路径 %s 不在沙箱目录 %s 内", cleanPath, cleanChrootDir)
	}

	return nil
}
