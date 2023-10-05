package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/bluele/gcache"

	_ "github.com/yuhaohwang/bililive-go/src/cmd/bililive/internal"
	"github.com/yuhaohwang/bililive-go/src/cmd/bililive/internal/flag"
	"github.com/yuhaohwang/bililive-go/src/configs"
	"github.com/yuhaohwang/bililive-go/src/consts"
	"github.com/yuhaohwang/bililive-go/src/instance"
	"github.com/yuhaohwang/bililive-go/src/listeners"
	"github.com/yuhaohwang/bililive-go/src/live"
	"github.com/yuhaohwang/bililive-go/src/log"
	"github.com/yuhaohwang/bililive-go/src/metrics"
	"github.com/yuhaohwang/bililive-go/src/pkg/events"
	"github.com/yuhaohwang/bililive-go/src/pkg/utils"
	"github.com/yuhaohwang/bililive-go/src/pushers"
	"github.com/yuhaohwang/bililive-go/src/recorders"
	"github.com/yuhaohwang/bililive-go/src/servers"
)

// getConfig 函数用于获取程序的配置信息。
func getConfig() (*configs.Config, error) {
	var config *configs.Config

	// 检查命令行参数是否指定了配置文件。
	if *flag.Conf != "" {
		// 如果指定了配置文件，则尝试从文件加载配置。
		c, err := configs.NewConfigWithFile(*flag.Conf)
		if err != nil {
			return nil, err
		}
		config = c
	} else {
		// 如果没有指定配置文件，则从命令行标志生成配置。
		config = flag.GenConfigFromFlags()
	}

	// 如果配置中未启用RPC且没有定义任何直播房间，尝试从可执行文件旁边的config.yml文件加载配置。
	if !config.RPC.Enable && len(config.LiveRooms) == 0 {
		configBesidesExe, err := getConfigBesidesExecutable()
		if err == nil {
			return configBesidesExe, configBesidesExe.Verify()
		}
	}

	// 返回最终的配置，并验证配置的有效性。
	return config, config.Verify()
}

// getConfigBesidesExecutable 函数用于获取可执行文件旁边的配置信息。
func getConfigBesidesExecutable() (*configs.Config, error) {
	// 获取可执行文件的路径。
	exePath, err := os.Executable()
	if err != nil {
		return nil, err
	}

	// 构建配置文件的完整路径。
	configPath := filepath.Join(filepath.Dir(exePath), "config.yml")

	// 尝试从文件加载配置。
	config, err := configs.NewConfigWithFile(configPath)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func main() {
	// 获取配置信息
	config, err := getConfig()
	if err != nil {
		// 如果获取配置信息失败，打印错误信息到标准错误并退出程序。
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}

	// 创建一个新的程序实例（instance.Instance），并设置配置信息。
	inst := new(instance.Instance)
	inst.Config = config
	// TODO: 用哈希表替换gcache。
	// LRU似乎在这里不是必要的。
	inst.Cache = gcache.New(1024).LRU().Build()

	// 创建一个带有实例信息的上下文（context.Context）。
	ctx := context.WithValue(context.Background(), instance.Key, inst)

	// 创建日志记录器（logger）。
	logger := log.New(ctx)
	// 打印程序版本信息。
	logger.Infof("%s 版本: %s 启动链接", consts.AppName, consts.AppVersion)
	if config.File != "" {
		// 如果配置文件路径非空，则打印配置文件路径和忽略其他标志信息。
		logger.Debugf("配置路径: %s.", config.File)
		logger.Debugf("其他标志已被忽略.")
	} else {
		// 否则，打印未使用配置文件并打印使用的标志。
		logger.Debugf("未使用配置文件.")
		logger.Debugf("标志: %s 被使用.", os.Args)
	}
	// 打印程序信息。
	logger.Debugf("%+v", consts.AppInfo)
	logger.Debugf("%+v", inst.Config)

	// 检查是否存在FFmpeg二进制文件。
	if !utils.IsFFmpegExist(ctx) {
		logger.Fatalln("未找到FFmpeg二进制文件，请检查.")
	}

	// 创建事件分发器。
	events.NewDispatcher(ctx)

	// 初始化直播房间信息并添加到实例的Lives映射中。
	inst.Lives = make(map[live.ID]live.Live)
	for index := range inst.Config.LiveRooms {
		room := &inst.Config.LiveRooms[index]
		u, err := url.Parse(room.Url)
		if err != nil {
			logger.WithField("url", room).Error(err)
			continue
		}
		opts := make([]live.Option, 0)
		if v, ok := inst.Config.Cookies[u.Host]; ok {
			opts = append(opts, live.WithKVStringCookies(u, v))
		}
		opts = append(opts, live.WithQuality(room.Quality))
		l, err := live.New(u, inst.Cache, opts...)
		if err != nil {
			logger.WithField("url", room).Error(err.Error())
			continue
		}
		if _, ok := inst.Lives[l.GetLiveId()]; ok {
			logger.Errorf("%s 已存在!", room)
			continue
		}
		inst.Lives[l.GetLiveId()] = l
		room.LiveId = l.GetLiveId()
	}

	// 如果配置中启用了RPC服务器，启动RPC服务器。
	if inst.Config.RPC.Enable {
		if err := servers.NewServer(ctx).Start(ctx); err != nil {
			logger.WithError(err).Fatalf("初始化服务器失败")
		}
	}

	// 创建监听器管理器和录制器管理器，并启动它们。
	lm := listeners.NewManager(ctx)
	rm := recorders.NewManager(ctx)
	pm := pushers.NewManager(ctx)
	if err := lm.Start(ctx); err != nil {
		logger.Fatalf("初始化监听器管理器失败，错误: %s", err)
	}
	if err := rm.Start(ctx); err != nil {
		logger.Fatalf("初始化录制器管理器失败，错误: %s", err)
	}
	if err := pm.Start(ctx); err != nil {
		logger.Fatalf("初始化推送器管理器失败，错误: %s", err)
	}

	// 初始化指标收集器并启动它。
	if err = metrics.NewCollector(ctx).Start(ctx); err != nil {
		logger.Fatalf("初始化指标收集器失败，错误: %s", err)
	}

	// 遍历所有直播房间，如果房间配置为正在监听，则添加到监听器管理器。
	for _, _live := range inst.Lives {
		room, err := inst.Config.GetLiveRoomByUrl(_live.GetRawUrl())
		if err != nil {
			logger.WithFields(map[string]interface{}{"room": _live.GetRawUrl()}).Error(err)
			panic(err)
		}
		if room.Listen {
			if err := lm.AddListener(ctx, _live); err != nil {
				logger.WithFields(map[string]interface{}{"url": _live.GetRawUrl()}).Error(err)
			}
		}
		// 休眠5秒钟，避免过于频繁的操作。
		time.Sleep(time.Second * 5)
	}

	// 创建一个用于捕获信号的通道。
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		// 如果配置中启用了RPC服务器，关闭RPC服务器。
		if inst.Config.RPC.Enable {
			inst.Server.Close(ctx)
		}
		// 关闭监听器管理器和录制器管理器。
		inst.ListenerManager.Close(ctx)
		inst.RecorderManager.Close(ctx)
	}()

	// 等待程序实例的WaitGroup计数为0，即等待所有协程结束。
	inst.WaitGroup.Wait()
	logger.Info("再见~")
}
