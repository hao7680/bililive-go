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
	"github.com/yuhaohwang/bililive-go/src/recorders"
	"github.com/yuhaohwang/bililive-go/src/servers"
)

// 获取配置信息
func getConfig() (*configs.Config, error) {
	var config *configs.Config
	if *flag.Conf != "" {
		c, err := configs.NewConfigWithFile(*flag.Conf)
		if err != nil {
			return nil, err
		}
		config = c
	} else {
		config = flag.GenConfigFromFlags()
	}
	if !config.RPC.Enable && len(config.LiveRooms) == 0 {
		// 如果配置无效，则尝试使用可执行文件旁边的config.yml文件。
		config, err := getConfigBesidesExecutable()
		if err == nil {
			return config, config.Verify()
		}
	}
	return config, config.Verify()
}

// 获取可执行文件旁边的配置信息
func getConfigBesidesExecutable() (*configs.Config, error) {
	exePath, err := os.Executable()
	if err != nil {
		return nil, err
	}
	configPath := filepath.Join(filepath.Dir(exePath), "config.yml")
	config, err := configs.NewConfigWithFile(configPath)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func main() {
	config, err := getConfig()
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}

	inst := new(instance.Instance)
	inst.Config = config
	// TODO: 用哈希表替换gcache。
	// LRU似乎在这里不是必要的。
	inst.Cache = gcache.New(1024).LRU().Build()
	ctx := context.WithValue(context.Background(), instance.Key, inst)

	logger := log.New(ctx)
	logger.Infof("%s 版本: %s 启动链接", consts.AppName, consts.AppVersion)
	if config.File != "" {
		logger.Debugf("配置路径: %s.", config.File)
		logger.Debugf("其他标志已被忽略.")
	} else {
		logger.Debugf("未使用配置文件.")
		logger.Debugf("标志: %s 被使用.", os.Args)
	}
	logger.Debugf("%+v", consts.AppInfo)
	logger.Debugf("%+v", inst.Config)

	if !utils.IsFFmpegExist(ctx) {
		logger.Fatalln("未找到FFmpeg二进制文件，请检查.")
	}

	events.NewDispatcher(ctx)

	inst.Lives = make(map[live.ID]live.Live)
	for index, _ := range inst.Config.LiveRooms {
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

	if inst.Config.RPC.Enable {
		if err := servers.NewServer(ctx).Start(ctx); err != nil {
			logger.WithError(err).Fatalf("初始化服务器失败")
		}
	}
	lm := listeners.NewManager(ctx)
	rm := recorders.NewManager(ctx)
	if err := lm.Start(ctx); err != nil {
		logger.Fatalf("初始化监听器管理器失败，错误: %s", err)
	}
	if err := rm.Start(ctx); err != nil {
		logger.Fatalf("初始化录制器管理器失败，错误: %s", err)
	}

	if err = metrics.NewCollector(ctx).Start(ctx); err != nil {
		logger.Fatalf("初始化指标收集器失败，错误: %s", err)
	}

	for _, _live := range inst.Lives {
		room, err := inst.Config.GetLiveRoomByUrl(_live.GetRawUrl())
		if err != nil {
			logger.WithFields(map[string]interface{}{"room": _live.GetRawUrl()}).Error(err)
			panic(err)
		}
		if room.IsListening {
			if err := lm.AddListener(ctx, _live); err != nil {
				logger.WithFields(map[string]interface{}{"url": _live.GetRawUrl()}).Error(err)
			}
		}
		time.Sleep(time.Second * 5)
	}

	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		if inst.Config.RPC.Enable {
			inst.Server.Close(ctx)
		}
		inst.ListenerManager.Close(ctx)
		inst.RecorderManager.Close(ctx)
	}()

	inst.WaitGroup.Wait()
	logger.Info("再见~")
}
