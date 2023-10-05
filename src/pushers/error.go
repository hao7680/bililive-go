package pushers

import "errors"

var (
	// ErrPusherExist 表示已存在的记录器错误。
	ErrPusherExist = errors.New("pusher is exist")

	// ErrPusherNotExist 表示不存在的记录器错误。
	ErrPusherNotExist = errors.New("pusher is not exist")

	// ErrPusherNotSupportStatus 表示解析器不支持获取状态的错误。
	ErrPusherNotSupportStatus = errors.New("pusher not support get status")

	// ErrRtmpNotExist 表示RTMP不存在
	ErrRtmpNotExist = errors.New("rtmp is not exist")

	// ErrListenNotEnabled 表示监听未启用
	ErrListenNotEnabled = errors.New("listen is not enabled")

	// ErrPushNotEnabled 表示推送未启用
	ErrPushNotEnabled = errors.New("push is not enabled")
)
