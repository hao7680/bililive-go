package yizhibo

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/yuhaohwang/requests"

	"github.com/yuhaohwang/bililive-go/src/live"
	"github.com/yuhaohwang/bililive-go/src/live/internal"
	"github.com/yuhaohwang/bililive-go/src/pkg/utils"
)

// 常量定义
const (
	domain = "www.yizhibo.com"
	cnName = "一直播"

	apiUrl = "http://www.yizhibo.com/live/h5api/get_basic_live_info"
)

// 初始化函数，注册一直播的构建器
func init() {
	live.Register(domain, new(builder))
}

// builder 结构体用于构建 Live 类型的直播实例
type builder struct{}

// Build 方法用于构建 Live 类型的直播实例
func (b *builder) Build(url *url.URL, opt ...live.Option) (live.Live, error) {
	return &Live{
		BaseLive: internal.NewBaseLive(url, opt...),
	}, nil
}

// Live 结构体表示一个一直播实例
type Live struct {
	internal.BaseLive
}

// requestRoomInfo 方法用于请求房间信息
func (l *Live) requestRoomInfo() ([]byte, error) {
	scid := strings.Split(strings.Split(l.Url.Path, "/")[2], ".")[0]
	resp, err := requests.Get(apiUrl, live.CommonUserAgent, requests.Query("scid", scid))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, live.ErrRoomNotExist
	}
	body, err := resp.Bytes()
	if err != nil {
		return nil, err
	}
	if gjson.GetBytes(body, "result").Int() != 1 {
		return nil, live.ErrRoomNotExist
	}
	return body, nil
}

// GetInfo 方法用于获取一直播实例的信息
func (l *Live) GetInfo() (info *live.Info, err error) {
	data, err := l.requestRoomInfo()
	if err != nil {
		return nil, err
	}
	info = &live.Info{
		Live:     l,
		HostName: gjson.GetBytes(data, "data.nickname").String(),
		RoomName: gjson.GetBytes(data, "data.live_title").String(),
		Status:   gjson.GetBytes(data, "data.status").Int() == 10,
	}
	return info, nil
}

// GetStreamUrls 方法用于获取一直播实例的流媒体 URL
func (l *Live) GetStreamUrls() (us []*url.URL, err error) {
	resp, err := requests.Get(l.GetRawUrl(), live.CommonUserAgent)
	if err != nil {
		return nil, err
	}
	body, err := resp.Text()
	if err != nil {
		return nil, err
	}
	return utils.GenUrls(utils.Match1(`play_url:"(.*?)",?`, body))
}

// GetPlatformCNName 方法返回平台的中文名称
func (l *Live) GetPlatformCNName() string {
	return cnName
}
