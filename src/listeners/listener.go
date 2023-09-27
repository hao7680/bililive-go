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
	inst := instance.GetInstance(ctx)
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
	if !atomic.CompareAndSwapUint32(&l.state, begin, pending) {
		return nil
	}
	defer atomic.CompareAndSwapUint32(&l.state, pending, running)

	l.ed.DispatchEvent(events.NewEvent(ListenStart, l.Live))
	l.refresh()
	go l.run()
	return nil
}

// Close 关闭监听器。
func (l *listener) Close() {
	if !atomic.CompareAndSwapUint32(&l.state, running, stopped) {
		return
	}
	l.ed.DispatchEvent(events.NewEvent(ListenStop, l.Live))
	close(l.stop)
}

// refresh 刷新监听器状态。
func (l *listener) refresh() {
	info, err := l.Live.GetInfo()
	if err != nil {
		l.logger.
			WithError(err).
			WithField("url", l.Live.GetRawUrl()).
			Error("failed to load room info")
		return
	}

	var (
		latestStatus = status{roomName: info.RoomName, roomStatus: info.Status}
		evtTyp       events.EventType
		logInfo      string
		fields       = map[string]interface{}{
			"room": info.RoomName,
			"host": info.HostName,
		}
	)
	defer func() { l.status = latestStatus }()
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
	if isStatusChanged {
		l.ed.DispatchEvent(events.NewEvent(evtTyp, l.Live))
		l.logger.WithFields(fields).Info(logInfo)
	}

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
	ticker := jitterbug.New(
		time.Duration(l.config.Interval)*time.Second,
		jitterbug.Norm{
			Stdev: time.Second * 3,
		},
	)
	defer ticker.Stop()

	for {
		select {
		case <-l.stop:
			return
		case <-ticker.C:
			l.refresh()
		}
	}
}
