package internal

import (
	"fmt"
	"net/url"
	"time"

	"github.com/yuhaohwang/bililive-go/src/live"
	"github.com/yuhaohwang/bililive-go/src/pkg/utils"
)

// BaseLive 结构体包含了直播平台的基本信息和选项
type BaseLive struct {
	Url           *url.URL      // 直播平台的 URL
	LastStartTime time.Time     // 上次直播开始时间
	LiveId        live.ID       // 直播唯一标识符
	Options       *live.Options // 直播选项
}

// genLiveId 根据 URL 生成直播唯一标识符
func genLiveId(url *url.URL) live.ID {
	return genLiveIdByString(fmt.Sprintf("%s%s", url.Host, url.Path))
}

// genLiveIdByString 根据字符串生成直播唯一标识符
func genLiveIdByString(value string) live.ID {
	return live.ID(utils.GetMd5String([]byte(value)))
}

// NewBaseLive 创建一个 BaseLive 实例
func NewBaseLive(url *url.URL, opt ...live.Option) BaseLive {
	return BaseLive{
		Url:     url,
		LiveId:  genLiveId(url),
		Options: live.MustNewOptions(opt...),
	}
}

// SetLiveIdByString 设置直播唯一标识符
func (a *BaseLive) SetLiveIdByString(value string) {
	a.LiveId = genLiveIdByString(value)
}

// GetLiveId 获取直播唯一标识符
func (a *BaseLive) GetLiveId() live.ID {
	return a.LiveId
}

// GetRawUrl 获取原始的直播平台 URL
func (a *BaseLive) GetRawUrl() string {
	return a.Url.String()
}

// GetLastStartTime 获取上次直播开始时间
func (a *BaseLive) GetLastStartTime() time.Time {
	return a.LastStartTime
}

// SetLastStartTime 设置上次直播开始时间
func (a *BaseLive) SetLastStartTime(time time.Time) {
	a.LastStartTime = time
}
