package ffmpeg

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yuhaohwang/bililive-go/src/instance"
	"github.com/yuhaohwang/bililive-go/src/live"
	"github.com/yuhaohwang/bililive-go/src/pkg/parser"
	"github.com/yuhaohwang/bililive-go/src/pkg/utils"
)

const (
	Name = "ffmpeg"

	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/59.0.3071.115 Safari/537.36"
)

func init() {
	parser.Register(Name, new(builder))
}

type builder struct{}

func (b *builder) Build(cfg map[string]string) (parser.Parser, error) {
	debug := false
	if debugFlag, ok := cfg["debug"]; ok && debugFlag != "" {
		debug = true
	}
	return &Parser{
		debug:       debug,
		closeOnce:   new(sync.Once),
		statusReq:   make(chan struct{}, 1),
		statusResp:  make(chan map[string]string, 1),
		timeoutInUs: cfg["timeout_in_us"],
	}, nil
}

type Parser struct {
	cmd         *exec.Cmd
	cmdStdIn    io.WriteCloser
	cmdStdout   io.ReadCloser
	closeOnce   *sync.Once
	debug       bool
	timeoutInUs string

	statusReq  chan struct{}
	statusResp chan map[string]string
}

// scanFFmpegStatus 扫描FFmpeg的状态输出
func (p *Parser) scanFFmpegStatus() <-chan []byte {
	ch := make(chan []byte)
	br := bufio.NewScanner(p.cmdStdout)
	br.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}

		if idx := bytes.Index(data, []byte("progress=continue\n")); idx >= 0 {
			return idx + 1, data[0:idx], nil
		}

		return 0, nil, nil
	})
	go func() {
		defer close(ch)
		for br.Scan() {
			ch <- br.Bytes()
		}
	}()
	return ch
}

// decodeFFmpegStatus 解码FFmpeg的状态信息
func (p *Parser) decodeFFmpegStatus(b []byte) (status map[string]string) {
	status = map[string]string{
		"parser": Name,
	}
	s := bufio.NewScanner(bytes.NewReader(b))
	s.Split(bufio.ScanLines)
	for s.Scan() {
		split := bytes.SplitN(s.Bytes(), []byte("="), 2)
		if len(split) != 2 {
			continue
		}
		status[string(bytes.TrimSpace(split[0]))] = string(bytes.TrimSpace(split[1]))
	}
	return
}

// scheduler 启动调度程序来定期获取FFmpeg状态
func (p *Parser) scheduler() {
	defer close(p.statusResp)
	statusCh := p.scanFFmpegStatus()
	for {
		select {
		case <-p.statusReq:
			select {
			case b, ok := <-statusCh:
				if !ok {
					return
				}
				p.statusResp <- p.decodeFFmpegStatus(b)
			case <-time.After(time.Second * 3):
				p.statusResp <- nil
			}
		default:
			if _, ok := <-statusCh; !ok {
				return
			}
		}
	}
}

// Status 获取FFmpeg的状态信息
func (p *Parser) Status() (map[string]string, error) {
	// TODO: 检查解析器是否正在运行
	p.statusReq <- struct{}{}
	return <-p.statusResp, nil
}

// ParseLiveStream 解析直播流
func (p *Parser) ParseLiveStream(ctx context.Context, url *url.URL, live live.Live, file string) (err error) {
	ffmpegPath, err := utils.GetFFmpegPath(ctx)
	if err != nil {
		return err
	}

	encoder := "no"

	// 不编码
	args := []string{
		"-nostats",       // 禁止显示统计信息
		"-progress", "-", // 将进度信息输出到标准输出
		"-y", "-re", // 覆盖输出
		"-user_agent", userAgent, // UA
		"-referer", live.GetRawUrl(), // 直播间地址
		"-rw_timeout", p.timeoutInUs, // 读写超时
		"-i", url.String(), // 直播流
		"-c", "copy", // 不转码
		"-bsf:a", "aac_adtstoasc", // 音频比特流过滤器
		"-f", "flv",
	}

	// hevc_vaapi
	hevcArgs := []string{
		"-vaapi_device", "/dev/dri/renderD129", // 指定加速卡
		"-nostats",       // 禁止显示统计信息
		"-progress", "-", // 将进度信息输出到标准输出
		"-y", "-re", // 覆盖输出
		"-user_agent", userAgent, // UA
		"-referer", live.GetRawUrl(), // 直播间地址
		"-rw_timeout", p.timeoutInUs, // 读写超时
		"-i", url.String(), // 直播流
		"-vf", "format=nv12,hwupload", // 启用硬件上传
		"-c:v", "hevc_vaapi", // 视频编码器
		"-b:v", "5M", // 码率
		"-c:a", "aac", // 音频编码器
		"-bsf:a", "aac_adtstoasc", // 音频比特流过滤器
	}

	// h264_vaapi
	h264Args := []string{
		"-vaapi_device", "/dev/dri/renderD129", // 指定加速卡
		"-nostats",       // 禁止显示统计信息
		"-progress", "-", // 将进度信息输出到标准输出
		"-y", "-re", // 覆盖输出
		"-user_agent", userAgent, // UA
		"-referer", live.GetRawUrl(), // 直播间地址
		"-rw_timeout", p.timeoutInUs, // 读写超时
		"-i", url.String(), // 直播流
		"-vf", "format=nv12,hwupload", // 启用硬件上传
		"-c:v", "h264_vaapi", // 视频编码器
		"-b:v", "8M", // 码率
		"-c:a", "aac", // 音频编码器
		"-bsf:a", "aac_adtstoasc", // 音频比特流过滤器
	}

	fileName := strings.TrimSuffix(file, filepath.Ext(file))
	fileName = strings.TrimSuffix(fileName, "_%03d")

	if encoder == "hevc" {
		args = hevcArgs
		file = fileName + ".mp4"
	} else if encoder == "h264" {
		args = h264Args
	} else if encoder == "cache" {
		m3u8 := fileName + ".m3u8"
		args = append(args, "-f", "segment")       // 切片
		args = append(args, "-segment_time", "60") // 切片时长
		args = append(args, "-segment_wrap", "10") // 切片循环
		args = append(args, "-segment_list", m3u8) // 切片列表
		// args = append(args, "-fs", "50M")          // 切片大小
	}

	if fileName == filepath.Ext(file) {
		args = append(args, "-buffer_size", "250M") // 缓存
	}

	if encoder != "cache" {
		inst := instance.GetInstance(ctx)
		MaxFileSize := inst.Config.VideoSplitStrategies.MaxFileSize
		if MaxFileSize < 0 {
			inst.Logger.Infof("无效的MaxFileSize：%d", MaxFileSize)
		} else if MaxFileSize > 0 {
			args = append(args, "-fs", strconv.Itoa(MaxFileSize))
		}
	}

	args = append(args, file)

	p.cmd = exec.Command(ffmpegPath, args...)
	// 打印执行的命令
	fmt.Printf("Command to be executed: %s\n", p.cmd.String())

	if p.cmdStdIn, err = p.cmd.StdinPipe(); err != nil {
		return err
	}
	if p.cmdStdout, err = p.cmd.StdoutPipe(); err != nil {
		return err
	}
	if p.debug {
		p.cmd.Stderr = os.Stderr
	}
	if err = p.cmd.Start(); err != nil {
		p.cmd.Process.Kill()
		return err
	}
	go p.scheduler()
	return p.cmd.Wait()
}

// Stop 停止解析器
func (p *Parser) Stop() error {
	p.closeOnce.Do(func() {
		if p.cmd.ProcessState == nil {
			p.cmdStdIn.Write([]byte("q"))
		}
	})
	return nil
}
