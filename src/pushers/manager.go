package pushers

import (
	"context"
	"sync"

	"github.com/yuhaohwang/bililive-go/src/instance"
	"github.com/yuhaohwang/bililive-go/src/interfaces"
	"github.com/yuhaohwang/bililive-go/src/listeners"
	"github.com/yuhaohwang/bililive-go/src/live"
	"github.com/yuhaohwang/bililive-go/src/pkg/events"
)

// NewManager 创建一个新的 Pusher Manager 实例。
func NewManager(ctx context.Context) Manager {
	pm := &manager{
		pushers: make(map[live.ID]Pusher),
	}
	instance.GetInstance(ctx).PusherManager = pm

	return pm
}

// Manager 定义 Pusher Manager 的接口。
type Manager interface {
	interfaces.Module
	AddPusher(ctx context.Context, live live.Live) error
	RemovePusher(ctx context.Context, liveId live.ID) error
	GetPusher(ctx context.Context, liveId live.ID) (Pusher, error)
	HasPusher(ctx context.Context, liveId live.ID) bool
}

// 用于测试的变量
var (
	newPusher = NewPusher
)

// manager 是 Pusher Manager 的实现。
type manager struct {
	lock    sync.RWMutex
	pushers map[live.ID]Pusher
}

// registryListener 注册事件监听器以响应直播开始、房间名称更改、监听停止等事件。
func (m *manager) registryListener(ctx context.Context, ed events.Dispatcher) {
	// 1. 监听开启推送事件。
	ed.AddEventListener(listeners.LiveStart, events.NewEventListener(func(event *events.Event) {
		live := event.Object.(live.Live) // 类型断言，将event.Object转换为live.Live类型。

		// 获取应用程序实例
		inst := instance.GetInstance(ctx)

		// 获取应用程序配置
		config := inst.Config

		// 获取直播间配置
		room, _ := config.GetLiveRoomByUrl(live.GetRawUrl())

		// 如果未开启推送则退出
		if room.Push && room.Rtmp != "" {
			return
		}

		// 尝试添加一个新的录制器。
		if err := m.AddPusher(ctx, live); err != nil {
			// 如果添加录制器失败，则记录错误。
			instance.GetInstance(ctx).Logger.Errorf("failed to add pusher, err: %v", err)
		}
	}))

	// 2. 监听关闭推送事件。
	ed.AddEventListener(listeners.LiveEnd, events.NewEventListener(func(event *events.Event) {
		live := event.Object.(live.Live) // 类型断言，将event.Object转换为live.Live类型。
		// 检查是否有对应的录制器。
		if !m.HasPusher(ctx, live.GetLiveId()) {
			return
		}
		// 尝试移除录制器。
		if err := m.RemovePusher(ctx, live.GetLiveId()); err != nil {
			// 如果移除录制器失败，则记录错误。
			instance.GetInstance(ctx).Logger.Errorf("failed to remove pusher, err: %v", err)
		}
	}))
}

// Start 启动 Pusher Manager 并注册事件监听器。
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

// Close 关闭 Pusher Manager。
func (m *manager) Close(ctx context.Context) {
	// 1. 获取锁以同步操作。
	m.lock.Lock()
	defer m.lock.Unlock()
	// 2. 关闭所有活跃的录制器。
	for id, pusher := range m.pushers {
		pusher.Close()
		delete(m.pushers, id)
	}
	// 3. 减少等待组的计数。
	inst := instance.GetInstance(ctx)
	inst.WaitGroup.Done()
}

// AddPusher 添加一个录制器。
func (m *manager) AddPusher(ctx context.Context, live live.Live) error {

	// 获取应用程序实例
	inst := instance.GetInstance(ctx)

	// 获取应用程序配置
	config := inst.Config

	// 获取直播间配置
	room, _ := config.GetLiveRoomByUrl(live.GetRawUrl())

	//如果不存在Rtmp 则退出
	if room.Rtmp == "" {
		return ErrRtmpNotExist
	}

	// 1. 加锁以同步操作。
	m.lock.Lock()
	defer m.lock.Unlock()
	// 2. 检查是否已存在该录制器。
	if _, ok := m.pushers[live.GetLiveId()]; ok {
		return ErrPusherExist
	}
	// 3. 创建新的录制器。
	pusher, err := newPusher(ctx, live)
	if err != nil {
		return err
	}
	// 4. 将新录制器添加到管理器。
	m.pushers[live.GetLiveId()] = pusher
	// 5. 启动录制器。
	return pusher.Start(ctx)
}

// RemovePusher 移除录制器。
func (m *manager) RemovePusher(ctx context.Context, liveId live.ID) error {
	// 1. 加锁以同步操作。
	m.lock.Lock()
	defer m.lock.Unlock()
	// 2. 检查录制器是否存在。
	pusher, ok := m.pushers[liveId]
	if !ok {
		return ErrPusherNotExist
	}
	// 3. 关闭录制器。
	pusher.Close()
	// 4. 从管理器中移除录制器。
	delete(m.pushers, liveId)
	return nil
}

// GetPusher 获取指定录制器。
func (m *manager) GetPusher(ctx context.Context, liveId live.ID) (Pusher, error) {
	// 1. 获取读锁。
	m.lock.RLock()
	defer m.lock.RUnlock()
	// 2. 尝试获取指定的录制器。
	r, ok := m.pushers[liveId]
	if !ok {
		return nil, ErrPusherNotExist
	}
	return r, nil
}

// HasPusher 检查是否存在指定录制器。
func (m *manager) HasPusher(ctx context.Context, liveId live.ID) bool {
	// 1. 获取读锁。
	m.lock.RLock()
	defer m.lock.RUnlock()
	// 2. 检查录制器是否存在。
	_, ok := m.pushers[liveId]
	return ok
}
