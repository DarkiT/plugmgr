package pluginmanager

import (
	"github.com/pkg/errors"
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

// PluginError 优化:
// - 使用 errors.Wrap 来包装错误,提供更多上下文信息
type PluginError struct {
	Op     string
	Err    error
	Plugin string
}

func (e *PluginError) Error() string {
	if e.Plugin != "" {
		return errors.Wrapf(e.Err, "插件错误: %s: %s", e.Plugin, e.Op).Error()
	}
	return errors.Wrapf(e.Err, "插件错误: %s", e.Op).Error()
}

func (e *PluginError) Unwrap() error {
	return e.Err
}

func NewPluginError(op string, plugin string, err error) *PluginError {
	return &PluginError{
		Op:     op,
		Plugin: plugin,
		Err:    err,
	}
}
