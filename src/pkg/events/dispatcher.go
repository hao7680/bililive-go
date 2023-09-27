package events

import (
	"container/list"
	"context"
	"sync"

	"github.com/yuhaohwang/bililive-go/src/instance"
	"github.com/yuhaohwang/bililive-go/src/interfaces"
)

// NewDispatcher 创建一个新的事件分发器并返回。
func NewDispatcher(ctx context.Context) Dispatcher {
	ed := &dispatcher{
		saver: make(map[EventType]*list.List),
	}
	inst := instance.GetInstance(ctx)
	if inst != nil {
		inst.EventDispatcher = ed
	}
	return ed
}

// Dispatcher 定义事件分发器的接口。
type Dispatcher interface {
	interfaces.Module
	AddEventListener(eventType EventType, listener *EventListener)
	RemoveEventListener(eventType EventType, listener *EventListener)
	RemoveAllEventListener(eventType EventType)
	DispatchEvent(event *Event)
}

// dispatcher 表示事件分发器的结构。
type dispatcher struct {
	sync.RWMutex
	saver map[EventType]*list.List // map<EventType, List<*EventListener>>
}

// Start 实现了接口中的 Start 方法。
func (e *dispatcher) Start(ctx context.Context) error {
	return nil
}

// Close 实现了接口中的 Close 方法。
func (e *dispatcher) Close(ctx context.Context) {

}

// AddEventListener 添加事件监听器。
func (e *dispatcher) AddEventListener(eventType EventType, listener *EventListener) {
	e.Lock()
	defer e.Unlock()
	listeners, ok := e.saver[eventType]
	if !ok || listener == nil {
		listeners = list.New()
		e.saver[eventType] = listeners
	}
	listeners.PushBack(listener)
}

// RemoveEventListener 移除事件监听器。
func (e *dispatcher) RemoveEventListener(eventType EventType, listener *EventListener) {
	e.Lock()
	defer e.Unlock()
	listeners, ok := e.saver[eventType]
	if !ok || listeners == nil {
		return
	}
	for e := listeners.Front(); e != nil; e = e.Next() {
		if e.Value == listener {
			listeners.Remove(e)
		}
	}
	if listeners.Len() == 0 {
		delete(e.saver, eventType)
	}
}

// RemoveAllEventListener 移除指定事件类型的所有监听器。
func (e *dispatcher) RemoveAllEventListener(eventType EventType) {
	e.Lock()
	defer e.Unlock()
	e.saver = make(map[EventType]*list.List)
}

// DispatchEvent 分发事件。
func (e *dispatcher) DispatchEvent(event *Event) {
	if event == nil {
		return
	}
	e.RLock()
	listeners, ok := e.saver[event.Type]
	if !ok || listeners == nil {
		e.RUnlock()
		return
	}
	hs := make([]*EventListener, 0)
	for e := listeners.Front(); e != nil; e = e.Next() {
		hs = append(hs, e.Value.(*EventListener))
	}
	e.RUnlock()
	go func() {
		for _, h := range hs {
			h.Handler(event)
		}
	}()
}
