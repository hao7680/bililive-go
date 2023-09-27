package kuaishou

import (
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
	domain = "live.kuaishou.com"
	cnName = "快手"

	regRenderData = `window\.__INITIAL_STATE__ *= *(.*?) *; *\(`
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

// getData 从页面获取数据
func (l *Live) getData() (*gjson.Result, error) {
	// 从选项中获取 Cookies
	cookies := l.Options.Cookies.Cookies(l.Url)
	cookieKVs := make(map[string]string)
	for _, item := range cookies {
		cookieKVs[item.Name] = item.Value
	}
	// 发送 GET 请求获取页面内容
	resp, err := requests.Get(l.Url.String(), live.CommonUserAgent, requests.Cookies(cookieKVs))
	if err != nil {
		return nil, err
	}
	switch code := resp.StatusCode; code {
	case http.StatusOK:
	case http.StatusNotFound:
		return nil, live.ErrRoomNotExist
	default:
		return nil, fmt.Errorf("failed to get page, code: %v, %w", code, live.ErrInternalError)
	}

	// 解析页面内容
	body, err := resp.Text()
	if err != nil {
		return nil, err
	}
	// 从页面正文中提取渲染数据
	rawData := utils.Match1(regRenderData, body)
	if rawData == "" {
		return nil, fmt.Errorf("failed to get RENDER_DATA from page, %w", live.ErrInternalError)
	}
	// 对渲染数据进行 URL 解码
	unescapedRawData, err := url.QueryUnescape(rawData)
	if err != nil {
		return nil, err
	}
	// 解析 JSON 数据
	result := gjson.Parse(unescapedRawData)
	return &result, nil
}

// GetInfo 获取直播信息
func (l *Live) GetInfo() (info *live.Info, err error) {
	data, err := l.getData()
	if err != nil {
		return nil, err
	}
	info = &live.Info{
		Live:     l,
		HostName: data.Get("liveroom.author.name").String(),
		RoomName: data.Get("liveroom.liveStream.caption").String(),
		Status:   data.Get("liveroom.isLiving").Bool(),
	}
	return
}

// GetStreamUrls 获取直播流 URL
func (l *Live) GetStreamUrls() (us []*url.URL, err error) {
	data, err := l.getData()
	if err != nil {
		return nil, err
	}
	var urls []string

	addr := ""
	addr = "liveroom.liveStream.playUrls.0.adaptationSet.representation.0.url"

	// 由于更高清晰度需要cookie，暂时无法传，先注释
	//maxQuality := len(data.Get("liveroom.liveStream.playUrls.0.adaptationSet.representation").Array()) - 1
	//if l.Options.Quality != 0 && maxQuality >= l.Options.Quality {
	//	addr = "liveroom.liveStream.playUrls.0.adaptationSet.representation." + strconv.Itoa(l.Options.Quality) + ".url"
	//} else if l.Options.Quality != 0 {
	//	addr = "liveroom.liveStream.playUrls.0.adaptationSet.representation." + strconv.Itoa(maxQuality) + ".url"
	//} else {
	//	addr = "liveroom.liveStream.playUrls.0.adaptationSet.representation.0.url"
	//}

	data.Get(addr).ForEach(func(key, value gjson.Result) bool {
		urls = append(urls, value.String())
		return true
	})
	return utils.GenUrls(urls...)
}

// GetPlatformCNName 获取平台的中文名称
func (l *Live) GetPlatformCNName() string {
	return cnName
}
