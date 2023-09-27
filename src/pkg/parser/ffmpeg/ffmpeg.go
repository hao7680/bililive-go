package ffmpeg

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strconv"
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
	args := []string{
		"-nostats",
		"-progress", "-",
		"-y", "-re",
		"-user_agent", userAgent,
		"-referer", live.GetRawUrl(),
		"-rw_timeout", p.timeoutInUs,
		"-i", url.String(),
		"-c", "copy",
		"-bsf:a", "aac_adtstoasc",
	}

	inst := instance.GetInstance(ctx)
	MaxFileSize := inst.Config.VideoSplitStrategies.MaxFileSize
	if MaxFileSize < 0 {
		inst.Logger.Infof("无效的MaxFileSize：%d", MaxFileSize)
	} else if MaxFileSize > 0 {
		args = append(args, "-fs", strconv.Itoa(MaxFileSize))
	}

	args = append(args, file)
	p.cmd = exec.Command(ffmpegPath, args...)
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
