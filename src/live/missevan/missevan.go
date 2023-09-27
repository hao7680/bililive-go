package missevan

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
	domain = "fm.missevan.com"
	cnName = "猫耳"

	roomInitUrl = "https://fm.missevan.com/api/v2/live/"
)

func init() {
	live.Register(domain, new(builder))
}

type builder struct{}

func (b *builder) Build(url *url.URL, opt ...live.Option) (live.Live, error) {
	return &Live{
		BaseLive: internal.NewBaseLive(url, opt...),
	}, nil
}

type Live struct {
	internal.BaseLive
}

// getRoomId 获取房间 ID
func (l *Live) getRoomId() (string, error) {
	paths := strings.Split(l.Url.Path, "/")
	if len(paths) < 2 {
		return "", live.ErrRoomUrlIncorrect
	}
	roomid := paths[2]

	return roomid, nil
}

// getRoomInfo 获取房间信息
func (l *Live) getRoomInfo() ([]byte, error) {
	roomid, err := l.getRoomId()
	if err != nil {
		return nil, err
	}
	resp, err := requests.Get(roomInitUrl + roomid)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, live.ErrRoomNotExist
	}
	body, err := resp.Bytes()
	if err != nil || gjson.GetBytes(body, "code").Int() != 0 {
		return nil, live.ErrRoomNotExist
	}
	return body, nil
}

// GetInfo 获取直播信息
func (l *Live) GetInfo() (info *live.Info, err error) {
	body, err := l.getRoomInfo()
	if err != nil {
		return nil, live.ErrRoomNotExist
	}
	info = &live.Info{
		Live:      l,
		HostName:  gjson.GetBytes(body, "info.room.creator_username").String(),
		RoomName:  gjson.GetBytes(body, "info.room.name").String(),
		Status:    gjson.GetBytes(body, "info.room.status.open").Int() == 1,
		AudioOnly: true,
	}

	return info, nil
}

// GetStreamUrls 获取直播流 URL
func (l *Live) GetStreamUrls() (us []*url.URL, err error) {
	body, err := l.getRoomInfo()
	if err != nil {
		return nil, live.ErrRoomNotExist
	}
	return utils.GenUrls(gjson.GetBytes(body, "info.room.channel.flv_pull_url").String())
}

// GetPlatformCNName 获取平台的中文名称
func (l *Live) GetPlatformCNName() string {
	return cnName
}
