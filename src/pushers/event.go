package pushers

import "github.com/yuhaohwang/bililive-go/src/pkg/events"

// PusherStart 是一个事件类型，表示录制器开始录制。
const PusherStart events.EventType = "PusherStart"

// PusherStop 是一个事件类型，表示录制器停止录制。
const PusherStop events.EventType = "PusherStop"

// PusherRestart 是一个事件类型，表示录制器重新启动录制。
const PusherRestart events.EventType = "PusherRestart"
