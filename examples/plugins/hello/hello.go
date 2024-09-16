package main

import (
	"fmt"

	pm "github.com/darkit/plugins"
)

var Plugin HelloPlugin

type HelloPluginConfig struct {
	Greeting string
}

type HelloPlugin struct {
	config HelloPluginConfig
}

func init() {
	Plugin.config = HelloPluginConfig{Greeting: "Hello"}
}

func (p *HelloPlugin) Metadata() pm.PluginMetadata {
	return pm.PluginMetadata{
		Name:         "HelloPlugin",
		Version:      "1.1.0",
		Dependencies: map[string]string{},
		Config:       p.config,
	}
}

func (p *HelloPlugin) Init() error {
	fmt.Println("HelloPlugin initialized with greeting:", p.config.Greeting)
	return nil
}

func (p *HelloPlugin) Execute(data interface{}) (interface{}, error) {
	name, ok := data.(string)
	if !ok {
		return nil, fmt.Errorf("invalid input: expected a string")
	}
	message := fmt.Sprintf("%s %s from HelloPlugin!", p.config.Greeting, name)
	fmt.Println(message)
	return message, nil
}

func (p *HelloPlugin) Shutdown() error {
	fmt.Println("HelloPlugin shut down")
	return nil
}

func (p *HelloPlugin) PreLoad(config interface{}) error {
	if config != nil {
		if cfg, ok := config.(HelloPluginConfig); ok {
			p.config = cfg
		} else {
			return fmt.Errorf("invalid config type")
		}
	}
	fmt.Println("HelloPlugin pre-load with config:", p.config)
	return nil
}

func (p *HelloPlugin) PostLoad() error {
	fmt.Println("HelloPlugin post-load")
	return nil
}

func (p *HelloPlugin) PreUnload() error {
	fmt.Println("HelloPlugin pre-unload")
	return nil
}

func (p *HelloPlugin) ManageConfig(config interface{}) (interface{}, error) {
	if config == nil {
		return p.config, nil
	}

	newConfig, ok := config.(HelloPluginConfig)
	if !ok {
		return nil, fmt.Errorf("invalid config type")
	}

	p.config = newConfig
	fmt.Println("HelloPlugin config updated:", p.config)
	return p.config, nil
}

func (p *HelloPlugin) ConfigType() interface{} {
	return p.config
}
