package plugmgr

import (
	"path/filepath"
	"strings"
)

type Sandbox interface {
	Enable() error
	Disable() error
	VerifyPluginPath(path string) error
}

type ISandbox struct {
	chrootDir string
}

func NewSandbox(chrootDir string) *ISandbox {
	if chrootDir == "" {
		chrootDir = "./sandbox"
	}
	return &ISandbox{
		chrootDir: chrootDir,
	}
}

// VerifyPluginPath 验证插件路径是否在沙箱目录内
//
//	参数:
//	- path: 插件路径
//	返回:
//	- error: 验证过程中的错误
//	功能:
//	- 使用 filepath.Abs 获取插件的绝对路径
//	- 使用 filepath.Clean 规范化路径
//	- 检查插件路径是否在沙箱目录内
func (s *ISandbox) VerifyPluginPath(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return wrap(err, "获取插件绝对路径失败")
	}

	cleanPath := filepath.Clean(absPath)
	cleanChrootDir := filepath.Clean(s.chrootDir)

	if !strings.HasPrefix(filepath.Clean(cleanPath), filepath.Clean(cleanChrootDir)+string(filepath.Separator)) {
		return wrapf(ErrPluginSandboxViolation, "插件路径 %s 不在沙箱目录 %s 内", cleanPath, cleanChrootDir)
	}

	return nil
}
