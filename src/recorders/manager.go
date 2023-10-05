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
		recorders: make(map[live.ID]Recorder),
		cfg:       instance.GetInstance(ctx).Config,
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
	lock      sync.RWMutex
	recorders map[live.ID]Recorder
	cfg       *configs.Config
}

// registryListener 注册事件监听器以响应直播开始、房间名称更改、监听停止等事件。
func (m *manager) registryListener(ctx context.Context, ed events.Dispatcher) {
	// 1. 监听直播开始事件。
	ed.AddEventListener(listeners.LiveStart, events.NewEventListener(func(event *events.Event) {
		live := event.Object.(live.Live) // 类型断言，将event.Object转换为live.Live类型。

		// 获取应用程序实例
		inst := instance.GetInstance(ctx)

		// 获取应用程序配置
		config := inst.Config

		// 获取直播间配置
		room, _ := config.GetLiveRoomByUrl(live.GetRawUrl())

		// 如果未开启监听、录制则退出
		if !room.Listen || !room.Record {
			return
		}

		// 尝试添加一个新的录制器。
		if err := m.AddRecorder(ctx, live); err != nil {
			// 如果添加录制器失败，则记录错误。
			instance.GetInstance(ctx).Logger.Errorf("failed to add recorder, err: %v", err)
		}
	}))

	// 2. 监听房间名称更改事件。
	ed.AddEventListener(listeners.RoomNameChanged, events.NewEventListener(func(event *events.Event) {
		live := event.Object.(live.Live) // 类型断言。
		// 检查是否有对应的录制器。
		if !m.HasRecorder(ctx, live.GetLiveId()) {
			return
		}
		// 尝试重启录制器。
		if err := m.RestartRecorder(ctx, live); err != nil {
			// 如果重启录制器失败，则记录错误。
			instance.GetInstance(ctx).Logger.Errorf("failed to cronRestart recorder, err: %v", err)
		}
	}))

	// 3. 创建一个通用的事件监听器来移除录制器。
	removeEvtListener := events.NewEventListener(func(event *events.Event) {
		live := event.Object.(live.Live) // 类型断言。
		// 检查是否有对应的录制器。
		if !m.HasRecorder(ctx, live.GetLiveId()) {
			return
		}
		// 尝试移除录制器。
		if err := m.RemoveRecorder(ctx, live.GetLiveId()); err != nil {
			// 如果移除录制器失败，则记录错误。
			instance.GetInstance(ctx).Logger.Errorf("failed to remove recorder, err: %v", err)
		}
	})

	// 4. 使用上面创建的通用监听器来监听直播结束和监听停止事件。
	ed.AddEventListener(listeners.LiveEnd, removeEvtListener)
	ed.AddEventListener(listeners.ListenStop, removeEvtListener)
}

// Start 启动 Recorder Manager 并注册事件监听器。
func (m *manager) Start(ctx context.Context) error {
	// 1. 获取当前实例和配置信息。
	inst := instance.GetInstance(ctx)
	// 2. 如果RPC功能启用或有直播活动，则添加一个等待组。
	if inst.Config.RPC.Enable || len(inst.Lives) > 0 {
		inst.WaitGroup.Add(1)
	}
	// 3. 注册事件监听器。
	m.registryListener(ctx, inst.EventDispatcher.(events.Dispatcher))
	return nil
}

// Close 关闭 Recorder Manager。
func (m *manager) Close(ctx context.Context) {
	// 1. 获取锁以同步操作。
	m.lock.Lock()
	defer m.lock.Unlock()
	// 2. 关闭所有活跃的录制器。
	for id, recorder := range m.recorders {
		recorder.Close()
		delete(m.recorders, id)
	}
	// 3. 减少等待组的计数。
	inst := instance.GetInstance(ctx)
	inst.WaitGroup.Done()
}

// AddRecorder 添加一个录制器。
func (m *manager) AddRecorder(ctx context.Context, live live.Live) error {

	// 获取应用程序实例
	inst := instance.GetInstance(ctx)

	// 获取应用程序配置
	config := inst.Config

	// 获取直播间配置
	room, _ := config.GetLiveRoomByUrl(live.GetRawUrl())

	//如果未启用监听，则退出
	if !room.Listen {
		return ErrListenNotEnabled
	}

	//如果未启用录制，则退出
	if !room.Record {
		return ErrRecordNotEnabled
	}

	// 1. 加锁以同步操作。
	m.lock.Lock()
	defer m.lock.Unlock()
	// 2. 检查是否已存在该录制器。
	if _, ok := m.recorders[live.GetLiveId()]; ok {
		return ErrRecorderExist
	}
	// 3. 创建新的录制器。
	recorder, err := newRecorder(ctx, live)
	if err != nil {
		return err
	}
	// 4. 将新录制器添加到管理器。
	m.recorders[live.GetLiveId()] = recorder
	// 5. 如果配置了视频分割，启动定时重启任务。
	if maxDur := m.cfg.VideoSplitStrategies.MaxDuration; maxDur != 0 {
		go m.cronRestart(ctx, live)
	}
	// 6. 启动录制器。
	return recorder.Start(ctx)
}

// cronRestart 定期重新启动录制器，用于分割视频。
func (m *manager) cronRestart(ctx context.Context, live live.Live) {
	// 1. 获取指定的录制器。
	recorder, err := m.GetRecorder(ctx, live.GetLiveId())
	if err != nil {
		return
	}
	// 2. 检查是否需要重新启动录制器。
	if time.Since(recorder.StartTime()) < m.cfg.VideoSplitStrategies.MaxDuration {
		time.AfterFunc(time.Minute/4, func() {
			m.cronRestart(ctx, live)
		})
		return
	}
	// 3. 重新启动录制器。
	if err := m.RestartRecorder(ctx, live); err != nil {
		return
	}
}

// RestartRecorder 重新启动录制器，用于分割视频。
func (m *manager) RestartRecorder(ctx context.Context, live live.Live) error {
	// 1. 移除当前录制器。
	if err := m.RemoveRecorder(ctx, live.GetLiveId()); err != nil {
		return err
	}
	// 2. 添加新的录制器。
	if err := m.AddRecorder(ctx, live); err != nil {
		return err
	}
	return nil
}

// RemoveRecorder 移除录制器。
func (m *manager) RemoveRecorder(ctx context.Context, liveId live.ID) error {
	// 1. 加锁以同步操作。
	m.lock.Lock()
	defer m.lock.Unlock()
	// 2. 检查录制器是否存在。
	recorder, ok := m.recorders[liveId]
	if !ok {
		return ErrRecorderNotExist
	}
	// 3. 关闭录制器。
	recorder.Close()
	// 4. 从管理器中移除录制器。
	delete(m.recorders, liveId)
	return nil
}

// GetRecorder 获取指定录制器。
func (m *manager) GetRecorder(ctx context.Context, liveId live.ID) (Recorder, error) {
	// 1. 获取读锁。
	m.lock.RLock()
	defer m.lock.RUnlock()
	// 2. 尝试获取指定的录制器。
	r, ok := m.recorders[liveId]
	if !ok {
		return nil, ErrRecorderNotExist
	}
	return r, nil
}

// HasRecorder 检查是否存在指定录制器。
func (m *manager) HasRecorder(ctx context.Context, liveId live.ID) bool {
	// 1. 获取读锁。
	m.lock.RLock()
	defer m.lock.RUnlock()
	// 2. 检查录制器是否存在。
	_, ok := m.recorders[liveId]
	return ok
}
