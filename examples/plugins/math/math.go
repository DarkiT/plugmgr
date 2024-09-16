package main

import (
	"fmt"

	pm "github.com/darkit/plugins"
)

var Plugin MathPlugin

type MathPluginConfig struct {
	DefaultValue int
}

type MathPlugin struct {
	config MathPluginConfig
}

func init() {
	Plugin.config = MathPluginConfig{DefaultValue: 0}
}

func (p *MathPlugin) Metadata() pm.PluginMetadata {
	return pm.PluginMetadata{
		Name:         "MathPlugin",
		Version:      "1.1.0",
		Dependencies: map[string]string{},
		Config:       p.config,
	}
}

func (p *MathPlugin) Init() error {
	fmt.Println("MathPlugin initialized with default value:", p.config.DefaultValue)
	return nil
}

func (p *MathPlugin) Execute(data interface{}) (interface{}, error) {
	params, ok := data.([]int)
	if !ok || len(params) != 2 {
		return nil, fmt.Errorf("invalid input: expected two integers")
	}
	result := p.Add(params[0], params[1])
	fmt.Printf("MathPlugin: %d + %d = %d\n", params[0], params[1], result)
	return result, nil
}

func (p *MathPlugin) Shutdown() error {
	fmt.Println("MathPlugin shut down")
	return nil
}

func (p *MathPlugin) PreLoad(config []byte) error {
	fmt.Println("MathPlugin: PreLoad called")
	newConfig, err := pm.Deserializer[MathPluginConfig](config)
	if err != nil {
		return fmt.Errorf("invalid config type")
	}
	p.config = newConfig

	return nil
}

func (p *MathPlugin) PostLoad() error {
	fmt.Println("MathPlugin post-load")
	return nil
}

func (p *MathPlugin) PreUnload() error {
	fmt.Println("MathPlugin pre-unload")
	return nil
}

func (p *MathPlugin) ManageConfig(config []byte) ([]byte, error) {
	c, err := pm.Serializer(p.config)
	if err != nil {
		return nil, err
	}

	if config == nil {
		return c, nil
	}

	newConfig, err := pm.Deserializer[MathPluginConfig](config)
	if err != nil {
		return nil, fmt.Errorf("invalid config type")
	}

	p.config = newConfig
	fmt.Println("HelloPlugin config updated:", p.config)

	return c, nil
}

func (p *MathPlugin) Add(a, b int) int {
	return a + b + p.config.DefaultValue
}
