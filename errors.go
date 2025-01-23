package plugmgr

import (
	"fmt"

	"github.com/pkg/errors"
)

var (
	ErrPluginAlreadyLoaded    = errors.New("插件已加载")
	ErrInvalidPluginInterface = errors.New("无效的插件接口")
	ErrPluginNotFound         = errors.New("未找到插件")
	ErrIncompatibleVersion    = errors.New("插件版本不兼容")
	ErrMissingDependency      = errors.New("缺少插件依赖")
	ErrCircularDependency     = errors.New("检测到循环依赖")
	ErrPluginSandboxViolation = errors.New("插件违反沙箱规则")
)

type PluginError struct {
	Op      string
	Code    int
	Plugin  string
	Err     error
	Details map[string]interface{}
}

func (e *PluginError) Error() string {
	if len(e.Details) > 0 {
		return fmt.Sprintf("插件错误[%d]: %s: %s: %v, 详细信息: %v", e.Code, e.Plugin, e.Op, e.Err, e.Details)
	}
	return fmt.Sprintf("插件错误[%d]: %s: %s: %v", e.Code, e.Plugin, e.Op, e.Err)
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
