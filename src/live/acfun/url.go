package acfun

import (
	"container/heap"
	"encoding/json"
	"net/url"

	"github.com/yuhaohwang/bililive-go/src/pkg/utils"
)

// representation 表示直播流媒体的一个表示形式
type representation struct {
	Url   string `json:"url"`   // URL 表示直播流的地址
	Level int    `json:"level"` // Level 表示直播流的级别
}

// representations 是多个 representation 的切片
type representations []representation

// Len 返回 representations 的长度
func (r representations) Len() int { return len(r) }

// Less 比较两个 representation 的级别，用于堆排序
func (r representations) Less(i, j int) bool {
	return r[i].Level > r[j].Level
}

// Swap 交换两个 representation 的位置，用于堆排序
func (r representations) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

// Push 将一个 representation 推入堆
func (r *representations) Push(x interface{}) {
	*r = append(*r, x.(representation))
}

// Pop 弹出堆顶的 representation
func (r *representations) Pop() interface{} {
	old := *r
	n := len(old)
	item := old[n-1]
	*r = old[0 : n-1]
	return item
}

// GenUrls 生成直播流媒体的 URL 切片
func (r representations) GenUrls() ([]*url.URL, error) {
	urls := make([]string, r.Len())
	for idx, item := range r {
		urls[idx] = item.Url
	}
	return utils.GenUrls(urls...)
}

// newRepresentationsFromJSON 从 JSON 字符串创建 representations
func newRepresentationsFromJSON(s string) (representations, error) {
	rs := make(representations, 0)
	if err := json.Unmarshal([]byte(s), &rs); err != nil {
		return nil, err
	}
	heap.Fix(&rs, rs.Len()-1) // 修复堆排序
	return rs, nil
}
