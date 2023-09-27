//go:generate mockgen -package recorders -destination mock_test.go github.com/yuhaohwang/bililive-go/src/recorders Recorder,Manager
package recorders

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/bluele/gcache"
	"github.com/sirupsen/logrus"

	"github.com/yuhaohwang/bililive-go/src/configs"
	"github.com/yuhaohwang/bililive-go/src/instance"
	"github.com/yuhaohwang/bililive-go/src/interfaces"
	"github.com/yuhaohwang/bililive-go/src/live"
	"github.com/yuhaohwang/bililive-go/src/pkg/events"
	"github.com/yuhaohwang/bililive-go/src/pkg/parser"
	"github.com/yuhaohwang/bililive-go/src/pkg/parser/ffmpeg"
	"github.com/yuhaohwang/bililive-go/src/pkg/parser/native/flv"
	"github.com/yuhaohwang/bililive-go/src/pkg/utils"
)

const (
	begin uint32 = iota
	pending
	running
	stopped
)

// for test
var (
	newParser = func(u *url.URL, useNativeFlvParser bool, cfg map[string]string) (parser.Parser, error) {
		parserName := ffmpeg.Name
		if strings.Contains(u.Path, ".flv") && useNativeFlvParser {
			parserName = flv.Name
		}
		return parser.New(parserName, cfg)
	}

	mkdir = func(path string) error {
		return os.MkdirAll(path, os.ModePerm)
	}

	removeEmptyFile = func(file string) {
		if stat, err := os.Stat(file); err == nil && stat.Size() == 0 {
			os.Remove(file)
		}
	}
)

// 默认的文件名模板
func getDefaultFileNameTmpl(config *configs.Config) *template.Template {
	return template.Must(template.New("filename").Funcs(utils.GetFuncMap(config)).
		Parse(`{{ .Live.GetPlatformCNName }}/{{ .HostName | filenameFilter }}/[{{ now | date "2006-01-02 15-04-05"}}][{{ .HostName | filenameFilter }}][{{ .RoomName | filenameFilter }}].flv`))
}

// Recorder 定义 Recorder 接口。
type Recorder interface {
	Start(ctx context.Context) error
	StartTime() time.Time
	GetStatus() (map[string]string, error)
	Close()
}

// recorder 是 Recorder 接口的实现。
type recorder struct {
	Live       live.Live
	OutPutPath string

	config     *configs.Config
	ed         events.Dispatcher
	logger     *interfaces.Logger
	cache      gcache.Cache
	startTime  time.Time
	parser     parser.Parser
	parserLock *sync.RWMutex

	stop  chan struct{}
	state uint32
}

// NewRecorder 创建一个新的 Recorder 实例。
func NewRecorder(ctx context.Context, live live.Live) (Recorder, error) {
	inst := instance.GetInstance(ctx)
	return &recorder{
		Live:       live,
		OutPutPath: instance.GetInstance(ctx).Config.OutPutPath,
		config:     inst.Config,
		cache:      inst.Cache,
		startTime:  time.Now(),
		ed:         inst.EventDispatcher.(events.Dispatcher),
		logger:     inst.Logger,
		state:      begin,
		stop:       make(chan struct{}),
		parserLock: new(sync.RWMutex),
	}, nil
}

// tryRecord 尝试录制直播流。
func (r *recorder) tryRecord(ctx context.Context) {
	// 获取直播流的URL列表
	urls, err := r.Live.GetStreamUrls()
	if err != nil || len(urls) == 0 {
		r.getLogger().WithError(err).Warn("无法获取直播流URL，将在5秒后重试...")
		time.Sleep(5 * time.Second)
		return
	}

	// 从缓存中获取直播信息
	obj, _ := r.cache.Get(r.Live)
	info := obj.(*live.Info)

	// 设置文件名模板
	tmpl := getDefaultFileNameTmpl(r.config)
	if r.config.OutputTmpl != "" {
		_tmpl, err := template.New("user_filename").Funcs(utils.GetFuncMap(r.config)).Parse(r.config.OutputTmpl)
		if err == nil {
			tmpl = _tmpl
		}
	}

	// 生成文件名
	buf := new(bytes.Buffer)
	if err = tmpl.Execute(buf, info); err != nil {
		panic(fmt.Sprintf("无法渲染文件名，错误：%v", err))
	}
	fileName := filepath.Join(r.OutPutPath, buf.String())
	outputPath, _ := filepath.Split(fileName)
	url := urls[0]

	// 如果URL中包含 "m3u8"，则将文件名更改为 .ts 扩展名
	if strings.Contains(url.Path, "m3u8") {
		fileName = fileName[:len(fileName)-4] + ".ts"
	}

	// 如果只有音频，将文件名更改为 .aac 扩展名
	if info.AudioOnly {
		fileName = fileName[:strings.LastIndex(fileName, ".")] + ".aac"
	}

	// 创建输出目录
	if err = mkdir(outputPath); err != nil {
		r.getLogger().WithError(err).Errorf("无法创建输出目录[%s]", outputPath)
		return
	}

	// 初始化解析器配置
	parserCfg := map[string]string{
		"timeout_in_us": strconv.Itoa(r.config.TimeoutInUs),
	}
	if r.config.Debug {
		parserCfg["debug"] = "true"
	}

	// 根据 URL 初始化解析器
	p, err := newParser(url, r.config.Feature.UseNativeFlvParser, parserCfg)
	if err != nil {
		r.getLogger().WithError(err).Error("初始化解析器失败")
		return
	}

	// 设置并关闭当前解析器
	r.setAndCloseParser(p)

	// metadata.json
	// 如果 fileName 有扩展名，则替换为 ".json"，否则添加 ".json"
	ext := filepath.Ext(fileName)
	jsonFilePath := ""
	if ext != "" {
		// 有扩展名，直接替换为 ".metadata.json"
		jsonFilePath = fileName[:len(fileName)-len(ext)] + ".metadata.json"
	} else {
		// 没有扩展名，直接加上 ".metadata.json"
		jsonFilePath = fileName + ".metadata.json"
	}

	// 记录开始时间
	r.startTime = time.Now()
	r.getLogger().Info("开始解析直播流(" + url.String() + ", " + fileName + ")")

	jsonData := info
	jsonData.Recording = true
	// 保存 JSON 数据到文件
	r.saveJSONToFile(jsonFilePath, jsonData)

	// 解析直播流并记录结果
	result := r.parser.ParseLiveStream(ctx, url, r.Live, fileName)
	r.getLogger().Println(result)

	// 记录结束时间
	r.getLogger().Info("结束解析直播流(" + url.String() + ", " + fileName + ")")

	jsonData.Recording = false
	// 再次保存 JSON 数据到文件
	r.saveJSONToFile(jsonFilePath, jsonData)

	// 移除空文件
	removeEmptyFile(fileName)

	// 获取 FFmpeg 路径
	ffmpegPath, err := utils.GetFFmpegPath(ctx)
	if err != nil {
		r.getLogger().WithError(err).Error("无法找到 FFmpeg")
		return
	}

	// 执行自定义命令或转换
	cmdStr := strings.Trim(r.config.OnRecordFinished.CustomCommandline, "")
	if len(cmdStr) > 0 {
		tmpl, err := template.New("custom_commandline").Funcs(utils.GetFuncMap(r.config)).Parse(cmdStr)
		if err != nil {
			r.getLogger().WithError(err).Error("自定义命令行解析失败")
			return
		}

		buf := new(bytes.Buffer)
		if err := tmpl.Execute(buf, struct {
			*live.Info
			FileName string
			Ffmpeg   string
		}{
			Info:     info,
			FileName: fileName,
			Ffmpeg:   ffmpegPath,
		}); err != nil {
			r.getLogger().WithError(err).Errorln("无法渲染自定义命令行")
			return
		}

		bash := ""
		args := []string{}
		switch runtime.GOOS {
		case "linux":
			bash = "sh"
			args = []string{"-c"}
		case "windows":
			bash = "cmd"
			args = []string{"/C"}
		default:
			r.getLogger().Warnln("不支持的操作系统", runtime.GOOS)
		}

		args = append(args, buf.String())
		r.getLogger().Debugf("开始执行自定义命令行: %s", args[1])
		cmd := exec.Command(bash, args...)
		if r.config.Debug {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}
		if err = cmd.Run(); err != nil {
			r.getLogger().WithError(err).Debugf("自定义命令行执行失败(%s %s)\n", bash, strings.Join(args, " "))
		} else if r.config.OnRecordFinished.DeleteFlvAfterConvert {
			os.Remove(fileName)
		}
		r.getLogger().Debugf("结束执行自定义命令行: %s", args[1])
	} else if r.config.OnRecordFinished.ConvertToMp4 {
		convertCmd := exec.Command(
			ffmpegPath,
			"-hide_banner",
			"-i",
			fileName,
			"-c",
			"copy",
			fileName+".mp4",
		)
		if err = convertCmd.Run(); err != nil {
			convertCmd.Process.Kill()
			r.getLogger().Debugln(err)
		} else if r.config.OnRecordFinished.DeleteFlvAfterConvert {
			os.Remove(fileName)
		}
	}
}

// run 启动录制器的主循环。
func (r *recorder) run(ctx context.Context) {
	for {
		select {
		case <-r.stop:
			return
		default:
			r.tryRecord(ctx)
		}
	}
}

// getParser 获取当前解析器。
func (r *recorder) getParser() parser.Parser {
	r.parserLock.RLock()         // 获取解析器互斥锁，允许多个协程同时读取解析器
	defer r.parserLock.RUnlock() // 在方法结束时释放解析器互斥锁

	return r.parser // 返回当前的解析器
}

// setAndCloseParser 设置解析器并关闭旧的解析器。
func (r *recorder) setAndCloseParser(p parser.Parser) {
	r.parserLock.Lock()         // 获取互斥锁，防止多个协程同时访问和修改解析器
	defer r.parserLock.Unlock() // 在方法结束时释放互斥锁，确保不会出现死锁

	if r.parser != nil { // 如果当前已经有一个解析器
		r.parser.Stop() // 调用当前解析器的 Stop 方法，关闭旧的解析器
	}
	r.parser = p // 将新的解析器赋值给 r.parser，替换旧的解析器
}

// Start 启动录制器。
func (r *recorder) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapUint32(&r.state, begin, pending) {
		return nil
	}
	go r.run(ctx)
	r.getLogger().Info("Record Start")
	r.ed.DispatchEvent(events.NewEvent(RecorderStart, r.Live))
	atomic.CompareAndSwapUint32(&r.state, pending, running)
	return nil
}

// StartTime 返回录制器启动的时间。
func (r *recorder) StartTime() time.Time {
	return r.startTime
}

// Close 关闭录制器。
func (r *recorder) Close() {
	if !atomic.CompareAndSwapUint32(&r.state, running, stopped) {
		return
	}
	close(r.stop)
	if p := r.getParser(); p != nil {
		p.Stop()
	}
	r.getLogger().Info("Record End")
	r.ed.DispatchEvent(events.NewEvent(RecorderStop, r.Live))
}

// getLogger 返回记录器实例。
func (r *recorder) getLogger() *logrus.Entry {
	return r.logger.WithFields(r.getFields())
}

// getFields 返回记录器的字段。
func (r *recorder) getFields() map[string]interface{} {
	obj, err := r.cache.Get(r.Live)
	if err != nil {
		return nil
	}
	info := obj.(*live.Info)
	return map[string]interface{}{
		"host": info.HostName,
		"room": info.RoomName,
	}
}

// GetStatus 获取录制器的状态。
func (r *recorder) GetStatus() (map[string]string, error) {
	statusP, ok := r.getParser().(parser.StatusParser)
	if !ok {
		return nil, ErrParserNotSupportStatus
	}
	return statusP.Status()
}

// saveJSONToFile 将 JSON 数据保存到文件
func (r *recorder) saveJSONToFile(jsonFilePath string, info *live.Info) error {

	// 将 info 结构体转换为 JSON 格式
	jsonData, err := info.MarshalJSON()
	if err != nil {
		r.getLogger().Info("编码JSON时发生错误:", err)
	}

	// 保存 JSON 数据到文件，覆盖同名文件
	err = os.WriteFile(jsonFilePath, jsonData, 0644)
	if err != nil {
		r.getLogger().Info("写入METADATA JSON文件时发生错误:", err)
	}

	r.getLogger().Info("已保存METADATA JSON至:", jsonFilePath)
	return nil
}
