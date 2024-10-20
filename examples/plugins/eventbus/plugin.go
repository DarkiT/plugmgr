package eventbus

import (
	"fmt"
	plugins "github.com/darkit/plugins"
	"github.com/darkit/plugins/eventbus"
)

type EventBusPlugin struct {
	Name    string
	Version string
	Config  interface{}
	bus     *eventbus.Pipe[string]
}

func (p *EventBusPlugin) Metadata() plugins.PluginMetadata {
	return plugins.PluginMetadata{
		Name:         p.Name,
		Version:      p.Version,
		Dependencies: map[string]string{},
	}
}

func (p *EventBusPlugin) PreLoad(config []byte) error {
	return plugins.Deserializer(config, &p.Config)
}

func (p *EventBusPlugin) Init() error {
	p.bus = eventbus.NewPipe[string]()
	return nil
}

func (p *EventBusPlugin) PostLoad() error {
	fmt.Println("EventBusPlugin: PostLoad called")
	return nil
}

func (p *EventBusPlugin) Execute(data ...interface{}) (interface{}, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("不足的参数")
	}

	action, ok := data[0].(string)
	if !ok {
		return nil, fmt.Errorf("第一个参数必须是字符串")
	}

	message, ok := data[1].(string)
	if !ok {
		return nil, fmt.Errorf("第二个参数必须是字符串")
	}

	switch action {
	case "publish":
		return nil, p.bus.Publish(message)
	case "subscribe":
		handler := func(payload string) {
			fmt.Printf("Received message: %s\n", payload)
		}
		return nil, p.bus.Subscribe(handler)
	default:
		return nil, fmt.Errorf("未知的操作: %s", action)
	}
}

func (p *EventBusPlugin) PreUnload() error {
	fmt.Println("EventBusPlugin: PreUnload called")
	return nil
}

func (p *EventBusPlugin) Shutdown() error {
	p.bus.Close()
	return nil
}

func (p *EventBusPlugin) ManageConfig(config []byte) ([]byte, error) {
	var newConfig interface{}
	if err := plugins.Deserializer(config, &newConfig); err != nil {
		return nil, err
	}
	p.Config = newConfig
	return plugins.Serializer(p.Config)
}

var Plugin EventBusPlugin = EventBusPlugin{
	Name:    "EventBusPlugin",
	Version: "1.0.0",
}
