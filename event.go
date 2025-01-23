package plugmgr

import (
	"sync"
)

type Event interface {
	Name() string
	Data() interface{}
}

// BaseEvent 提供基础事件实现
type BaseEvent struct {
	name string
	data interface{}
}

func (e *BaseEvent) Name() string {
	return e.name
}

func (e *BaseEvent) Data() interface{} {
	return e.data
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

// EventHandler 事件处理函数
type EventHandler func(Event)

// workerPool 内部工作池实现
type workerPool struct {
	tasks chan func()
	wg    sync.WaitGroup
}

func newWorkerPool(size int) *workerPool {
	pool := &workerPool{
		tasks: make(chan func(), 100), // 设置合理的缓冲区大小
	}

	pool.wg.Add(size)
	for i := 0; i < size; i++ {
		go func() {
			defer pool.wg.Done()
			for task := range pool.tasks {
				task()
			}
		}()
	}
	return pool
}

// EventBus 事件总线
type EventBus struct {
	handlers   map[string][]EventHandler
	mu         sync.RWMutex
	workerPool *workerPool
}

func NewEventBus() *EventBus {
	return &EventBus{
		handlers:   make(map[string][]EventHandler),
		workerPool: newWorkerPool(10), // 默认10个工作协程
	}
}

// Subscribe 订阅事件
func (eb *EventBus) Subscribe(eventName string, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.handlers[eventName] = append(eb.handlers[eventName], handler)
}

// Publish 发布事件
func (eb *EventBus) Publish(event Event) {
	eb.mu.RLock()
	handlers := make([]EventHandler, len(eb.handlers[event.Name()]))
	copy(handlers, eb.handlers[event.Name()])
	eb.mu.RUnlock()

	if len(handlers) == 0 {
		return
	}

	for _, handler := range handlers {
		h := handler // 创建副本避免闭包问题
		select {
		case eb.workerPool.tasks <- func() { h(event) }:
			// 任务提交到工作池
		default:
			// 如果工作池队列满，直接运行
			go h(event)
		}
	}
}
