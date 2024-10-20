package pluginmanager

import (
	"github.com/pkg/errors"
)

// Enable 优化:
// - 使用 errors.New 提供更详细的错误信息
func (s *ISandbox) Enable() error {
	return errors.New("沙箱功能在Windows平台上不受支持")
}

// Disable 优化:
// - 使用 errors.New 提供更详细的错误信息
func (s *ISandbox) Disable() error {
	return errors.New("沙箱功能在Windows平台上不受支持")
}
