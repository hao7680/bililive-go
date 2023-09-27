package log

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/yuhaohwang/bililive-go/src/instance"
	"github.com/yuhaohwang/bililive-go/src/interfaces"
)

// New 创建一个新的日志记录器实例。
func New(ctx context.Context) *interfaces.Logger {
	// 获取应用程序实例。
	inst := instance.GetInstance(ctx)

	// 将默认日志级别设置为 Info，或者如果启用 Debug 模式则设置为 Debug。
	logLevel := logrus.InfoLevel
	if inst.Config.Debug {
		logLevel = logrus.DebugLevel
	}

	// 获取应用程序配置。
	config := inst.Config

	// 创建一个日志写入器列表，起始使用 os.Stderr。
	writers := []io.Writer{os.Stderr}

	// 定义日志输出文件夹。
	outputFolder := config.Log.OutPutFolder

	// 检查输出文件夹是否存在，如果不存在则创建。
	if _, err := os.Stat(outputFolder); os.IsNotExist(err) {
		log.Fatalf("错误: \"%s\", 无法确定日志输出文件夹: %s", err, outputFolder)
	} else {
		// 如果启用了 SaveEveryLog，为当前运行创建一个日志文件。
		if config.Log.SaveEveryLog {
			runID := time.Now().Format("run-2006-01-02-15-04-05")
			logLocation := filepath.Join(outputFolder, runID+".log")
			logFile, err := os.OpenFile(logLocation, os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Fatalf("无法打开日志文件 %s 以进行输出: %s", logLocation, err)
			} else {
				writers = append(writers, logFile)
			}
		}

		// 如果启用了 SaveLastLog，创建或截断默认的日志文件。
		if config.Log.SaveLastLog {
			logLocation := filepath.Join(outputFolder, "bililive-go.log")
			logFile, err := os.OpenFile(logLocation, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				log.Fatalf("无法打开默认日志文件 %s 以进行输出: %s", logLocation, err)
			} else {
				writers = append(writers, logFile)
			}
		}
	}

	// 使用指定配置创建日志记录器实例。
	logger := &interfaces.Logger{Logger: &logrus.Logger{
		Out: io.MultiWriter(writers...),
		Formatter: &logrus.TextFormatter{
			DisableColors:   true,
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		},
		Hooks: make(logrus.LevelHooks),
		Level: logLevel,
	}}

	// 设置应用程序日志记录器为创建的日志记录器实例。
	inst.Logger = logger

	return logger
}
