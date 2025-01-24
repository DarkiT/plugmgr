package plugmgr

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// 定义错误类型常量
const (
	errTypeValidation = "validation"
	errTypeRuntime    = "runtime"
	errTypeSystem     = "system"
)

// 常用的插件错误
var (
	ErrPluginAlreadyLoaded    = NewPluginError("插件已加载", errTypeValidation)
	ErrInvalidPluginInterface = NewPluginError("无效的插件接口", errTypeValidation)
	ErrPluginNotFound         = NewPluginError("未找到插件", errTypeValidation)
	ErrIncompatibleVersion    = NewPluginError("插件版本不兼容", errTypeValidation)
	ErrMissingDependency      = NewPluginError("缺少插件依赖", errTypeValidation)
	ErrCircularDependency     = NewPluginError("检测到循环依赖", errTypeValidation)
	ErrPluginSandboxViolation = NewPluginError("插件违反沙箱规则", errTypeRuntime)
)

// newError 返回一个带有提供消息的错误
func newError(message string) error {
	if message == "" {
		return nil
	}
	return NewPluginError(message, errTypeSystem)
}

// newErrorf 返回一个带有格式化消息的错误
func newErrorf(format string, args ...any) error {
	if format == "" {
		return nil
	}
	return NewPluginError(fmt.Sprintf(format, args...), errTypeSystem)
}

// wrap 返回一个错误，将 err 注解上提供的消息
func wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return &withMessage{
		cause: err,
		msg:   message,
	}
}

// wrapf 返回一个错误，将 err 注解上提供的格式化消息
func wrapf(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	return &withMessage{
		cause: err,
		msg:   fmt.Sprintf(format, args...),
	}
}

// is 报告 err 链中的任何错误是否与 target 匹配
func is(err, target error) bool {
	return errors.Is(err, target)
}

// as 在 err 链中找到与 target 匹配的第一个错误
func as(err error, target interface{}) bool {
	return errors.As(err, target)
}

// unwrap 返回 err 的底层错误
func unwrap(err error) error {
	return errors.Unwrap(err)
}

// isPluginError 检查错误是否为 PluginError 类型
func isPluginError(err error) bool {
	var pe *PluginError
	return errors.As(err, &pe)
}

// getPluginError 将错误转换为 PluginError
func getPluginError(err error) (*PluginError, bool) {
	var pe *PluginError
	if errors.As(err, &pe) {
		return pe, true
	}
	return nil, false
}

// NewPluginError 创建新的插件错误
func NewPluginError(message string, errType string) *PluginError {
	var stack strings.Builder

	// 获取调用栈信息
	for i := 2; i < 7; i++ { // 限制堆栈深度为5
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		fn := runtime.FuncForPC(pc)
		stack.WriteString(fmt.Sprintf("\n\t%s:%d %s", file, line, fn.Name()))
	}

	return &PluginError{
		message: message,
		errType: errType,
		stack:   stack.String(),
	}
}

// PluginError 定义插件错误结构
type PluginError struct {
	message  string
	errType  string
	stack    string
	metadata map[string]interface{}
}

// Error 实现 error 接口
func (e *PluginError) Error() string {
	return e.message
}

// Type 返回错误类型
func (e *PluginError) Type() string {
	return e.errType
}

// Stack 返回错误堆栈
func (e *PluginError) Stack() string {
	return e.stack
}

// WithMetadata 添加元数据到错误
func (e *PluginError) WithMetadata(key string, value interface{}) *PluginError {
	if e.metadata == nil {
		e.metadata = make(map[string]interface{})
	}
	e.metadata[key] = value
	return e
}

// Metadata 获取错误的元数据
func (e *PluginError) Metadata() map[string]interface{} {
	return e.metadata
}

// withMessage 定义带消息的错误结构
type withMessage struct {
	cause error
	msg   string
}

func (w *withMessage) Error() string {
	return w.msg + ": " + w.cause.Error()
}

func (w *withMessage) Cause() error {
	return w.cause
}
