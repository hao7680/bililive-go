package openrec

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/yuhaohwang/requests"

	"github.com/yuhaohwang/bililive-go/src/live"
	"github.com/yuhaohwang/bililive-go/src/live/internal"
	"github.com/yuhaohwang/bililive-go/src/pkg/utils"
)

const (
	domain = "www.openrec.tv" // 定义域名
	cnName = "openrec"        // 定义平台中文名称
)

// Live 结构体表示 OpenRec 平台的直播信息。
type Live struct {
	internal.BaseLive
}

func init() {
	live.Register(domain, new(builder))
}

// builder 结构体用于构建 OpenRec 平台的直播。
type builder struct{}

// Build 方法实现了构建 OpenRec 平台直播的逻辑。
func (b *builder) Build(url *url.URL, opt ...live.Option) (live.Live, error) {
	return &Live{
		BaseLive: internal.NewBaseLive(url, opt...),
	}, nil
}

// GetInfo 方法用于获取 OpenRec 平台的直播信息。
func (l *Live) GetInfo() (info *live.Info, err error) {
	resp, err := requests.Get(l.Url.String(), live.CommonUserAgent)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, live.ErrRoomNotExist
	}
	body, err := resp.Text()
	if err != nil {
		return nil, err
	}
	var (
		roomName = utils.ParseString(
			utils.Match1(`"title":"([^:]*)",`, body),
			utils.StringFilterFunc(strings.TrimSpace),
			utils.UnescapeHTMLEntity,
		)
		hostName = utils.ParseString(
			utils.Match1(`"name":"([^:]*)",`, body),
			utils.ParseUnicode,
			utils.UnescapeHTMLEntity,
		)
		status = utils.Match1(`"onairStatus":(\d),`, body)
	)
	if roomName == "" || hostName == "" || status == "" {
		return nil, live.ErrInternalError
	}
	info = &live.Info{
		Live:     l,
		RoomName: roomName,
		HostName: hostName,
		Status:   status == "1",
	}
	return info, nil
}

// GetStreamUrls 方法用于获取 OpenRec 平台的直播流媒体 URL。
func (l *Live) GetStreamUrls() (us []*url.URL, err error) {
	resp, err := requests.Get(l.Url.String(), live.CommonUserAgent)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, live.ErrRoomNotExist
	}
	body, err := resp.Text()
	if err != nil {
		return nil, err
	}
	return utils.GenUrls(utils.Match1(`{"url":"(\S*m3u8)",`, body))
}

// GetPlatformCNName 方法返回 OpenRec 平台的中文名称。
func (l *Live) GetPlatformCNName() string {
	return cnName
}
