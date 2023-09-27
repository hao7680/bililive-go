// 由 MockGen 生成的代码。请勿编辑。
// 源自 github.com/yuhaohwang/bililive-go/src/pkg/events (接口：Dispatcher)

// 包 mock 是一个生成的 GoMock 包。
package mock

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	events "github.com/yuhaohwang/bililive-go/src/pkg/events"
)

// MockDispatcher 是 Dispatcher 接口的模拟。
type MockDispatcher struct {
	ctrl     *gomock.Controller
	recorder *MockDispatcherMockRecorder
}

// MockDispatcherMockRecorder 是 MockDispatcher 的模拟记录器。
type MockDispatcherMockRecorder struct {
	mock *MockDispatcher
}

// NewMockDispatcher 创建一个新的模拟实例。
func NewMockDispatcher(ctrl *gomock.Controller) *MockDispatcher {
	mock := &MockDispatcher{ctrl: ctrl}
	mock.recorder = &MockDispatcherMockRecorder{mock}
	return mock
}

// EXPECT 返回一个对象，允许调用者指示预期的使用。
func (m *MockDispatcher) EXPECT() *MockDispatcherMockRecorder {
	return m.recorder
}

// AddEventListener 模拟基本方法。
func (m *MockDispatcher) AddEventListener(arg0 events.EventType, arg1 *events.EventListener) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AddEventListener", arg0, arg1)
}

// AddEventListener 指示对 AddEventListener 的预期调用。
func (mr *MockDispatcherMockRecorder) AddEventListener(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddEventListener", reflect.TypeOf((*MockDispatcher)(nil).AddEventListener), arg0, arg1)
}

// Close 模拟基本方法。
func (m *MockDispatcher) Close(arg0 context.Context) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Close", arg0)
}

// Close 指示对 Close 的预期调用。
func (mr *MockDispatcherMockRecorder) Close(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockDispatcher)(nil).Close), arg0)
}

// DispatchEvent 模拟基本方法。
func (m *MockDispatcher) DispatchEvent(arg0 *events.Event) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "DispatchEvent", arg0)
}

// DispatchEvent 指示对 DispatchEvent 的预期调用。
func (mr *MockDispatcherMockRecorder) DispatchEvent(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DispatchEvent", reflect.TypeOf((*MockDispatcher)(nil).DispatchEvent), arg0)
}

// RemoveAllEventListener 模拟基本方法。
func (m *MockDispatcher) RemoveAllEventListener(arg0 events.EventType) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "RemoveAllEventListener", arg0)
}

// RemoveAllEventListener 指示对 RemoveAllEventListener 的预期调用。
func (mr *MockDispatcherMockRecorder) RemoveAllEventListener(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveAllEventListener", reflect.TypeOf((*MockDispatcher)(nil).RemoveAllEventListener), arg0)
}

// RemoveEventListener 模拟基本方法。
func (m *MockDispatcher) RemoveEventListener(arg0 events.EventType, arg1 *events.EventListener) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "RemoveEventListener", arg0, arg1)
}

// RemoveEventListener 指示对 RemoveEventListener 的预期调用。
func (mr *MockDispatcherMockRecorder) RemoveEventListener(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveEventListener", reflect.TypeOf((*MockDispatcher)(nil).RemoveEventListener), arg0, arg1)
}

// Start 模拟基本方法。
func (m *MockDispatcher) Start(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Start", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Start 指示对 Start 的预期调用。
func (mr *MockDispatcherMockRecorder) Start(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockDispatcher)(nil).Start), arg0)
}
