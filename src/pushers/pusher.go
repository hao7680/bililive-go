//go:generate mockgen -package pushers -destination mock_test.go github.com/yuhaohwang/bililive-go/src/pushers Pusher,Manager
package pushers

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bluele/gcache"
	"github.com/sirupsen/logrus"
	"github.com/yuhaohwang/bililive-go/src/configs"
	"github.com/yuhaohwang/bililive-go/src/instance"
	"github.com/yuhaohwang/bililive-go/src/interfaces"
	"github.com/yuhaohwang/bililive-go/src/live"
	"github.com/yuhaohwang/bililive-go/src/pkg/events"
	"github.com/yuhaohwang/bililive-go/src/pkg/parser"
	"github.com/yuhaohwang/bililive-go/src/pkg/parser/ffmpeg"
)

const (
	begin uint32 = iota
	pending
	running
	stopped
)

// Pusher 定义 Pusher 接口。
type Pusher interface {
	Start(ctx context.Context) error
	StartTime() time.Time
	GetStatus() (map[string]string, error)
	Close()
}

// pusher 是 Pusher 接口的实现。
type pusher struct {
	Live       live.Live
	OutPutPath string

	CacheFile string
	RtmpUrl   string

	config     *configs.Config
	ed         events.Dispatcher
	logger     *interfaces.Logger
	cache      gcache.Cache
	startTime  time.Time
	parser     parser.Parser
	parserLock *sync.RWMutex

	stop  chan struct{}
	state uint32
}

// NewPusher 创建一个新的 Pusher 实例。
func NewPusher(ctx context.Context, live live.Live) (Pusher, error) {
	inst := instance.GetInstance(ctx)
	return &pusher{
		Live:       live,
		OutPutPath: instance.GetInstance(ctx).Config.OutPutPath,
		config:     inst.Config,
		cache:      inst.Cache,
		startTime:  time.Now(),
		ed:         inst.EventDispatcher.(events.Dispatcher),
		logger:     inst.Logger,
		state:      begin,
		stop:       make(chan struct{}),
		parserLock: new(sync.RWMutex),
	}, nil
}

// tryPush 尝试录制缓存流。
func (r *pusher) tryPush(ctx context.Context) {
	// 获取直播流的URL列表
	urls, err := r.Live.GetStreamUrls()
	if err != nil || len(urls) == 0 {
		r.getLogger().WithError(err).Warn("无法获取直播流URL，将在5秒后重试...")
		time.Sleep(5 * time.Second)
		return
	}

	// 获取应用程序实例
	room, _ := r.config.GetLiveRoomByUrl(r.Live.GetRawUrl())

	if !room.Push && room.Rtmp != "" {
		return
	}

	url := urls[0]

	// 初始化解析器配置
	parserCfg := map[string]string{
		"timeout_in_us": strconv.Itoa(r.config.TimeoutInUs),
	}
	if r.config.Debug {
		parserCfg["debug"] = "true"
	}

	// 根据 URL 初始化解析器
	p, err := parser.New(ffmpeg.Name, parserCfg)
	if err != nil {
		r.getLogger().WithError(err).Error("初始化解析器失败")
		return
	}

	// 设置并关闭当前解析器
	r.setAndCloseParser(p)

	// 推送缓存流并记录结果
	result := r.parser.ParseLiveStream(ctx, url, r.Live, room.Rtmp)
	r.getLogger().Println(result)

	// 记录结束时间

	// 再次保存 JSON 数据到文件

	// 移除空文件

	// 获取 FFmpeg 路径

	// 执行自定义命令或转换
}

// run 启动录制器的主循环。
func (r *pusher) run(ctx context.Context) {
	for {
		select {
		case <-r.stop:
			return
		default:
			r.tryPush(ctx)
		}
	}
}

// getPusher 获取当前解析器。
func (r *pusher) getPusher() parser.Parser {
	r.parserLock.RLock()         // 获取解析器互斥锁，允许多个协程同时读取解析器
	defer r.parserLock.RUnlock() // 在方法结束时释放解析器互斥锁
	return r.parser              // 返回当前的解析器
}

// setAndCloseParser 设置解析器并关闭旧的解析器。
func (r *pusher) setAndCloseParser(p parser.Parser) {
	r.parserLock.Lock()         // 获取互斥锁，防止多个协程同时访问和修改解析器
	defer r.parserLock.Unlock() // 在方法结束时释放互斥锁，确保不会出现死锁

	if r.parser != nil { // 如果当前已经有一个解析器
		r.parser.Stop() // 调用当前解析器的 Stop 方法，关闭旧的解析器
	}
	r.parser = p // 将新的解析器赋值给 r.parser，替换旧的解析器
}

// Start 启动录制器。
func (r *pusher) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapUint32(&r.state, begin, pending) {
		return nil
	}
	go r.run(ctx)
	r.getLogger().Info("Push Start")
	r.ed.DispatchEvent(events.NewEvent(PusherStart, r.Live))
	atomic.CompareAndSwapUint32(&r.state, pending, running)
	return nil
}

// StartTime 返回录制器启动的时间。
func (r *pusher) StartTime() time.Time {
	return r.startTime
}

// Close 关闭录制器。
func (r *pusher) Close() {
	if !atomic.CompareAndSwapUint32(&r.state, running, stopped) {
		return
	}
	close(r.stop)
	if p := r.getPusher(); p != nil {
		p.Stop()
	}
	r.getLogger().Info("Push End")
	r.ed.DispatchEvent(events.NewEvent(PusherStop, r.Live))
}

// getLogger 返回记录器实例。
func (r *pusher) getLogger() *logrus.Entry {
	return r.logger.WithFields(r.getFields())
}

// getFields 返回记录器的字段。
func (r *pusher) getFields() map[string]interface{} {
	obj, err := r.cache.Get(r.Live)
	if err != nil {
		return nil
	}
	info := obj.(*live.Info)
	return map[string]interface{}{
		"host": info.HostName,
		"room": info.RoomName,
	}
}

// GetStatus 获取录制器的状态。
func (r *pusher) GetStatus() (map[string]string, error) {
	statusP, ok := r.getPusher().(parser.StatusParser)
	if !ok {
		return nil, ErrPusherNotSupportStatus
	}
	return statusP.Status()
}
