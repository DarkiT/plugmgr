package plugmgr

// Enable 激活沙箱
func (s *ISandbox) Enable() error {
	return newError("沙箱功能在Windows平台上不受支持")
}

// Disable 关闭沙箱
func (s *ISandbox) Disable() error {
	return newError("沙箱功能在Windows平台上不受支持")
}
