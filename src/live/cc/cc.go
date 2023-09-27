package cc

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/tidwall/gjson"
	"github.com/yuhaohwang/requests"

	"github.com/yuhaohwang/bililive-go/src/live"
	"github.com/yuhaohwang/bililive-go/src/live/internal"
	"github.com/yuhaohwang/bililive-go/src/pkg/utils"
)

const (
	domain = "cc.163.com"
	cnName = "CC直播"

	apiUrl = "http://cgi.v.cc.163.com/video_play_url/"
	dataRe = `<script id="__NEXT_DATA__" type="application/json" crossorigin="anonymous">(.*?)</script>`
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

// getData 获取CC直播数据
func (l *Live) getData() (*gjson.Result, error) {
	resp, err := requests.Get(l.Url.String(), live.CommonUserAgent)
	if err != nil {
		return nil, err
	}
	body, err := resp.Bytes()
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, live.ErrRoomNotExist
	}
	data := utils.UnescapeHTMLEntity(utils.Match1(dataRe, string(body)))
	if data == "" {
		return nil, errors.New("数据为空")
	}
	result := gjson.Parse(data)
	return &result, nil
}

// getCcID 获取CC直播的ccid
func (l *Live) getCcID() (string, error) {
	data, err := l.getData()
	if err != nil {
		return "", err
	}
	return data.Get("props.pageProps.roomInfoInitData.micfirst.ccid").String(), nil
}

// GetInfo 获取直播信息
func (l *Live) GetInfo() (info *live.Info, err error) {
	data, err := l.getData()
	if err != nil {
		return nil, err
	}
	var (
		hostName = data.Get("props.pageProps.roomInfoInitData.micfirst.nickname").String()
		roomName = data.Get("props.pageProps.roomInfoInitData.live.title").String()
	)

	if hostName == "" || roomName == "" {
		return nil, errors.New("解析主播姓名和房间名称失败")
	}

	info = &live.Info{
		Live:     l,
		HostName: hostName,
		RoomName: roomName,
		Status:   data.Get("props.pageProps.roomInfoInitData.live.swf").String() != "",
	}
	return info, nil
}

// GetStreamUrls 获取直播流媒体URL
func (l *Live) GetStreamUrls() (us []*url.URL, err error) {
	ccid, err := l.getCcID()
	if err != nil {
		return nil, err
	}
	resp, err := requests.Get(fmt.Sprintf("%s%s", apiUrl, ccid), live.CommonUserAgent)
	if err != nil {
		return nil, err
	}
	body, err := resp.Bytes()
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, live.ErrRoomNotExist
	}
	return utils.GenUrls(
		gjson.GetBytes(body, "videourl").String(),
		gjson.GetBytes(body, "bakvideourl").String(),
	)
}

// GetPlatformCNName 获取直播平台的中文名称
func (l *Live) GetPlatformCNName() string {
	return cnName
}
