package pluginmanager

import (
	"sync"
)

type Event interface {
	Name() string
}

type PluginLoadedEvent struct {
	PluginName string
}

func (e PluginLoadedEvent) Name() string {
	return "PluginLoaded"
}

type PluginUnloadedEvent struct {
	PluginName string
}

func (e PluginUnloadedEvent) Name() string {
	return "PluginUnloaded"
}

type PluginHotReloadedEvent struct {
	PluginName string
}

func (e PluginHotReloadedEvent) Name() string {
	return "PluginHotReloaded"
}

type EventHandler func(Event)

// EventBus 优化:
// - 使用 sync.RWMutex 来提高并发性能
type EventBus struct {
	handlers map[string][]EventHandler
	mu       sync.RWMutex
}

func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[string][]EventHandler),
	}
}

// Subscribe 优化:
// - 使用写锁确保并发安全
func (eb *EventBus) Subscribe(eventName string, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.handlers[eventName] = append(eb.handlers[eventName], handler)
}

// Publish 优化:
// - 使用读锁提高并发性能
// - 使用 go 关键字异步执行事件处理器
func (eb *EventBus) Publish(event Event) {
	eb.mu.RLock()
	handlers := eb.handlers[event.Name()]
	eb.mu.RUnlock()

	for _, handler := range handlers {
		go handler(event)
	}
}
