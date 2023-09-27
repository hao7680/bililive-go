package interfaces

import (
	"context"

	"github.com/sirupsen/logrus"
)

// Module 接口定义了应用程序中各种模块应该实现的方法。
type Module interface {
	// Start 启动模块的方法，接收一个上下文 ctx 作为参数，返回一个可能的错误。
	Start(ctx context.Context) error

	// Close 关闭模块的方法，接收一个上下文 ctx 作为参数。
	Close(ctx context.Context)
}

// Logger 结构体包装了 logrus.Logger，用于在应用程序中进行日志记录。
type Logger struct {
	*logrus.Logger
}
