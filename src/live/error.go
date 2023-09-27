package live

import (
	"errors"
)

// ErrRoomNotExist 表示房间不存在的错误。
var ErrRoomNotExist = errors.New("room not exists")

// ErrRoomUrlIncorrect 表示房间 URL 不正确的错误。
var ErrRoomUrlIncorrect = errors.New("room url incorrect")

// ErrInternalError 表示内部错误的错误。
var ErrInternalError = errors.New("internal error")
