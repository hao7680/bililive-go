package listeners

import "errors"

// ErrListenerExist 表示已经存在与某个直播流关联的监听器的错误。
var ErrListenerExist = errors.New("该直播已经存在监听器")

// ErrListenerNotExist 表示某个直播流没有与之关联的监听器的错误。
var ErrListenerNotExist = errors.New("该直播没有监听器")
