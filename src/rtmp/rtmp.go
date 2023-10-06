// Package rtmp 包含监听器相关的代码。
package rtmp

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"time"

	"github.com/lthibault/jitterbug"

	"github.com/yuhaohwang/bililive-go/src/configs"
	"github.com/yuhaohwang/bililive-go/src/instance"
	"github.com/yuhaohwang/bililive-go/src/interfaces"
	"github.com/yuhaohwang/bililive-go/src/pkg/events"
)

// 定义状态常量用于标记监听器的状态。
const (
	begin uint32 = iota
	pending
	running
	stopped
)

// Rtmp 定义了监听器接口，用于启动和关闭监听器。
type Rtmp interface {
	Start() error
	Close()
}

// NewRtmp 创建一个新的监听器实例。
func NewRtmp(ctx context.Context) Rtmp {
	// 1. 获取应用程序实例 inst。
	inst := instance.GetInstance(ctx)

	// 2. 创建并返回一个新的监听器实例。
	return &rtmp{
		inst:   inst,
		config: inst.Config,
		ed:     inst.EventDispatcher.(events.Dispatcher),
		logger: inst.Logger,
		state:  begin,
		stop:   make(chan struct{}),
	}
}

// rtmp 实现了 Rtmp 接口。
type rtmp struct {
	inst   *instance.Instance
	config *configs.Config
	ed     events.Dispatcher
	logger *interfaces.Logger
	state  uint32
	stop   chan struct{}
}

// Start 启动监听器。
func (l *rtmp) Start() error {
	// 1. 使用原子操作检查并设置监听器的状态为 pending，如果状态不是 begin 则返回 nil。
	if !atomic.CompareAndSwapUint32(&l.state, begin, pending) {
		return nil
	}

	// 2. 在函数退出时将监听器的状态设置为 running。
	defer atomic.CompareAndSwapUint32(&l.state, pending, running)

	// 3. 分发 RtmpStart 事件，表示监听器已经启动。
	l.ed.DispatchEvent(events.NewEvent(RtmpStart, l.config.LiveRooms))

	// 4. 刷新监听器状态。
	l.refresh()

	// 5. 启动监听器的主循环。
	go l.run()

	// 6. 监听配置更改事件。
	l.ed.AddEventListener(ConfigChanged, events.NewEventListener(func(event *events.Event) {
		l.refresh()
	}))
	return nil
}

// Close 关闭监听器。
func (l *rtmp) Close() {
	// 1. 使用原子操作检查并设置监听器的状态为 stopped，如果状态不是 running 则返回。
	if !atomic.CompareAndSwapUint32(&l.state, running, stopped) {
		return
	}

	// 2. 分发 RtmpStop 事件，表示监听器已经关闭。
	l.ed.DispatchEvent(events.NewEvent(RtmpStop, l.config.LiveRooms))

	// 3. 关闭监听器的停止通道。
	close(l.stop)

	// 4. 移除监听配置更改事件。
	l.ed.RemoveAllEventListener(ConfigChanged)
}

type Info struct {
	Id                string `json:"id"`                             // 直播唯一标识
	LiveUrl           string `json:"live_url"`                       // 直播原始 URL
	PlatformCNName    string `json:"platform_cn_name"`               // 平台中文名称
	HostName          string `json:"host_name"`                      // 主播名
	RoomName          string `json:"room_name"`                      // 房间名
	LastStartTime     string `json:"last_start_time,omitempty"`      // 上次开始时间的字符串表示形式
	LastStartTimeUnix int64  `json:"last_start_time_unix,omitempty"` // 上次开始时间的 UNIX 时间戳
	AudioOnly         bool   `json:"audio_only"`                     // 是否仅音频直播
}

// refresh 刷新监听器状态。
func (l *rtmp) refresh() {
	liveRooms := l.config.LiveRooms
	for _, v := range liveRooms {
		if v.Rtmp == "" {
			info, err := l.inst.Lives[v.LiveId].GetInfo()
			if err == nil {
				// 将 info 结构体转换为 JSON 格式
				jsonData, _ := info.MarshalJSON()
				var i Info
				json.Unmarshal([]byte(jsonData), &i)
				l.inst.WebsocketManager.BroadcastMessage("genNewKeyCode", i)
			}
		}
	}
}

// run 启动监听器的主循环。
func (l *rtmp) run() {
	// 1. 创建一个带随机间隔的定时器 ticker。
	ticker := jitterbug.New(
		time.Duration(l.config.Interval)*time.Second,
		jitterbug.Norm{
			Stdev: time.Second * 3,
		},
	)
	defer ticker.Stop()

	// 2. 循环监听多个通道事件。
	for {
		select {
		case <-l.stop:
			return
		case <-ticker.C:
			l.refresh()
		}
	}
}
