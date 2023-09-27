package recorders

import "errors"

var (
	// ErrRecorderExist 表示已存在的记录器错误。
	ErrRecorderExist = errors.New("recorder is exist")

	// ErrRecorderNotExist 表示不存在的记录器错误。
	ErrRecorderNotExist = errors.New("recorder is not exist")

	// ErrParserNotSupportStatus 表示解析器不支持获取状态的错误。
	ErrParserNotSupportStatus = errors.New("parser not support get status")
)
