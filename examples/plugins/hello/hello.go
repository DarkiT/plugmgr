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

func (p *HelloPlugin) PreLoad(config []byte) error {
	fmt.Println("HelloPlugin: PreLoad called")
	newConfig, err := pm.Deserializer[HelloPluginConfig](config)
	if err != nil {
		return fmt.Errorf("invalid config type")
	}
	p.config = newConfig

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

func (p *HelloPlugin) ManageConfig(config []byte) ([]byte, error) {
	c, err := pm.Serializer(p.config)
	if err != nil {
		return nil, err
	}

	if config == nil {
		return c, nil
	}

	newConfig, err := pm.Deserializer[HelloPluginConfig](config)
	if err != nil {
		return nil, fmt.Errorf("invalid config type")
	}

	p.config = newConfig
	fmt.Println("HelloPlugin config updated:", p.config)

	return c, nil
}
