package rtmp

import (
	"github.com/yuhaohwang/bililive-go/src/pkg/events"
)

// RtmpStart 表示监听器开始工作的事件类型。
const RtmpStart events.EventType = "RtmpStart"

// RtmpStop 表示监听器停止工作的事件类型。
const RtmpStop events.EventType = "RtmpStop"

// ConfigChanged 表示配置发生修改。
const ConfigChanged events.EventType = "ConfigChanged"
