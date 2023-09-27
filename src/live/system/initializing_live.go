package system

import (
	"net/url"

	"github.com/yuhaohwang/bililive-go/src/live"
	"github.com/yuhaohwang/bililive-go/src/live/internal"
)

// 初始化模块
func init() {
	live.InitializingLiveBuilderInstance = new(builder)
}

// builder 结构体用于构建 InitializingLive 类型的直播实例
type builder struct{}

// Build 方法用于构建 InitializingLive 类型的直播实例
func (b *builder) Build(live live.Live, url *url.URL, opt ...live.Option) (live.Live, error) {
	return &InitializingLive{
		BaseLive:     internal.NewBaseLive(url, opt...),
		OriginalLive: live,
	}, nil
}

// InitializingLive 结构体表示一个正在初始化的直播实例
type InitializingLive struct {
	internal.BaseLive
	OriginalLive live.Live
}

// GetInfo 方法用于获取 InitializingLive 直播实例的信息
func (l *InitializingLive) GetInfo() (info *live.Info, err error) {
	err = nil
	info = &live.Info{
		Live:         l,
		HostName:     "",
		RoomName:     l.GetRawUrl(),
		Status:       false,
		Initializing: true,
	}
	return
}

// GetStreamUrls 方法用于获取 InitializingLive 直播实例的流媒体 URL
func (l *InitializingLive) GetStreamUrls() (us []*url.URL, err error) {
	us = make([]*url.URL, 0)
	err = nil
	return
}

// GetPlatformCNName 方法返回平台的中文名称
func (l *InitializingLive) GetPlatformCNName() string {
	return ""
}
