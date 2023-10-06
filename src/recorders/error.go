package recorders

import "errors"

var (
	// ErrRecorderExist 表示已存在的记录器错误。
	ErrRecorderExist = errors.New("recorder is exist")

	// ErrRecorderNotExist 表示不存在的记录器错误。
	ErrRecorderNotExist = errors.New("recorder is not exist")

	// ErrParserNotSupportStatus 表示解析器不支持获取状态的错误。
	ErrParserNotSupportStatus = errors.New("parser not support get status")

	// ErrListenNotEnabled 表示监听未启用
	ErrListenNotEnabled = errors.New("listen is not enabled")

	// ErrListenNotEnabled 表示未处于正在监听状态
	ErrNoListening = errors.New("listening is not going on")

	// ErrRecordNotEnabled 表示录制未启用
	ErrRecordNotEnabled = errors.New("record is not enabled")
)
