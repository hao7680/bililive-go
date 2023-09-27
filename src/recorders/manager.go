package recorders

import (
	"context"
	"sync"
	"time"

	"github.com/yuhaohwang/bililive-go/src/configs"
	"github.com/yuhaohwang/bililive-go/src/instance"
	"github.com/yuhaohwang/bililive-go/src/interfaces"
	"github.com/yuhaohwang/bililive-go/src/listeners"
	"github.com/yuhaohwang/bililive-go/src/live"
	"github.com/yuhaohwang/bililive-go/src/pkg/events"
)

// NewManager 创建一个新的 Recorder Manager 实例。
func NewManager(ctx context.Context) Manager {
	rm := &manager{
		savers: make(map[live.ID]Recorder),
		cfg:    instance.GetInstance(ctx).Config,
	}
	instance.GetInstance(ctx).RecorderManager = rm

	return rm
}

// Manager 定义 Recorder Manager 的接口。
type Manager interface {
	interfaces.Module
	AddRecorder(ctx context.Context, live live.Live) error
	RemoveRecorder(ctx context.Context, liveId live.ID) error
	RestartRecorder(ctx context.Context, liveId live.Live) error
	GetRecorder(ctx context.Context, liveId live.ID) (Recorder, error)
	HasRecorder(ctx context.Context, liveId live.ID) bool
}

// 用于测试的变量
var (
	newRecorder = NewRecorder
)

// manager 是 Recorder Manager 的实现。
type manager struct {
	lock   sync.RWMutex
	savers map[live.ID]Recorder
	cfg    *configs.Config
}

// registryListener 注册事件监听器以响应直播开始、房间名称更改、监听停止等事件。
func (m *manager) registryListener(ctx context.Context, ed events.Dispatcher) {
	ed.AddEventListener(listeners.LiveStart, events.NewEventListener(func(event *events.Event) {
		live := event.Object.(live.Live)
		if err := m.AddRecorder(ctx, live); err != nil {
			instance.GetInstance(ctx).Logger.Errorf("failed to add recorder, err: %v", err)
		}
	}))

	ed.AddEventListener(listeners.RoomNameChanged, events.NewEventListener(func(event *events.Event) {
		live := event.Object.(live.Live)
		if !m.HasRecorder(ctx, live.GetLiveId()) {
			return
		}
		if err := m.RestartRecorder(ctx, live); err != nil {
			instance.GetInstance(ctx).Logger.Errorf("failed to cronRestart recorder, err: %v", err)
		}
	}))

	removeEvtListener := events.NewEventListener(func(event *events.Event) {
		live := event.Object.(live.Live)
		if !m.HasRecorder(ctx, live.GetLiveId()) {
			return
		}
		if err := m.RemoveRecorder(ctx, live.GetLiveId()); err != nil {
			instance.GetInstance(ctx).Logger.Errorf("failed to remove recorder, err: %v", err)
		}
	})
	ed.AddEventListener(listeners.LiveEnd, removeEvtListener)
	ed.AddEventListener(listeners.ListenStop, removeEvtListener)
}

// Start 启动 Recorder Manager 并注册事件监听器。
func (m *manager) Start(ctx context.Context) error {
	inst := instance.GetInstance(ctx)
	if inst.Config.RPC.Enable || len(inst.Lives) > 0 {
		inst.WaitGroup.Add(1)
	}
	m.registryListener(ctx, inst.EventDispatcher.(events.Dispatcher))
	return nil
}

// Close 关闭 Recorder Manager。
func (m *manager) Close(ctx context.Context) {
	m.lock.Lock()
	defer m.lock.Unlock()
	for id, recorder := range m.savers {
		recorder.Close()
		delete(m.savers, id)
	}
	inst := instance.GetInstance(ctx)
	inst.WaitGroup.Done()
}

// AddRecorder 添加一个录制器。
func (m *manager) AddRecorder(ctx context.Context, live live.Live) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if _, ok := m.savers[live.GetLiveId()]; ok {
		return ErrRecorderExist
	}
	recorder, err := newRecorder(ctx, live)
	if err != nil {
		return err
	}
	m.savers[live.GetLiveId()] = recorder

	if maxDur := m.cfg.VideoSplitStrategies.MaxDuration; maxDur != 0 {
		go m.cronRestart(ctx, live)
	}
	return recorder.Start(ctx)
}

// cronRestart 定期重新启动录制器，用于分割视频。
func (m *manager) cronRestart(ctx context.Context, live live.Live) {
	recorder, err := m.GetRecorder(ctx, live.GetLiveId())
	if err != nil {
		return
	}
	if time.Now().Sub(recorder.StartTime()) < m.cfg.VideoSplitStrategies.MaxDuration {
		time.AfterFunc(time.Minute/4, func() {
			m.cronRestart(ctx, live)
		})
		return
	}
	if err := m.RestartRecorder(ctx, live); err != nil {
		return
	}
}

// RestartRecorder 重新启动录制器，用于分割视频。
func (m *manager) RestartRecorder(ctx context.Context, live live.Live) error {
	if err := m.RemoveRecorder(ctx, live.GetLiveId()); err != nil {
		return err
	}
	if err := m.AddRecorder(ctx, live); err != nil {
		return err
	}
	return nil
}

// RemoveRecorder 移除录制器。
func (m *manager) RemoveRecorder(ctx context.Context, liveId live.ID) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	recorder, ok := m.savers[liveId]
	if !ok {
		return ErrRecorderNotExist
	}
	recorder.Close()
	delete(m.savers, liveId)
	return nil
}

// GetRecorder 获取指定录制器。
func (m *manager) GetRecorder(ctx context.Context, liveId live.ID) (Recorder, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	r, ok := m.savers[liveId]
	if !ok {
		return nil, ErrRecorderNotExist
	}
	return r, nil
}

// HasRecorder 检查是否存在指定录制器。
func (m *manager) HasRecorder(ctx context.Context, liveId live.ID) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	_, ok := m.savers[liveId]
	return ok
}
