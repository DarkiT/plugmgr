package main

import (
	"fmt"

	pm "github.com/darkit/plugins"
)

type MyPlugin struct {
	Name    string
	Version string
	Config  map[string]interface{}
}

func (p *MyPlugin) Metadata() pm.PluginMetadata {
	return pm.PluginMetadata{
		Name:    "MyPlugin",
		Version: "1.0.0",
		Dependencies: map[string]string{
			"SomeOtherPlugin": ">=1.0.0",
		},
		GoVersion: "1.16",
	}
}

func (p *MyPlugin) PreLoad(config interface{}) error {
	fmt.Println("MyPlugin: PreLoad called")
	if cfg, ok := config.(map[string]interface{}); ok {
		p.Config = cfg
	}
	return nil
}

func (p *MyPlugin) Init() error {
	fmt.Println("MyPlugin: Init called")
	return nil
}

func (p *MyPlugin) PostLoad() error {
	fmt.Println("MyPlugin: PostLoad called")
	return nil
}

func (p *MyPlugin) Execute(data ...interface{}) error {
	fmt.Println("MyPlugin: Execute called with data:", data)
	return nil
}

func (p *MyPlugin) PreUnload() error {
	fmt.Println("MyPlugin: PreUnload called")
	return nil
}

func (p *MyPlugin) Shutdown() error {
	fmt.Println("MyPlugin: Shutdown called")
	return nil
}

func (p *MyPlugin) UpdateConfig(config interface{}) error {
	fmt.Println("MyPlugin: UpdateConfig called")
	if cfg, ok := config.(map[string]interface{}); ok {
		p.Config = cfg
		return nil
	}
	return fmt.Errorf("invalid config type")
}

var Plugin MyPlugin

// 导出插件
var ExportedPlugin MyPluginExport

type MyPluginExport struct{}

func (pe *MyPluginExport) NewPlugin() interface{} {
	return &Plugin
}
