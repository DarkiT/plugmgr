package pluginmanager

import "errors"

func (s *ISandbox) Enable() error {
	return errors.New("your operating system is not supported")
}

func (s *ISandbox) Disable() error {
	return errors.New("your operating system is not supported")
}

func (s *ISandbox) VerifyPluginPath(_ string) error {
	return errors.New("your operating system is not supported")
}
