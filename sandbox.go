package pluginmanager

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
