// Package listeners 包含管理直播监听器的相关代码。
package listeners

import (
	"context"
	"sync"

	"github.com/yuhaohwang/bililive-go/src/instance"
	"github.com/yuhaohwang/bililive-go/src/interfaces"
	"github.com/yuhaohwang/bililive-go/src/live"
	"github.com/yuhaohwang/bililive-go/src/pkg/events"
)

// for test
var newListener = NewListener

// NewManager 创建一个新的监听器管理器。
func NewManager(ctx context.Context) Manager {
	// 1. 创建一个监听器管理器实例 lm。
	lm := &manager{
		savers: make(map[live.ID]Listener),
	}

	// 2. 获取应用程序实例 inst。
	instance.GetInstance(ctx).ListenerManager = lm

	// 3. 返回监听器管理器实例。
	return lm
}

// Manager 定义了监听器管理器的接口，它实现了 interfaces.Module 接口。
type Manager interface {
	interfaces.Module
	AddListener(ctx context.Context, live live.Live) error
	RemoveListener(ctx context.Context, liveId live.ID) error
	GetListener(ctx context.Context, liveId live.ID) (Listener, error)
	HasListener(ctx context.Context, liveId live.ID) bool
}

// manager 实现了监听器管理器的接口。
type manager struct {
	lock   sync.RWMutex
	savers map[live.ID]Listener
}

// registryListener 注册监听器，用于监听直播房间初始化完成事件。
func (m *manager) registryListener(ctx context.Context, ed events.Dispatcher) {
	// 1. 添加一个事件监听器，监听 "RoomInitializingFinished" 事件。
	ed.AddEventListener(RoomInitializingFinished, events.NewEventListener(func(event *events.Event) {
		// 2. 从事件对象中获取初始化完成参数。
		param := event.Object.(live.InitializingFinishedParam)
		initializingLive := param.InitializingLive
		live := param.Live
		info := param.Info

		// 3. 如果 info 中包含自定义的 LiveId，则将 live 的 LiveId 设置为自定义值。
		if info.CustomLiveId != "" {
			live.SetLiveIdByString(info.CustomLiveId)
		}

		// 4. 获取应用程序实例 inst。
		inst := instance.GetInstance(ctx)
		logger := inst.Logger

		// 5. 将 live 添加到应用程序实例的 Lives 列表中。
		inst.Lives[live.GetLiveId()] = live

		// 6. 通过直播的原始URL获取房间信息。
		room, err := inst.Config.GetLiveRoomByUrl(live.GetRawUrl())
		if err != nil {
			// 7. 如果获取房间信息出错，记录错误日志并抛出异常。
			logger.WithFields(map[string]interface{}{
				"room": live.GetRawUrl(),
			}).Error(err)
			panic(err)
		}

		// 8. 设置房间的 LiveId 为当前 live 的 LiveId。
		room.LiveId = live.GetLiveId()

		// 9. 如果房间正在监听中，尝试替换监听器。
		if room.Listen {
			if err := m.replaceListener(ctx, initializingLive, live); err != nil {
				// 10. 如果替换监听器出错，记录错误日志。
				logger.WithFields(map[string]interface{}{
					"url": live.GetRawUrl(),
				}).Error(err)
			}
		}
	}))
}

// Start 启动监听器管理器。
func (m *manager) Start(ctx context.Context) error {
	// 1. 获取应用程序实例 inst。
	inst := instance.GetInstance(ctx)

	// 2. 检查是否启用了 RPC 或者是否有直播信息。
	if inst.Config.RPC.Enable || len(inst.Lives) > 0 {
		// 3. 如果满足条件，将等待组计数加1。
		inst.WaitGroup.Add(1)
	}

	// 4. 注册监听器，监听 "RoomInitializingFinished" 事件。
	m.registryListener(ctx, inst.EventDispatcher.(events.Dispatcher))
	return nil
}

// Close 关闭监听器管理器。
func (m *manager) Close(ctx context.Context) {
	// 1. 获取互斥锁。
	m.lock.Lock()
	defer m.lock.Unlock()

	// 2. 遍历所有监听器，关闭它们，并从管理器中移除。
	for id, listener := range m.savers {
		listener.Close()
		delete(m.savers, id)
	}

	// 3. 获取应用程序实例 inst。
	inst := instance.GetInstance(ctx)

	// 4. 从等待组中减去1。
	inst.WaitGroup.Done()
}

// AddListener 添加一个监听器到管理器。
func (m *manager) AddListener(ctx context.Context, live live.Live) error {
	// 1. 获取互斥锁。
	m.lock.Lock()
	defer m.lock.Unlock()

	// 2. 检查是否已经存在相同 LiveId 的监听器。
	if _, ok := m.savers[live.GetLiveId()]; ok {
		return ErrListenerExist
	}

	// 3. 创建新的监听器，并将它添加到管理器中。
	listener := newListener(ctx, live)
	m.savers[live.GetLiveId()] = listener

	// 4. 启动新添加的监听器。
	return listener.Start()
}

// RemoveListener 从管理器中移除指定的监听器。
func (m *manager) RemoveListener(ctx context.Context, liveId live.ID) error {
	// 1. 获取互斥锁。
	m.lock.Lock()
	defer m.lock.Unlock()

	// 2. 检查是否存在具有指定 LiveId 的监听器。
	listener, ok := m.savers[liveId]
	if !ok {
		return ErrListenerNotExist
	}

	// 3. 关闭监听器并从管理器中移除。
	listener.Close()
	delete(m.savers, liveId)
	return nil
}

// replaceListener 替换监听器。
func (m *manager) replaceListener(ctx context.Context, oldLive live.Live, newLive live.Live) error {
	// 1. 获取互斥锁。
	m.lock.Lock()
	defer m.lock.Unlock()

	// 2. 获取旧 Live 的 LiveId。
	oldLiveId := oldLive.GetLiveId()

	// 3. 检查是否存在具有旧 LiveId 的监听器。
	oldListener, ok := m.savers[oldLiveId]
	if !ok {
		return ErrListenerNotExist
	}

	// 4. 关闭旧监听器。
	oldListener.Close()

	// 5. 创建新监听器。
	newListener := newListener(ctx, newLive)

	// 6. 如果新旧 LiveId 相同，将新监听器替换旧监听器。
	if oldLiveId == newLive.GetLiveId() {
		m.savers[oldLiveId] = newListener
	} else {
		// 7. 如果新旧 LiveId 不同，从管理器中删除旧监听器，将新监听器添加到管理器中。
		delete(m.savers, oldLiveId)
		m.savers[newLive.GetLiveId()] = newListener
	}

	// 8. 启动新监听器。
	return newListener.Start()
}

// GetListener 获取指定的监听器。
func (m *manager) GetListener(ctx context.Context, liveId live.ID) (Listener, error) {
	// 1. 获取读锁。
	m.lock.RLock()
	defer m.lock.RUnlock()

	// 2. 检查是否存在具有指定 LiveId 的监听器。
	listener, ok := m.savers[liveId]
	if !ok {
		return nil, ErrListenerNotExist
	}

	// 3. 返回找到的监听器。
	return listener, nil
}

// HasListener 检查是否存在指定的监听器。
func (m *manager) HasListener(ctx context.Context, liveId live.ID) bool {
	// 1. 获取读锁。
	m.lock.RLock()
	defer m.lock.RUnlock()

	// 2. 检查是否存在具有指定 LiveId 的监听器。
	_, ok := m.savers[liveId]
	return ok
}
