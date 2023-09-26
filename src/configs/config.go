package configs

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"time"

	"github.com/hr3lxphr6j/bililive-go/src/live"
	"gopkg.in/yaml.v2"
)

// RPC包含RPC相关信息。
type RPC struct {
	Enable bool   `yaml:"enable"` // 是否启用RPC
	Bind   string `yaml:"bind"`   // 绑定的地址和端口
}

var defaultRPC = RPC{
	Enable: true,
	Bind:   "127.0.0.1:8080",
}

// verify 验证RPC设置的有效性。
func (r *RPC) verify() error {
	if r == nil {
		return nil
	}
	if !r.Enable {
		return nil
	}
	if _, err := net.ResolveTCPAddr("tcp", r.Bind); err != nil {
		return err
	}
	return nil
}

// Feature包含特性相关信息。
type Feature struct {
	UseNativeFlvParser         bool `yaml:"use_native_flv_parser"`         // 是否使用本地FLV解析器
	RemoveSymbolOtherCharacter bool `yaml:"remove_symbol_other_character"` // 是否删除特殊符号
}

// VideoSplitStrategies包含视频分割策略信息。
type VideoSplitStrategies struct {
	OnRoomNameChanged bool          `yaml:"on_room_name_changed"` // 当房间名称更改时是否分割视频
	MaxDuration       time.Duration `yaml:"max_duration"`         // 最大分割视频时长
	MaxFileSize       int           `yaml:"max_file_size"`        // 最大分割文件大小
}

// OnRecordFinished包含录制完成后的操作信息。
type OnRecordFinished struct {
	ConvertToMp4          bool   `yaml:"convert_to_mp4"`           // 是否转换为MP4格式
	DeleteFlvAfterConvert bool   `yaml:"delete_flv_after_convert"` // 转换后是否删除FLV文件
	CustomCommandline     string `yaml:"custom_commandline"`       // 自定义命令行操作
}

// Log包含日志相关信息。
type Log struct {
	OutPutFolder string `yaml:"out_put_folder"` // 输出日志文件夹
	SaveLastLog  bool   `yaml:"save_last_log"`  // 是否保存最后一条日志
	SaveEveryLog bool   `yaml:"save_every_log"` // 是否保存每一条日志
}

// Config包含所有配置信息。
type Config struct {
	File                 string               `yaml:"-"`                      // 配置文件路径
	RPC                  RPC                  `yaml:"rpc"`                    // RPC配置
	Debug                bool                 `yaml:"debug"`                  // 是否启用调试模式
	Interval             int                  `yaml:"interval"`               // 采集间隔
	OutPutPath           string               `yaml:"out_put_path"`           // 输出路径
	FfmpegPath           string               `yaml:"ffmpeg_path"`            // FFmpeg路径
	Log                  Log                  `yaml:"log"`                    // 日志配置
	Feature              Feature              `yaml:"feature"`                // 特性配置
	LiveRooms            []LiveRoom           `yaml:"live_rooms"`             // 直播房间配置
	OutputTmpl           string               `yaml:"out_put_tmpl"`           // 输出模板
	VideoSplitStrategies VideoSplitStrategies `yaml:"video_split_strategies"` // 视频分割策略
	Cookies              map[string]string    `yaml:"cookies"`                // Cookies配置
	OnRecordFinished     OnRecordFinished     `yaml:"on_record_finished"`     // 录制完成后的操作配置
	TimeoutInUs          int                  `yaml:"timeout_in_us"`          // 超时时间（微秒）

	liveRoomIndexCache map[string]int
}

// LiveRoom包含直播房间信息。
type LiveRoom struct {
	Url         string  `yaml:"url"`          // 直播房间URL
	IsListening bool    `yaml:"is_listening"` // 是否监听直播
	LiveId      live.ID `yaml:"-"`            // 直播ID
	Quality     int     `yaml:"quality"`      // 视频质量
}

// liveRoomAlias用于在配置中同时支持字符串和LiveRoom格式。
type liveRoomAlias LiveRoom

// UnmarshalYAML 实现了LiveRoom的自定义反序列化。
func (l *LiveRoom) UnmarshalYAML(unmarshal func(interface{}) error) error {
	liveRoomAlias := liveRoomAlias{
		IsListening: true,
	}
	if err := unmarshal(&liveRoomAlias); err != nil {
		var url string
		if err = unmarshal(&url); err != nil {
			return err
		}
		liveRoomAlias.Url = url
	}
	*l = LiveRoom(liveRoomAlias)

	return nil
}

// NewLiveRoomsWithStrings 从字符串数组创建LiveRoom列表。
func NewLiveRoomsWithStrings(strings []string) []LiveRoom {
	if len(strings) == 0 {
		return make([]LiveRoom, 0, 4)
	}
	liveRooms := make([]LiveRoom, len(strings))
	for index, url := range strings {
		liveRooms[index].Url = url
		liveRooms[index].IsListening = true
		liveRooms[index].Quality = 0
	}
	return liveRooms
}

var defaultConfig = Config{
	RPC:        defaultRPC,
	Debug:      false,
	Interval:   30,
	OutPutPath: "./",
	FfmpegPath: "",
	Log: Log{
		OutPutFolder: "./",
		SaveLastLog:  true,
		SaveEveryLog: false,
	},
	Feature: Feature{
		UseNativeFlvParser:         false,
		RemoveSymbolOtherCharacter: false,
	},
	LiveRooms:          []LiveRoom{},
	File:               "",
	liveRoomIndexCache: map[string]int{},
	VideoSplitStrategies: VideoSplitStrategies{
		OnRoomNameChanged: false,
	},
	OnRecordFinished: OnRecordFinished{
		ConvertToMp4:          false,
		DeleteFlvAfterConvert: false,
	},
	TimeoutInUs: 60000000,
}

// NewConfig 创建新的Config对象。
func NewConfig() *Config {
	config := defaultConfig
	config.liveRoomIndexCache = map[string]int{}
	return &config
}

// Verify 验证配置的有效性。
func (c *Config) Verify() error {
	if c == nil {
		return fmt.Errorf("配置为空")
	}
	if err := c.RPC.verify(); err != nil {
		return err
	}
	if c.Interval <= 0 {
		return fmt.Errorf("采集间隔不能小于等于0")
	}
	if _, err := os.Stat(c.OutPutPath); err != nil {
		return fmt.Errorf(`输出路径 "%s" 不存在`, c.OutPutPath)
	}
	if maxDur := c.VideoSplitStrategies.MaxDuration; maxDur > 0 && maxDur < time.Minute {
		return fmt.Errorf("max_duration的最小值为一分钟")
	}
	if !c.RPC.Enable && len(c.LiveRooms) == 0 {
		return fmt.Errorf("RPC未启用，且未设置直播房间，程序没有可执行操作")
	}
	return nil
}

// RefreshLiveRoomIndexCache 刷新直播房间索引缓存。
func (c *Config) RefreshLiveRoomIndexCache() {
	for index, room := range c.LiveRooms {
		c.liveRoomIndexCache[room.Url] = index
	}
}

// RemoveLiveRoomByUrl 通过URL移除直播房间。
func (c *Config) RemoveLiveRoomByUrl(url string) error {
	c.RefreshLiveRoomIndexCache()
	if index, ok := c.liveRoomIndexCache[url]; ok {
		if index >= 0 && index < len(c.LiveRooms) && c.LiveRooms[index].Url == url {
			c.LiveRooms = append(c.LiveRooms[:index], c.LiveRooms[index+1:]...)
			delete(c.liveRoomIndexCache, url)
			return nil
		}
	}
	return errors.New("移除房间失败：" + url)
}

// GetLiveRoomByUrl 通过URL获取直播房间。
func (c *Config) GetLiveRoomByUrl(url string) (*LiveRoom, error) {
	room, err := c.getLiveRoomByUrlImpl(url)
	if err != nil {
		c.RefreshLiveRoomIndexCache()
		if room, err = c.getLiveRoomByUrlImpl(url); err != nil {
			return nil, err
		}
	}
	return room, nil
}

// getLiveRoomByUrlImpl 通过URL获取直播房间的实现方法。
func (c Config) getLiveRoomByUrlImpl(url string) (*LiveRoom, error) {
	if index, ok := c.liveRoomIndexCache[url]; ok {
		if index >= 0 && index < len(c.LiveRooms) && c.LiveRooms[index].Url == url {
			return &c.LiveRooms[index], nil
		}
	}
	return nil, errors.New("房间 " + url + " 不存在")
}

// NewConfigWithBytes 使用字节数组创建Config对象。
func NewConfigWithBytes(b []byte) (*Config, error) {
	config := defaultConfig
	if err := yaml.Unmarshal(b, &config); err != nil {
		return nil, err
	}
	config.RefreshLiveRoomIndexCache()
	return &config, nil
}

// NewConfigWithFile 使用文件创建Config对象。
func NewConfigWithFile(file string) (*Config, error) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("无法打开文件：%s", file)
	}
	config, err := NewConfigWithBytes(b)
	if err != nil {
		return nil, err
	}
	config.File = file
	return config, nil
}

// Marshal 将配置对象序列化为字节数组并保存到文件。
func (c *Config) Marshal() error {
	if c.File == "" {
		return errors.New("未设置配置文件路径")
	}
	b, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(c.File, b, os.ModeAppend)
}

// GetFilePath 获取配置文件路径。
func (c Config) GetFilePath() (string, error) {
	if c.File == "" {
		return "", errors.New("未设置配置文件路径")
	}
	return c.File, nil
}
