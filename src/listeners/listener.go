//go:generate mockgen -package listeners -destination mock_test.go github.com/yuhaohwang/bililive-go/src/listeners Listener,Manager

// Package listeners 包含监听器相关的代码。
package listeners

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/lthibault/jitterbug"

	"github.com/yuhaohwang/bililive-go/src/configs"
	"github.com/yuhaohwang/bililive-go/src/instance"
	"github.com/yuhaohwang/bililive-go/src/interfaces"
	"github.com/yuhaohwang/bililive-go/src/live"
	"github.com/yuhaohwang/bililive-go/src/live/system"
	"github.com/yuhaohwang/bililive-go/src/pkg/events"
)

// 定义状态常量用于标记监听器的状态。
const (
	begin uint32 = iota
	pending
	running
	stopped
)

// Listener 定义了监听器接口，用于启动和关闭监听器。
type Listener interface {
	Start() error
	Close()
}

// NewListener 创建一个新的监听器实例。
func NewListener(ctx context.Context, live live.Live) Listener {
	// 1. 获取应用程序实例 inst。
	inst := instance.GetInstance(ctx)

	// 2. 创建并返回一个新的监听器实例。
	return &listener{
		Live:   live,
		status: status{},
		config: inst.Config,
		stop:   make(chan struct{}),
		ed:     inst.EventDispatcher.(events.Dispatcher),
		logger: inst.Logger,
		state:  begin,
	}
}

// listener 实现了 Listener 接口。
type listener struct {
	Live   live.Live
	status status

	config *configs.Config
	ed     events.Dispatcher
	logger *interfaces.Logger

	state uint32
	stop  chan struct{}
}

// Start 启动监听器。
func (l *listener) Start() error {
	// 1. 使用原子操作检查并设置监听器的状态为 pending，如果状态不是 begin 则返回 nil。
	if !atomic.CompareAndSwapUint32(&l.state, begin, pending) {
		return nil
	}

	// 2. 在函数退出时将监听器的状态设置为 running。
	defer atomic.CompareAndSwapUint32(&l.state, pending, running)

	// 3. 分发 ListenStart 事件，表示监听器已经启动。
	l.ed.DispatchEvent(events.NewEvent(ListenStart, l.Live))

	// 4. 刷新监听器状态。
	l.refresh()

	// 5. 启动监听器的主循环。
	go l.run()
	return nil
}

// Close 关闭监听器。
func (l *listener) Close() {
	// 1. 使用原子操作检查并设置监听器的状态为 stopped，如果状态不是 running 则返回。
	if !atomic.CompareAndSwapUint32(&l.state, running, stopped) {
		return
	}

	// 2. 分发 ListenStop 事件，表示监听器已经关闭。
	l.ed.DispatchEvent(events.NewEvent(ListenStop, l.Live))

	// 3. 关闭监听器的停止通道。
	close(l.stop)
}

// refresh 刷新监听器状态。
func (l *listener) refresh() {
	// 1. 获取直播信息和可能的错误。
	info, err := l.Live.GetInfo()
	if err != nil {
		l.logger.
			WithError(err).
			WithField("url", l.Live.GetRawUrl()).
			Error("failed to load room info")
		return
	}

	// 2. 创建最新状态 latestStatus。
	var (
		latestStatus = status{roomName: info.RoomName, roomStatus: info.Status}
		evtTyp       events.EventType
		logInfo      string
		fields       = map[string]interface{}{
			"room": info.RoomName,
			"host": info.HostName,
		}
	)

	// 3. 使用延迟函数来设置监听器状态为 latestStatus。
	defer func() { l.status = latestStatus }()

	// 4. 检查是否状态发生了变化，判断是否需要分发事件。
	isStatusChanged := true
	switch l.status.Diff(latestStatus) {
	case 0:
		isStatusChanged = false
	case statusToTrueEvt:
		l.Live.SetLastStartTime(time.Now())
		evtTyp = LiveStart
		logInfo = "Live Start"
	case statusToFalseEvt:
		evtTyp = LiveEnd
		logInfo = "Live end"
	case roomNameChangedEvt:
		if !l.config.VideoSplitStrategies.OnRoomNameChanged {
			return
		}
		evtTyp = RoomNameChanged
		logInfo = "Room name was changed"
	}

	// 5. 如果状态发生了变化，分发相应的事件，并记录日志。
	if isStatusChanged {
		l.ed.DispatchEvent(events.NewEvent(evtTyp, l.Live))
		l.logger.WithFields(fields).Info(logInfo)
	}

	// 6. 检查是否直播正在初始化中。
	if info.Initializing {
		initializingLive := l.Live.(*live.WrappedLive).Live.(*system.InitializingLive)
		info, err = initializingLive.OriginalLive.GetInfo()
		if err == nil {
			l.ed.DispatchEvent(events.NewEvent(RoomInitializingFinished, live.InitializingFinishedParam{
				InitializingLive: l.Live,
				Live:             initializingLive.OriginalLive,
				Info:             info,
			}))
		}
	}
}

// run 启动监听器的主循环。
func (l *listener) run() {
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
