package huajiao

import (
	"net/http"
	"net/url"

	"github.com/tidwall/gjson"
	"github.com/yuhaohwang/requests"

	"github.com/yuhaohwang/bililive-go/src/live"
	"github.com/yuhaohwang/bililive-go/src/live/internal"
	"github.com/yuhaohwang/bililive-go/src/pkg/utils"
)

const (
	domain = "www.huajiao.com"
	cnName = "花椒"

	apiUserInfo  = "https://webh.huajiao.com/User/getUserInfo"
	apiUserFeeds = "https://webh.huajiao.com/User/getUserFeeds"
	apiStream    = "https://live.huajiao.com/live/substream"
)

// init 函数用于在程序启动时注册花椒直播平台的 builder
func init() {
	live.Register(domain, new(builder))
}

// builder 结构体实现 live.Builder 接口，用于构建花椒直播平台的直播实例
type builder struct{}

// Build 方法根据传入的 URL 构建花椒直播实例
func (b *builder) Build(url *url.URL, opt ...live.Option) (live.Live, error) {
	return &Live{
		BaseLive: internal.NewBaseLive(url, opt...),
	}, nil
}

// Live 结构体表示一个花椒直播实例
type Live struct {
	uid string
	internal.BaseLive
}

// getUid 方法获取花椒直播用户的 UID
func (l *Live) getUid() (string, error) {
	if l.uid != "" {
		return l.uid, nil
	}

	var uid string
	if uid = utils.Match1(`https?:\/\/www.huajiao.com\/user\/(\d+)`, l.GetRawUrl()); uid != "" {
		// nothing to do
	} else if liveId := utils.Match1(`https?:\/\/www.huajiao.com\/l\/(\d+)`, l.GetRawUrl()); liveId != "" {
		resp, err := requests.Get(l.GetRawUrl(), live.CommonUserAgent)
		if err != nil {
			return "", err
		}
		if resp.StatusCode != http.StatusOK {
			return "", live.ErrRoomNotExist
		}
		body, err := resp.Text()
		if err != nil {
			return "", err
		}
		uid = utils.Match1(`<span class="js-author-id">(\d+)</span>`, body)
		// if uid == "" {
		// 	TODO: error log
		// }
	}

	if uid != "" && uid != "0" {
		l.uid = uid
		return l.uid, nil
	} else {
		return "", live.ErrRoomUrlIncorrect
	}
}

// getNickname 方法根据 UID 获取花椒直播用户的昵称
func (l *Live) getNickname(uid string) (string, error) {
	resp, err := requests.Get(apiUserInfo, live.CommonUserAgent, requests.Query("fmt", "json"), requests.Query("uid", uid))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", live.ErrRoomNotExist
	}
	body, err := resp.Bytes()
	if err != nil {
		return "", err
	}
	if errno := gjson.GetBytes(body, "errno").Int(); errno != 0 {
		return "", live.ErrRoomNotExist
	}
	return gjson.GetBytes(body, "data.nickname").String(), nil
}

// getLiveFeeds 方法根据 UID 获取花椒直播用户的直播信息
func (l *Live) getLiveFeeds(uid string) ([]gjson.Result, error) {
	resp, err := requests.Get(apiUserFeeds, live.CommonUserAgent, requests.Query("fmt", "json"), requests.Query("uid", uid))
	if err != nil {
		return nil, err
	}
	feedsData, err := resp.Bytes()
	if err != nil {
		return nil, err
	}
	return gjson.GetBytes(feedsData, "data.feeds.#(type==1)#").Array(), nil
}

// GetInfo 方法获取花椒直播房间的信息，包括主播名称、房间名称和直播状态
func (l *Live) GetInfo() (info *live.Info, err error) {
	uid, err := l.getUid()
	if err != nil {
		return nil, err
	}

	info = &live.Info{
		Live:         l,
		HostName:     "",
		RoomName:     "",
		Status:       false,
		CustomLiveId: "huajiao/" + uid,
	}
	nickname, err := l.getNickname(uid)
	if err != nil {
		return nil, err
	}
	info.HostName = nickname

	feeds, err := l.getLiveFeeds(uid)
	if err != nil {
		return nil, err
	}
	if len(feeds) == 0 {
		return info, nil
	}

	info.RoomName = feeds[0].Get("feed.title").String()
	info.Status = true
	return info, nil
}

// GetStreamUrls 方法获取花椒直播房间的流媒体 URL
func (l *Live) GetStreamUrls() (us []*url.URL, err error) {
	uid, err := l.getUid()
	if err != nil {
		return nil, err
	}
	feeds, err := l.getLiveFeeds(uid)
	if err != nil {
		return nil, err
	}
	if len(feeds) == 0 {
		return nil, live.ErrInternalError
	}
	var (
		sn     = feeds[0].Get("feed.sn").String()
		liveID = feeds[0].Get("feed.relateid").String()
	)
	resp, err := requests.Get(apiStream, live.CommonUserAgent, requests.Queries(map[string]string{
		"sn":     sn,
		"uid":    uid,
		"liveid": liveID,
		"encode": "h264",
	}))
	if err != nil {
		return nil, err
	}
	body, err := resp.Bytes()
	if err != nil {
		return nil, err
	}
	if errno := gjson.GetBytes(body, "errno").Int(); errno != 0 {
		return nil, live.ErrInternalError
	}

	return utils.GenUrls(gjson.GetBytes(body, "data.main").String())
}

// GetPlatformCNName 方法获取花椒直播平台的中文名称
func (l *Live) GetPlatformCNName() string {
	return cnName
}
