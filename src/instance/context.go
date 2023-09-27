package instance

import (
	"context"
)

// key 是一个自定义的类型，用于标识上下文中存储实例的键。
type key int

// 定义一个常量 Key，用于标识上下文中存储实例的键。可以将其设置为任何唯一的值。
const (
	Key key = 114514
)

// GetInstance 从给定的上下文中获取实例（*Instance），如果存在的话。
func GetInstance(ctx context.Context) *Instance {
	// 通过上下文的 Value 方法尝试获取存储在 Key 键下的实例。
	if s, ok := ctx.Value(Key).(*Instance); ok {
		return s
	}
	// 如果不存在该实例，返回 nil。
	return nil
}
