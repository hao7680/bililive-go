package instance

import (
	"sync"

	"github.com/bluele/gcache"

	"github.com/yuhaohwang/bililive-go/src/configs"
	"github.com/yuhaohwang/bililive-go/src/interfaces"
	"github.com/yuhaohwang/bililive-go/src/live"
)

// Instance 结构体包含了应用程序中的各种组件和配置信息。
type Instance struct {
	WaitGroup       sync.WaitGroup        // WaitGroup 用于等待各个 goroutine 的完成。
	Config          *configs.Config       // Config 包含应用程序的配置信息。
	Logger          *interfaces.Logger    // Logger 是日志记录器接口，用于记录日志。
	Lives           map[live.ID]live.Live // Lives 包含所有 live.Live 接口的实例。
	Cache           gcache.Cache          // Cache 是一个缓存实例，用于存储临时数据。
	Server          interfaces.Module     // Server 是应用程序的服务器模块。
	EventDispatcher interfaces.Module     // EventDispatcher 是事件分发器模块。
	ListenerManager interfaces.Module     // ListenerManager 是监听器管理器模块。
	RecorderManager interfaces.Module     // RecorderManager 是录制器管理器模块。
}
