package servers

import (
	"github.com/yuhaohwang/bililive-go/src/live"
)

// commonResp 结构体定义了通用的响应结构，用于返回 JSON 格式的响应数据。
type commonResp struct {
	ErrNo  int         `json:"err_no"`  // ErrNo 表示错误代码，用于标识请求的处理状态。
	ErrMsg string      `json:"err_msg"` // ErrMsg 包含了可选的错误消息，用于描述错误的详细信息。
	Data   interface{} `json:"data"`    // Data 包含响应的数据部分，可以是任何类型的数据。
}

// liveSlice 类型是 live.Info 结构体的切片类型，用于对多个直播信息进行排序。
type liveSlice []*live.Info

// Len 方法返回 liveSlice 切片的长度，用于排序接口的实现。
func (c liveSlice) Len() int {
	return len(c)
}

// Swap 方法用于交换 liveSlice 切片中两个元素的位置，用于排序接口的实现。
func (c liveSlice) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

// Less 方法用于比较 liveSlice 切片中两个元素的大小，用于排序接口的实现。
func (c liveSlice) Less(i, j int) bool {
	return c[i].Live.GetLiveId() < c[j].Live.GetLiveId()
}
