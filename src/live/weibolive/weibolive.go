package weibolive

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/yuhaohwang/bililive-go/src/pkg/utils"
	"github.com/yuhaohwang/requests"

	"github.com/yuhaohwang/bililive-go/src/live"
	"github.com/yuhaohwang/bililive-go/src/live/internal"
)

// 常量定义
const (
	domain = "weibo.com"
	cnName = "微博直播"

	liveurl = "https://weibo.com/l/!/2/wblive/room/show_pc_live.json?live_id="
)

// 初始化函数，注册微博直播的构建器
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

// Live 结构体表示一个微博直播实例
type Live struct {
	internal.BaseLive
	roomID string
}

// getRoomInfo 方法用于获取直播房间信息
func (l *Live) getRoomInfo() ([]byte, error) {
	paths := strings.Split(l.Url.Path, "/")
	if len(paths) < 5 {
		return nil, live.ErrRoomUrlIncorrect
	}
	roomid := paths[5]
	l.roomID = roomid

	resp, err := requests.Get(liveurl+roomid,
		live.CommonUserAgent,
		requests.Headers(map[string]interface{}{
			"Referer": l.Url,
		}))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, live.ErrRoomNotExist
	}
	body, err := resp.Bytes()
	if err != nil || gjson.GetBytes(body, "error_code").Int() != 0 {
		return nil, live.ErrRoomNotExist
	}
	return body, nil
}

// GetInfo 方法用于获取微博直播实例的信息
func (l *Live) GetInfo() (info *live.Info, err error) {
	body, err := l.getRoomInfo()
	if err != nil {
		return nil, live.ErrRoomNotExist
	}
	info = &live.Info{
		Live:         l,
		HostName:     gjson.GetBytes(body, "data.user.screenName").String(),
		RoomName:     gjson.GetBytes(body, "data.title").String(),
		Status:       gjson.GetBytes(body, "data.status").String() == "1",
		CustomLiveId: "weibolive/" + l.roomID,
	}
	return info, nil
}

// GetStreamUrls 方法用于获取微博直播实例的流媒体 URL
func (l *Live) GetStreamUrls() (us []*url.URL, err error) {
	body, err := l.getRoomInfo()
	if err != nil {
		return nil, live.ErrRoomNotExist
	}

	streamurl := gjson.GetBytes(body, "data.live_origin_flv_url").String()
	return utils.GenUrls(streamurl)
}

// GetPlatformCNName 方法返回平台的中文名称
func (l *Live) GetPlatformCNName() string {
	return cnName
}
