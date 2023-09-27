package events

// EventType 表示事件类型的字符串。
type EventType string

// EventHandler 定义事件处理函数的类型。
type EventHandler func(event *Event)

// Event 表示一个事件的结构。
type Event struct {
	Type   EventType   // 事件类型
	Object interface{} // 事件相关的对象
}

// NewEvent 创建一个新的事件。
func NewEvent(eventType EventType, object interface{}) *Event {
	return &Event{eventType, object}
}

// EventListener 表示事件监听器的结构。
type EventListener struct {
	Handler EventHandler // 事件处理函数
}

// NewEventListener 创建一个新的事件监听器。
func NewEventListener(handler EventHandler) *EventListener {
	return &EventListener{handler}
}
