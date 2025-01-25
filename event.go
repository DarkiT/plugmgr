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

type eventBus struct {
	mu       sync.RWMutex
	closed   atomic.Bool
	timeout  time.Duration
	handlers map[string][]EventHandler
}

// newEventBus 创建新的事件总线
func newEventBus() *eventBus {
	return &eventBus{
		handlers: make(map[string][]EventHandler),
		timeout:  5 * time.Second, // 默认超时时间
	}
}

// SetTimeout 设置事件处理超时时间
func (eb *eventBus) SetTimeout(timeout time.Duration) {
	eb.timeout = timeout
}

// Subscribe 订阅事件
func (eb *eventBus) Subscribe(eventName string, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.handlers[eventName] = append(eb.handlers[eventName], handler)
}

// Unsubscribe 取消订阅事件
func (eb *eventBus) Unsubscribe(eventName string, handler EventHandler) {
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
func (eb *eventBus) Close() error {
	if eb.closed.Swap(true) {
		return newError("事件总线已经关闭")
	}
	return nil
}

// PublishAsync 异步发布事件，带超时控制
func (eb *eventBus) PublishAsync(event Event) error {
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
func (eb *eventBus) Publish(event Event) error {
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
func (eb *eventBus) executeHandlerWithTimeout(handler EventHandler, event Event) {
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
func (eb *eventBus) getHandlers(eventName string) []EventHandler {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	handlers := make([]EventHandler, len(eb.handlers[eventName]))
	copy(handlers, eb.handlers[eventName])
	return handlers
}

// HasSubscribers 检查事件是否有订阅者
func (eb *eventBus) HasSubscribers(eventName string) bool {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	return len(eb.handlers[eventName]) > 0
}

// SubscribersCount 获取事件订阅者数量
func (eb *eventBus) SubscribersCount(eventName string) int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	return len(eb.handlers[eventName])
}
