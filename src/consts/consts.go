package consts

import (
	"fmt"
	"os"
	"runtime"
)

// AppName 是应用程序的名称常量。
const AppName = "BiliLive-go"

// Info 存储应用程序的信息。
type Info struct {
	AppName    string `json:"app_name"`    // 应用程序名称
	AppVersion string `json:"app_version"` // 应用程序版本
	BuildTime  string `json:"build_time"`  // 构建时间
	GitHash    string `json:"git_hash"`    // Git哈希值
	Pid        int    `json:"pid"`         // 进程ID
	Platform   string `json:"platform"`    // 平台信息
	GoVersion  string `json:"go_version"`  // Go版本
}

var (
	// BuildTime 存储应用程序的构建时间。
	BuildTime string
	// AppVersion 存储应用程序的版本。
	AppVersion string
	// GitHash 存储应用程序的Git哈希值。
	GitHash string
	// AppInfo 包含应用程序的信息。
	AppInfo = Info{
		AppName:    AppName,
		AppVersion: AppVersion,
		BuildTime:  BuildTime,
		GitHash:    GitHash,
		Pid:        os.Getpid(),
		Platform:   fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		GoVersion:  runtime.Version(),
	}
)
