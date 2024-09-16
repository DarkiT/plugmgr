// 版权所有 (C) 2024 Matt Dunleavy。保留所有权利。
// 本源代码的使用受 LICENSE 文件中的 MIT 许可证约束。

package pluginmanager

import (
	"plugin"
	"time"
)

type PluginMetadata struct {
	Name         string
	Version      string
	Dependencies map[string]string
	GoVersion    string
	Signature    []byte
	Config       interface{}
}

type Plugin interface {
	Metadata() PluginMetadata
	PreLoad(config interface{}) error
	Init() error
	PostLoad() error
	Execute(data interface{}) (interface{}, error)
	PreUnload() error
	Shutdown() error
	ManageConfig(config interface{}) (interface{}, error)
}

type PluginStats struct {
	ExecutionCount     int64
	LastExecutionTime  time.Duration
	TotalExecutionTime time.Duration
}

func LoadPlugin(path string) (Plugin, error) {
	p, err := plugin.Open(path)
	if err != nil {
		return nil, &PluginError{Op: "open", Err: err}
	}

	symPlugin, err := p.Lookup(PluginSymbol)
	if err != nil {
		return nil, &PluginError{Op: "lookup", Err: err}
	}

	plugin, ok := symPlugin.(Plugin)
	if !ok {
		return nil, &PluginError{Op: "assert", Err: ErrInvalidPluginInterface}
	}

	return plugin, nil
}

const PluginSymbol = "Plugin"
