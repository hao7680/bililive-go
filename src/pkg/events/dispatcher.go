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
	// 1. 创建一个新的事件分发器 ed。
	ed := &dispatcher{
		saver: make(map[EventType]*list.List),
	}

	// 2. 获取应用程序实例 inst，如果存在的话。
	inst := instance.GetInstance(ctx)

	// 3. 如果应用程序实例存在，将事件分发器 ed 赋值给应用程序实例的 EventDispatcher 字段。
	if inst != nil {
		inst.EventDispatcher = ed
	}

	// 4. 返回事件分发器 ed。
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

	// 1. 根据事件类型 eventType，从分发器的 saver 字典中获取与之相关的监听器列表 listeners 和一个标志 ok。
	listeners, ok := e.saver[eventType]

	// 2. 如果 ok 为 false 或者传入的监听器 listener 为 nil，说明该事件类型没有相关的监听器或者监听器为 nil。
	if !ok || listener == nil {
		// 3. 创建一个新的监听器列表 listeners。
		listeners = list.New()

		// 4. 将新创建的监听器列表与事件类型 eventType 关联，并保存到分发器的 saver 字典中。
		e.saver[eventType] = listeners
	}

	// 5. 将监听器 listener 添加到监听器列表 listeners 的尾部。
	listeners.PushBack(listener)
}

// RemoveEventListener 移除事件监听器。
func (e *dispatcher) RemoveEventListener(eventType EventType, listener *EventListener) {
	e.Lock()
	defer e.Unlock()

	// 1. 根据事件类型 eventType，从分发器的 saver 字典中获取与之相关的监听器列表 listeners 和一个标志 ok。
	listeners, ok := e.saver[eventType]

	// 2. 如果 ok 为 false 或者监听器列表 listeners 为空，说明没有与该事件类型相关的监听器。
	if !ok || listeners == nil {
		return
	}

	// 3. 遍历监听器列表 listeners，查找并移除与传入的监听器 listener 相同的监听器。
	for e := listeners.Front(); e != nil; e = e.Next() {
		if e.Value == listener {
			listeners.Remove(e)
		}
	}

	// 4. 如果监听器列表 listeners 变为空，从分发器的 saver 字典中移除该事件类型。
	if listeners.Len() == 0 {
		delete(e.saver, eventType)
	}
}

// RemoveAllEventListener 移除指定事件类型的所有监听器。
func (e *dispatcher) RemoveAllEventListener(eventType EventType) {
	e.Lock()
	defer e.Unlock()

	// 1. 创建一个新的空的监听器列表，用于移除指定事件类型的所有监听器。
	e.saver = make(map[EventType]*list.List)
}

// DispatchEvent 分发事件。
func (e *dispatcher) DispatchEvent(event *Event) {
	// 1. 检查传入的事件是否为 nil。
	if event == nil {
		return
	}

	// 2. 获取读锁。
	e.RLock()

	// 3. 从事件分发器的保存的事件监听器列表中，根据事件类型 event.Type 获取与之相关的监听器列表 listeners。
	listeners, ok := e.saver[event.Type]

	// 4. 如果找不到对应的监听器列表（ok 为 false）或者监听器列表为空，就释放读锁并返回。没有监听器需要处理此事件。
	if !ok || listeners == nil {
		e.RUnlock()
		return
	}

	// 5. 创建一个切片 hs，用于存储事件监听器。初始时为空切片。
	hs := make([]*EventListener, 0)

	// 6. 遍历监听器列表 listeners 中的每个监听器。
	for e := listeners.Front(); e != nil; e = e.Next() {
		// 7. 将监听器 e.Value.(*EventListener) 添加到切片 hs 中。这是为了将所有需要处理事件的监听器保存在 hs 中。
		hs = append(hs, e.Value.(*EventListener))
	}

	// 8. 释放读锁，允许其他 goroutine 对事件分发器进行读取或修改。
	e.RUnlock()

	// 9. 启动一个新的 goroutine。在这个 goroutine 中，会遍历切片 hs 中的监听器，并调用它们的事件处理函数来处理事件。
	go func() {
		for _, h := range hs {
			h.Handler(event)
		}
	}()
}
