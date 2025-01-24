package plugmgr

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const (
	PluginLoaded         = "PluginLoaded"
	PluginInitialized    = "PluginInitialized"
	PluginExecuted       = "PluginExecuted"
	PluginConfigUpdated  = "PluginConfigUpdated"
	PluginPreUnload      = "PluginPreUnload"
	PluginUnloaded       = "PluginUnloaded"
	PluginExecutionError = "PluginExecutionError"
	PluginHotReloaded    = "PluginHotReloaded"
)

type Event struct {
	EventName string
	Data      EventData
}

type EventData struct {
	Name  string
	Data  interface{}
	Error error
}

type EventHandler func(Event)

type EventBus struct {
	mu       sync.RWMutex
	closed   atomic.Bool
	timeout  time.Duration
	handlers map[string][]EventHandler
}

// EventBusOption 配置选项
type EventBusOption func(*EventBus)

// WithTimeout 设置事件处理超时时间
func WithTimeout(timeout time.Duration) EventBusOption {
	return func(eb *EventBus) {
		eb.timeout = timeout
	}
}

// NewEventBus 创建新的事件总线
func NewEventBus(opts ...EventBusOption) *EventBus {
	eb := &EventBus{
		handlers: make(map[string][]EventHandler),
		timeout:  5 * time.Second, // 默认超时时间
	}

	for _, opt := range opts {
		opt(eb)
	}
	return eb
}

// Subscribe 订阅事件
func (eb *EventBus) Subscribe(eventName string, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.handlers[eventName] = append(eb.handlers[eventName], handler)
}

// Unsubscribe 取消订阅事件
func (eb *EventBus) Unsubscribe(eventName string, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	handlers := eb.handlers[eventName]
	for i, h := range handlers {
		if fmt.Sprintf("%p", h) == fmt.Sprintf("%p", handler) {
			eb.handlers[eventName] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}
}

// Close 关闭事件总线
func (eb *EventBus) Close() error {
	if eb.closed.Swap(true) {
		return newError("事件总线已经关闭")
	}
	return nil
}

// PublishAsync 异步发布事件，带超时控制
func (eb *EventBus) PublishAsync(event Event) error {
	if eb.closed.Load() {
		return newError("事件总线已经关闭")
	}

	handlers := eb.getHandlers(event.EventName)
	if len(handlers) == 0 {
		return nil
	}

	for _, handler := range handlers {
		go eb.executeHandlerWithTimeout(handler, event)
	}
	return nil
}

// Publish 同步发布事件
func (eb *EventBus) Publish(event Event) error {
	if eb.closed.Load() {
		return newError("事件总线已经关闭")
	}

	handlers := eb.getHandlers(event.EventName)
	if len(handlers) == 0 {
		return nil
	}

	for _, handler := range handlers {
		handler(event)
	}
	return nil
}

// executeHandlerWithTimeout 带超时控制的事件处理
func (eb *EventBus) executeHandlerWithTimeout(handler EventHandler, event Event) {
	ctx, cancel := context.WithTimeout(context.Background(), eb.timeout)
	defer cancel()

	done := make(chan struct{}, 1)
	go func() {
		handler(event)
		done <- struct{}{}
	}()

	select {
	case <-done:
		// 处理完成
	case <-ctx.Done():
		// 处理超时
	}
}

// getHandlers 获取特定事件的处理器
func (eb *EventBus) getHandlers(eventName string) []EventHandler {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	handlers := make([]EventHandler, len(eb.handlers[eventName]))
	copy(handlers, eb.handlers[eventName])
	return handlers
}

// HasSubscribers 检查事件是否有订阅者
func (eb *EventBus) HasSubscribers(eventName string) bool {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	return len(eb.handlers[eventName]) > 0
}

// SubscribersCount 获取事件订阅者数量
func (eb *EventBus) SubscribersCount(eventName string) int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	return len(eb.handlers[eventName])
}
