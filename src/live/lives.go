//go:generate mockgen -package mock -destination mock/mock.go github.com/yuhaohwang/bililive-go/src/live Live

// Package live 提供了用于处理各种直播平台的接口和工具函数。
package live

import (
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/bluele/gcache"
)

// 初始化直播平台构建器的映射，用于注册不同平台的构建器实现。
var (
	m                               = make(map[string]Builder)
	InitializingLiveBuilderInstance InitializingLiveBuilder
)

// Register 函数用于注册直播平台的构建器。
func Register(domain string, b Builder) {
	m[domain] = b
}

// getBuilder 函数用于获取指定域名的构建器。
func getBuilder(domain string) (Builder, bool) {
	builder, ok := m[domain]
	return builder, ok
}

// Builder 接口定义了直播平台构建器的方法。
type Builder interface {
	Build(*url.URL, ...Option) (Live, error)
}

// InitializingLiveBuilder 接口定义了初始化直播平台的构建器方法。
type InitializingLiveBuilder interface {
	Build(Live, *url.URL, ...Option) (Live, error)
}

// InitializingFinishedParam 结构体包含了初始化完成后的参数。
type InitializingFinishedParam struct {
	InitializingLive Live
	Live             Live
	Info             *Info
}

// Options 结构体包含了直播平台的选项，如 cookies 和视频质量等。
type Options struct {
	Cookies *cookiejar.Jar
	Quality int
}

// NewOptions 函数用于创建新的选项。
func NewOptions(opts ...Option) (*Options, error) {
	cookieJar, err := cookiejar.New(&cookiejar.Options{})
	if err != nil {
		return nil, err
	}
	options := &Options{Cookies: cookieJar, Quality: 0}
	for _, opt := range opts {
		opt(options)
	}
	return options, nil
}

// MustNewOptions 函数用于创建新的选项，如果出错则 panic。
func MustNewOptions(opts ...Option) *Options {
	options, err := NewOptions(opts...)
	if err != nil {
		panic(err)
	}
	return options
}

// Option 类型定义了设置选项的函数签名。
type Option func(*Options)

// WithKVStringCookies 函数用于设置 cookies 选项，接收键值对字符串。
func WithKVStringCookies(u *url.URL, cookies string) Option {
	return func(opts *Options) {
		cookiesList := make([]*http.Cookie, 0)
		for _, pairStr := range strings.Split(cookies, ";") {
			pairs := strings.SplitN(pairStr, "=", 2)
			if len(pairs) != 2 {
				continue
			}
			cookiesList = append(cookiesList, &http.Cookie{
				Name:  strings.TrimSpace(pairs[0]),
				Value: strings.TrimSpace(pairs[1]),
			})
		}
		opts.Cookies.SetCookies(u, cookiesList)
	}
}

// WithQuality 函数用于设置视频质量选项。
func WithQuality(quality int) Option {
	return func(opts *Options) {
		opts.Quality = quality
	}
}

// ID 类型用于表示直播的唯一标识。
type ID string

// StreamUrlInfo 结构体包含了直播流的相关信息。
type StreamUrlInfo struct {
	Url         *url.URL
	Name        string
	Description string
	Priority    int
}

// Live 接口定义了直播平台的基本方法。
type Live interface {
	SetLiveIdByString(string)
	GetLiveId() ID
	GetRawUrl() string
	GetInfo() (*Info, error)
	GetStreamUrls() ([]*url.URL, error)
	GetPlatformCNName() string
	GetLastStartTime() time.Time
	SetLastStartTime(time.Time)
}

// WrappedLive 结构体用于包装实现了 Live 接口的对象，添加了缓存功能。
type WrappedLive struct {
	Live
	cache gcache.Cache
}

// newWrappedLive 函数用于创建一个包装了 Live 接口对象的 WrappedLive。
func newWrappedLive(live Live, cache gcache.Cache) Live {
	return &WrappedLive{
		Live:  live,
		cache: cache,
	}
}

// GetInfo 方法用于获取直播信息，同时支持缓存功能。
func (w *WrappedLive) GetInfo() (*Info, error) {
	i, err := w.Live.GetInfo()
	if err != nil {
		if info, err2 := w.cache.Get(w); err2 == nil {
			info.(*Info).RoomName = err.Error()
		}
		return nil, err
	}
	if w.cache != nil {
		w.cache.Set(w, i)
	}
	return i, nil
}

// New 函数用于创建一个直播平台实例。
func New(url *url.URL, cache gcache.Cache, opts ...Option) (live Live, err error) {
	builder, ok := getBuilder(url.Host)
	if !ok {
		return nil, errors.New("not support this url")
	}
	live, err = builder.Build(url, opts...)
	if err != nil {
		return
	}
	live = newWrappedLive(live, cache)
	for i := 0; i < 3; i++ {
		var info *Info
		if info, err = live.GetInfo(); err == nil {
			if info.CustomLiveId != "" {
				live.SetLiveIdByString(info.CustomLiveId)
			}
			return
		}
		time.Sleep(1 * time.Second)
	}

	// 当房间初始化失败时，尝试使用初始化构建器
	live, err = InitializingLiveBuilderInstance.Build(live, url, opts...)
	live = newWrappedLive(live, cache)
	live.GetInfo() // 虚拟调用以初始化包装在 WrappedLive 中的缓存
	return
}
