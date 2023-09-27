package recorders

import "github.com/yuhaohwang/bililive-go/src/pkg/events"

// RecorderStart 是一个事件类型，表示录制器开始录制。
const RecorderStart events.EventType = "RecorderStart"

// RecorderStop 是一个事件类型，表示录制器停止录制。
const RecorderStop events.EventType = "RecorderStop"

// RecorderRestart 是一个事件类型，表示录制器重新启动录制。
const RecorderRestart events.EventType = "RecorderRestart"
