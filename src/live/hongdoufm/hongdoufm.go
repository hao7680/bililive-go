package hongdoufm

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

const (
	domain = "www.hongdoufm.com"
	cnName = "克拉克拉"

	roomInitUrl = "https://live.hongdoulive.com/LiveRoom/getRoomInfo?roomId="
)

// init 函数用于在程序启动时注册克拉克拉直播平台的 builder
func init() {
	live.Register(domain, new(builder))
}

// builder 结构体实现 live.Builder 接口，用于构建克拉克拉直播平台的直播实例
type builder struct{}

// Build 方法根据传入的 URL 构建克拉克拉直播实例
func (b *builder) Build(url *url.URL, opt ...live.Option) (live.Live, error) {
	return &Live{
		BaseLive: internal.NewBaseLive(url, opt...),
	}, nil
}

// Live 结构体表示一个克拉克拉直播实例
type Live struct {
	internal.BaseLive
	roomID string
}

// 克拉克拉平台直播间连接有两种格式
// 1、https://www.hongdoufm.com/room/roomid 这是直播间列表中的房间地址
// 2、http://www.hongdoufm.com/PcLive/index/detail?id=roomid 这是实际直播间地址，上述地址会经过302跳转

// getRoomInfo 方法获取克拉克拉直播房间信息
func (l *Live) getRoomInfo() ([]byte, error) {
	if strings.Contains(l.Url.String(), "?") {
		// 实际直播间地址
		result, _ := url.ParseQuery(l.Url.RawQuery)
		roomid := result.Get("id")
		l.roomID = roomid
	} else {
		// 列表直播间地址
		paths := strings.Split(l.Url.Path, "/")
		if len(paths) < 2 {
			return nil, live.ErrRoomUrlIncorrect
		}
		roomid := paths[2]
		l.roomID = roomid
	}

	resp, err := requests.Get(roomInitUrl + l.roomID)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, live.ErrRoomNotExist
	}
	body, err := resp.Bytes()
	if err != nil || gjson.GetBytes(body, "h.code").Int() != 200 {
		return nil, live.ErrRoomNotExist
	}
	return body, nil
}

// GetInfo 方法获取克拉克拉直播房间的信息，包括主播名称、房间名称和直播状态
func (l *Live) GetInfo() (info *live.Info, err error) {
	body, err := l.getRoomInfo()
	if err != nil {
		return nil, live.ErrRoomNotExist
	}
	info = &live.Info{
		Live:         l,
		HostName:     gjson.GetBytes(body, "b.userInfo.nickname").String(),
		RoomName:     gjson.GetBytes(body, "b.title").String(),
		Status:       gjson.GetBytes(body, "b.status").Int() == 4,
		CustomLiveId: "hongdoufm/" + l.roomID,
	}
	return info, nil
}

// GetStreamUrls 方法获取克拉克拉直播房间的流媒体 URL
func (l *Live) GetStreamUrls() (us []*url.URL, err error) {
	body, err := l.getRoomInfo()
	if err != nil {
		return nil, live.ErrRoomNotExist
	}
	return utils.GenUrls(gjson.GetBytes(body, "b.flvPlayUrl").String())
}

// GetPlatformCNName 方法获取克拉克拉直播平台的中文名称
func (l *Live) GetPlatformCNName() string {
	return cnName
}
