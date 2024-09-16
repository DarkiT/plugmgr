package main

import (
	"fmt"

	pm "github.com/darkit/plugins"
)

var Plugin MyPlugin

type MyPluginConfig struct {
	Setting1 string
	Setting2 int
}

type MyPlugin struct {
	Name    string
	Version string
	Config  MyPluginConfig
}

func init() {
	Plugin = MyPlugin{
		Name:    "MyPlugin",
		Version: "1.0.1",
		Config: MyPluginConfig{
			Setting1: "default",
			Setting2: 0,
		},
	}
}

func (p *MyPlugin) Metadata() pm.PluginMetadata {
	return pm.PluginMetadata{
		Name:    "MyPlugin",
		Version: "1.0.1",
		Dependencies: map[string]string{
			"SomeOtherPlugin": ">=1.0.0",
		},
		GoVersion: "1.16",
		Config:    p.Config,
	}
}

func (p *MyPlugin) PreLoad(config []byte) error {
	fmt.Println("MyPlugin: PreLoad called")
	newConfig, err := pm.Deserializer[MyPluginConfig](config)
	if err != nil {
		return fmt.Errorf("invalid config type")
	}
	p.Config = newConfig

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

func (p *MyPlugin) Execute(data interface{}) (interface{}, error) {
	fmt.Println("MyPlugin: Execute called with data:", data)
	// 这里可以根据实际需求处理输入数据并返回结果
	return fmt.Sprintf("Executed with Setting1: %s, Setting2: %d", p.Config.Setting1, p.Config.Setting2), nil
}

func (p *MyPlugin) PreUnload() error {
	fmt.Println("MyPlugin: PreUnload called")
	return nil
}

func (p *MyPlugin) Shutdown() error {
	fmt.Println("MyPlugin: Shutdown called")
	return nil
}

func (p *MyPlugin) ManageConfig(config []byte) ([]byte, error) {
	c, err := pm.Serializer(p.Config)
	if err != nil {
		return nil, err
	}

	if config == nil {
		return c, nil
	}

	newConfig, err := pm.Deserializer[MyPluginConfig](config)
	if err != nil {
		return nil, fmt.Errorf("invalid config type")
	}

	p.Config = newConfig
	fmt.Println("HelloPlugin config updated:", p.Config)

	return c, nil
}
