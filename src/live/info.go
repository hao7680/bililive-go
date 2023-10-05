package live

import (
	"encoding/json"
)

// Info 结构体用于存储直播信息，包括主播名、房间名、状态等。
type Info struct {
	Live                          Live
	HostName, RoomName            string
	RtmpUrl                       string
	Listen                        bool
	Record                        bool
	Push                          bool
	Status                        bool // 表示是否正在直播，可能最好重命名为 IsLiving
	Listening, Recording, Pushing bool
	Initializing                  bool
	CustomLiveId                  string
	AudioOnly                     bool
}

// MarshalJSON 方法用于将 Info 结构体序列化为 JSON 格式。
func (i *Info) MarshalJSON() ([]byte, error) {
	t := struct {
		Id                ID     `json:"id"`                             // 直播唯一标识
		LiveUrl           string `json:"live_url"`                       // 直播原始 URL
		PlatformCNName    string `json:"platform_cn_name"`               // 平台中文名称
		HostName          string `json:"host_name"`                      // 主播名
		RoomName          string `json:"room_name"`                      // 房间名
		Status            bool   `json:"status"`                         // 是否正在直播
		Listening         bool   `json:"listening"`                      // 是否正在监听
		Recording         bool   `json:"recording"`                      // 是否正在录制
		Pushing           bool   `json:"pushing"`                        // 是否正在转推
		Initializing      bool   `json:"initializing"`                   // 是否正在初始化
		LastStartTime     string `json:"last_start_time,omitempty"`      // 上次开始时间的字符串表示形式
		LastStartTimeUnix int64  `json:"last_start_time_unix,omitempty"` // 上次开始时间的 UNIX 时间戳
		AudioOnly         bool   `json:"audio_only"`                     // 是否仅音频直播
		RtmpUrl           string `json:"rtmp_url"`                       // 直播转推 URL
		Listen            bool   `json:"listen"`                         // 是否开启直播监听
		Record            bool   `json:"record"`                         // 是否开启直播录制
		Push              bool   `json:"push"`                           // 是否开启直播转推
	}{
		Id:             i.Live.GetLiveId(),
		LiveUrl:        i.Live.GetRawUrl(),
		PlatformCNName: i.Live.GetPlatformCNName(),
		HostName:       i.HostName,
		RoomName:       i.RoomName,
		Status:         i.Status,
		Listening:      i.Listening,
		Recording:      i.Recording,
		Pushing:        i.Pushing,
		Initializing:   i.Initializing,
		AudioOnly:      i.AudioOnly,
		RtmpUrl:        i.RtmpUrl,
		Listen:         i.Listen,
		Record:         i.Record,
		Push:           i.Push,
	}
	if !i.Live.GetLastStartTime().IsZero() {
		t.LastStartTime = i.Live.GetLastStartTime().Format("2006-01-02 15:04:05")
		t.LastStartTimeUnix = i.Live.GetLastStartTime().Unix()
	}
	return json.Marshal(t)
}
