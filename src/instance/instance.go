package instance

import (
	"sync"

	"github.com/bluele/gcache"

	"github.com/yuhao_hwang/bililive-go/src/configs"
	"github.com/yuhao_hwang/bililive-go/src/interfaces"
	"github.com/yuhao_hwang/bililive-go/src/live"
)

type Instance struct {
	WaitGroup       sync.WaitGroup
	Config          *configs.Config
	Logger          *interfaces.Logger
	Lives           map[live.ID]live.Live
	Cache           gcache.Cache
	Server          interfaces.Module
	EventDispatcher interfaces.Module
	ListenerManager interfaces.Module
	RecorderManager interfaces.Module
}
