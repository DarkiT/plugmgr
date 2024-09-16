// 版权所有 (C) 2024 Matt Dunleavy。保留所有权利。
// 本源代码的使用受 LICENSE 文件中的 MIT 许可证约束。

package pluginmanager

import (
	"errors"
	"fmt"
)

var (
	ErrPluginAlreadyLoaded    = errors.New("插件已加载")
	ErrInvalidPluginInterface = errors.New("无效的插件接口")
	ErrPluginNotFound         = errors.New("未找到插件")
	ErrIncompatibleVersion    = errors.New("插件版本不兼容")
	ErrMissingDependency      = errors.New("缺少插件依赖")
	ErrCircularDependency     = errors.New("检测到循环依赖")
	ErrPluginSandboxViolation = errors.New("插件尝试违反沙箱")
)

type PluginError struct {
	Op     string
	Err    error
	Plugin string
}

func (e *PluginError) Error() string {
	if e.Plugin != "" {
		return fmt.Sprintf("插件错误: %s: %s: %v", e.Plugin, e.Op, e.Err)
	}
	return fmt.Sprintf("插件错误: %s: %v", e.Op, e.Err)
}

func (e *PluginError) Unwrap() error {
	return e.Err
}
