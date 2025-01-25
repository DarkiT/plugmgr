package main

import (
	"fmt"

	pm "github.com/darkit/plugmgr"
)

var Plugin *HelloPlugin

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
	var newConfig HelloPluginConfig
	if err := pm.Deserializer(config, &newConfig); err != nil {
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

func (p *HelloPlugin) ConfigUpdated(config []byte) ([]byte, error) {
	var newConfig HelloPluginConfig
	if err := pm.Deserializer(config, &newConfig); err != nil {
		return nil, err
	}
	p.config = newConfig
	return pm.Serializer(p.config)
}
