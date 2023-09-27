//go:generate mockgen -package mock -destination mock/mock.go github.com/yuhaohwang/bililive-go/src/pkg/parser Parser
package parser

import (
	"context"
	"errors"
	"net/url"

	"github.com/yuhaohwang/bililive-go/src/live"
)

// Builder 定义了解析器构建器的接口。
type Builder interface {
	Build(cfg map[string]string) (Parser, error)
}

// Parser 定义了解析器的接口。
type Parser interface {
	ParseLiveStream(ctx context.Context, url *url.URL, live live.Live, file string) error
	Stop() error
}

// StatusParser 扩展了Parser接口，增加了Status方法。
type StatusParser interface {
	Parser
	Status() (map[string]string, error)
}

var m = make(map[string]Builder)

// Register 用于注册解析器构建器。
func Register(name string, b Builder) {
	m[name] = b
}

// New 根据名称和配置创建新的解析器实例。
func New(name string, cfg map[string]string) (Parser, error) {
	builder, ok := m[name]
	if !ok {
		return nil, errors.New("未知解析器")
	}
	return builder.Build(cfg)
}
