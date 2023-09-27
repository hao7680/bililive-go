package listeners

import (
	"github.com/yuhaohwang/bililive-go/src/pkg/events"
)

// ListenStart 表示监听器开始工作的事件类型。
const ListenStart events.EventType = "ListenStart"

// ListenStop 表示监听器停止工作的事件类型。
const ListenStop events.EventType = "ListenStop"

// LiveStart 表示直播开始的事件类型。
const LiveStart events.EventType = "LiveStart"

// LiveEnd 表示直播结束的事件类型。
const LiveEnd events.EventType = "LiveEnd"

// RoomNameChanged 表示房间名称变更的事件类型。
const RoomNameChanged events.EventType = "RoomNameChanged"

// RoomInitializingFinished 表示房间初始化完成的事件类型。
const RoomInitializingFinished events.EventType = "RoomInitializingFinished"
