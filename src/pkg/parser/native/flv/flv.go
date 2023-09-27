package flv

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime/debug"
	"sync"

	"github.com/yuhaohwang/bililive-go/src/instance"
	"github.com/yuhaohwang/bililive-go/src/live"
	"github.com/yuhaohwang/bililive-go/src/pkg/parser"
	"github.com/yuhaohwang/bililive-go/src/pkg/reader"
	"github.com/yuhaohwang/bililive-go/src/pkg/utils"
)

const (
	Name = "native"

	audioTag  uint8 = 8
	videoTag  uint8 = 9
	scriptTag uint8 = 18

	ioRetryCount int = 3
)

var (
	flvSign = []byte{0x46, 0x4c, 0x56, 0x01} // FLV版本01

	ErrNotFlvStream = errors.New("非FLV流")
	ErrUnknownTag   = errors.New("未知标签")
)

func init() {
	parser.Register(Name, new(builder))
}

type builder struct{}

func (b *builder) Build(cfg map[string]string) (parser.Parser, error) {
	// timeout, err := time.ParseDuration(cfg["timeout_in_us"] + "us")
	// if err != nil {
	// 	timeout = time.Minute
	// }
	return &Parser{
		Metadata:  Metadata{},
		hc:        &http.Client{},
		stopCh:    make(chan struct{}),
		closeOnce: new(sync.Once),
	}, nil
}

type Metadata struct {
	HasVideo, HasAudio bool
}

type Parser struct {
	Metadata Metadata

	i              *reader.BufferedReader
	o              io.Writer
	avcHeaderCount uint8
	tagCount       uint32

	hc        *http.Client
	stopCh    chan struct{}
	closeOnce *sync.Once
}

// ParseLiveStream 解析直播流
func (p *Parser) ParseLiveStream(ctx context.Context, url *url.URL, live live.Live, file string) error {
	// 初始化输入流
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Add("User-Agent", "Chrome/59.0.3071.115")
	resp, err := p.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	p.i = reader.New(resp.Body)
	defer p.i.Free()

	// 初始化输出流
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	p.o = f
	defer f.Close()

	// 开始解析
	return p.doParse(ctx)
}

// Stop 停止解析
func (p *Parser) Stop() error {
	p.closeOnce.Do(func() {
		close(p.stopCh)
	})
	return nil
}

// doParse 执行解析
func (p *Parser) doParse(ctx context.Context) error {
	// 解析FLV文件头
	b, err := p.i.ReadN(9)
	if err != nil {
		return err
	}
	// 验证FLV文件头
	if !bytes.Equal(b[:4], flvSign) {
		return ErrNotFlvStream
	}
	// 设置视频和音频标志位
	p.Metadata.HasVideo = uint8(b[4])&(1<<2) != 0
	p.Metadata.HasAudio = uint8(b[4])&1 != 0

	// 验证文件头偏移量必须为9
	if binary.BigEndian.Uint32(b[5:]) != 9 {
		return ErrNotFlvStream
	}

	// 写入FLV文件头
	if err := p.doWrite(ctx, p.i.AllBytes()); err != nil {
		return err
	}
	p.i.Reset()

	// 开始解析标签
	for {
		select {
		case <-p.stopCh:
			return nil
		default:
			if err := p.parseTag(ctx); err != nil {
				return err
			}
		}
	}
}

// doCopy 复制数据
func (p *Parser) doCopy(ctx context.Context, n uint32) error {
	if writtenCount, err := io.CopyN(p.o, p.i, int64(n)); err != nil || writtenCount != int64(writtenCount) {
		utils.PrintStack(ctx)
		if err == nil {
			err = fmt.Errorf("doCopy(%d), %d 字节已写入", n, writtenCount)
		}
		return err
	}
	return nil
}

// doWrite 写入数据
func (p *Parser) doWrite(ctx context.Context, b []byte) error {
	inst := instance.GetInstance(ctx)
	logger := inst.Logger
	leftInputSize := len(b)
	for retryLeft := ioRetryCount; retryLeft > 0 && leftInputSize > 0; retryLeft-- {
		writtenCount, err := p.o.Write(b[len(b)-leftInputSize:])
		leftInputSize -= writtenCount
		if err != nil {
			logger.Debugf(string(debug.Stack()))
			return err
		}
		if leftInputSize != 0 {
			logger.Debugf("doWrite() 剩余 %d 字节待写入", leftInputSize)
		}
	}
	if leftInputSize != 0 {
		return fmt.Errorf("doWrite([%d]byte) 尝试了 %d 次，仍然有 %d 字节待写入", len(b), ioRetryCount, leftInputSize)
	}
	return nil
}
