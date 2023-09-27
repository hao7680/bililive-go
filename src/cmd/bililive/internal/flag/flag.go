package flag

import (
	"os"
	"time"

	"github.com/alecthomas/kingpin"

	"github.com/yuhaohwang/bililive-go/src/configs"
	"github.com/yuhaohwang/bililive-go/src/consts"
	"github.com/yuhaohwang/bililive-go/src/pkg/utils"
)

// 创建一个新的应用程序实例
var (
	app = kingpin.New(consts.AppName, "一个命令行直播流保存工具。").Version(consts.AppVersion)

	// 调试模式标志
	Debug = app.Flag("debug", "启用调试模式。").Default("false").Bool()

	// 查询直播状态的时间间隔
	Interval = app.Flag("interval", "查询直播状态的时间间隔").Default("20").Short('t').Int()

	// 输出文件路径
	Output = app.Flag("output", "输出文件路径。").Short('o').Default("./").String()

	// FFMPEG路径
	FfmpegPath = app.Flag("ffmpeg-path", "FFMPEG路径（默认：从环境变量中查找FFMPEG）").Default("").String()

	// 直播房间URL列表
	Input = app.Flag("input", "直播房间URL列表").Short('i').Strings()

	// 配置文件路径
	Conf = app.Flag("config", "配置文件路径。").Short('c').String()

	// 启用RPC服务器标志
	RPC = app.Flag("enable-rpc", "启用RPC服务器。").Default("false").Bool()

	// RPC服务器绑定地址
	RPCBind = app.Flag("rpc-bind", "RPC服务器绑定地址").Default(":8080").String()

	// 使用本地FLV解析器标志
	NativeFlvParser = app.Flag("native-flv-parser", "使用本地FLV解析器").Default("false").Bool()

	// 输出文件名模板
	OutputFileTmpl = app.Flag("output-file-tmpl", "输出文件名模板").Default("").String()

	// 视频分割策略
	SplitStrategies = app.Flag("split-strategies", "视频分割策略，支持\"on_room_name_changed\", \"max_duration:(duration)\"").Strings()
)

func init() {
	// 解析命令行参数
	kingpin.MustParse(app.Parse(os.Args[1:]))
}

// GenConfigFromFlags 通过解析命令行参数生成配置信息。
func GenConfigFromFlags() *configs.Config {
	cfg := configs.NewConfig()
	cfg.RPC = configs.RPC{
		Enable: *RPC,
		Bind:   *RPCBind,
	}
	cfg.Debug = *Debug
	cfg.Interval = *Interval
	cfg.OutPutPath = *Output
	cfg.FfmpegPath = *FfmpegPath
	cfg.OutputTmpl = *OutputFileTmpl
	cfg.LiveRooms = configs.NewLiveRoomsWithStrings(*Input)
	cfg.Feature = configs.Feature{
		UseNativeFlvParser: *NativeFlvParser,
	}

	if SplitStrategies != nil && len(*SplitStrategies) > 0 {
		for _, s := range *SplitStrategies {
			// TODO: 不要硬编码
			if s == "on_room_name_changed" {
				cfg.VideoSplitStrategies.OnRoomNameChanged = true
			}
			if durStr := utils.Match1(`max_duration:(.*)`, s); durStr != "" {
				dur, err := time.ParseDuration(durStr)
				if err == nil {
					cfg.VideoSplitStrategies.MaxDuration = dur
				}
			}
		}
	}
	return cfg
}
