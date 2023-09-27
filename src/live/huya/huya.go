package huya

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/yuhaohwang/requests"

	"github.com/yuhaohwang/bililive-go/src/live"
	"github.com/yuhaohwang/bililive-go/src/live/internal"
	"github.com/yuhaohwang/bililive-go/src/pkg/utils"
)

const (
	domain = "www.huya.com"
	cnName = "虎牙"
)

// init 函数用于在程序启动时注册虎牙直播平台的 builder
func init() {
	live.Register(domain, new(builder))
}

// builder 结构体实现 live.Builder 接口，用于构建虎牙直播平台的直播实例
type builder struct{}

// Build 方法根据传入的 URL 构建虎牙直播实例
func (b *builder) Build(url *url.URL, opt ...live.Option) (live.Live, error) {
	return &Live{
		BaseLive: internal.NewBaseLive(url, opt...),
	}, nil
}

// Live 结构体表示一个虎牙直播实例
type Live struct {
	internal.BaseLive
}

// GetInfo 方法获取虎牙直播房间的信息，包括主播名称、房间名称和直播状态
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

	if res := utils.Match1("哎呀，虎牙君找不到这个主播，要不搜索看看？", body); res != "" {
		return nil, live.ErrRoomNotExist
	}

	if strings.Contains(body, "该主播涉嫌违规，正在整改中") {
		return &live.Info{
			Live:     l,
			HostName: "该主播涉嫌违规，正在整改中",
			RoomName: "该主播涉嫌违规，正在整改中",
			Status:   false,
		}, nil
	}

	var (
		strFilter = utils.NewStringFilterChain(utils.ParseUnicode, utils.UnescapeHTMLEntity)
		hostName  = strFilter.Do(utils.Match1(`"nick":"([^"]*)"`, body))
		roomName  = strFilter.Do(utils.Match1(`"introduction":"([^"]*)"`, body))
		status    = strFilter.Do(utils.Match1(`"isOn":([^,]*),`, body))
	)

	if hostName == "" || roomName == "" || status == "" {
		return nil, live.ErrInternalError
	}

	info = &live.Info{
		Live:     l,
		HostName: hostName,
		RoomName: roomName,
		Status:   status == "true",
	}
	return info, nil
}

// GetMD5Hash 方法计算字符串的 MD5 哈希值
func GetMD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

// parseAntiCode 方法解析反作弊参数并生成反作弊校验码
func parseAntiCode(anticode string, uid int64, streamName string) (string, error) {
	qr, err := url.ParseQuery(anticode)
	if err != nil {
		return "", err
	}
	qr.Set("ver", "1")
	qr.Set("sv", "2110211124")
	qr.Set("seqid", strconv.FormatInt(time.Now().Unix()*1000+uid, 10))
	qr.Set("uid", strconv.FormatInt(uid, 10))
	uuid, _ := uuid.NewV4()
	qr.Set("uuid", uuid.String())
	ss := GetMD5Hash(fmt.Sprintf("%s|%s|%s", qr.Get("seqid"), qr.Get("ctype"), qr.Get("t")))

	decodeString, _ := base64.StdEncoding.DecodeString(qr.Get("fm"))
	fm := string(decodeString)
	fm = strings.ReplaceAll(fm, "$0", qr.Get("uid"))
	fm = strings.ReplaceAll(fm, "$1", streamName)
	fm = strings.ReplaceAll(fm, "$2", ss)
	fm = strings.ReplaceAll(fm, "$3", qr.Get("wsTime"))

	qr.Set("wsSecret", GetMD5Hash(fm))
	return qr.Encode(), nil
}

// GetStreamUrls 方法获取虎牙直播房间的流媒体 URL
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

	// Decode stream part.
	streamStr := utils.Match1(`(?m)stream: (.*?)$`, body)

	var (
		sStreamName  = utils.Match1(`"sStreamName":"([^"]*)"`, streamStr)
		sFlvUrl      = strings.ReplaceAll(utils.Match1(`"sFlvUrl":"([^"]*)"`, streamStr), `\/`, `/`)
		sFlvAntiCode = utils.Match1(`"sFlvAntiCode":"([^"]*)"`, streamStr)
		uid          = (time.Now().Unix()%1e7*1e6 + int64(1e3*rand.Float64())) % 4294967295
	)
	query, err := parseAntiCode(sFlvAntiCode, uid, sStreamName)
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(fmt.Sprintf("%s/%s.flv?%s", sFlvUrl, sStreamName, query))
	if err != nil {
		return nil, err
	}
	// value := url.Values{}
	// value.Add("line", iLineIndex)
	// value.Add("p2p", "0")
	// value.Add("type", "web")
	// value.Add("ver", "1805071653")
	// value.Add("uid", fmt.Sprintf("%d", uid))
	// u.RawQuery = fmt.Sprintf("%s&%s", value.Encode(), utils.UnescapeHTMLEntity(sFlvAntiCode))
	return []*url.URL{u}, nil
}

// GetPlatformCNName 方法获取虎牙直播平台的中文名称
func (l *Live) GetPlatformCNName() string {
	return cnName
}
