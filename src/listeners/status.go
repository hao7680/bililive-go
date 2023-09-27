package listeners

type statusEvt uint8

const (
	statusToTrueEvt    statusEvt = 1 << iota // 状态从false变为true的事件
	statusToFalseEvt                         // 状态从true变为false的事件
	roomNameChangedEvt                       // 房间名称更改的事件
)

// status 表示监听器的状态，包括房间名称和房间状态。
type status struct {
	roomName   string // 房间名称
	roomStatus bool   // 房间状态
}

// Diff 比较两个状态之间的差异并返回相应的事件标志。
func (s status) Diff(that status) (res statusEvt) {
	if !s.roomStatus && that.roomStatus {
		res |= statusToTrueEvt
	}
	if s.roomStatus && !that.roomStatus {
		res |= statusToFalseEvt
	}
	if s.roomStatus && that.roomStatus && s.roomName != that.roomName {
		res |= roomNameChangedEvt
	}
	return res
}
