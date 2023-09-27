package flv

import (
	"context"
	"errors"
)

type (
	FrameType     uint8
	CodeID        uint8
	AVCPacketType uint8

	VideoTagHeader struct {
		FrameType       FrameType
		CodeID          CodeID
		AVCPacketType   AVCPacketType
		CompositionTime uint32
	}
)

const (
	// 帧类型
	KeyFrame             FrameType = 1 // AVC中的关键帧，可寻址帧
	InterFrame           FrameType = 2 // AVC中的非关键帧，不可寻址帧
	DisposableInterFrame FrameType = 3 // 仅用于H.263
	GeneratedKeyFrame    FrameType = 4 // 仅服务器使用
	VideoInfoFrame       FrameType = 5 // 视频信息/命令帧

	// 编码标识
	H263Code          CodeID = 2 // Sorenson H.263
	ScreenVideoCode   CodeID = 3 // 屏幕视频
	VP6Code           CodeID = 4 // On2 VP6
	VP6AlphaCode      CodeID = 5 // 带Alpha通道的On2 VP6
	ScreenVideoV2Code CodeID = 6 // 屏幕视频版本2
	AVCCode           CodeID = 7 // AVC

	// AVC包类型
	AVCSeqHeader AVCPacketType = 0 // AVC序列头
	AVCNALU      AVCPacketType = 1 // NAL单元
	AVCEndSeq    AVCPacketType = 2 // AVC序列结束（不需要或不支持较低级别的NALU序列结束）
)

func (p *Parser) parseVideoTag(ctx context.Context, length, timestamp uint32) (*VideoTagHeader, error) {
	// 解析标签头部
	b, err := p.i.ReadByte()
	l := length - 1
	if err != nil {
		return nil, err
	}
	tag := new(VideoTagHeader)
	tag.FrameType = FrameType(b >> 4 & 15)
	tag.CodeID = CodeID(b & 15)

	if tag.CodeID == AVCCode {
		// 读取AVCPacketType
		b, err := p.i.ReadByte()
		l -= 1
		if err != nil {
			return nil, err
		}
		tag.AVCPacketType = AVCPacketType(b)
		switch tag.AVCPacketType {
		case AVCNALU:
			// 读取CompositionTime
			b, err := p.i.ReadN(3)
			l -= 3
			if err != nil {
				return nil, err
			}
			tag.CompositionTime = uint32(b[0])<<16 | uint32(b[1])<<8 | uint32(b[2])
		case AVCSeqHeader:
			p.avcHeaderCount++
			if p.avcHeaderCount > 1 {
				// 新的sps和pps
				return nil, errors.New("EOF 新的sps和pps")
			}
		}
	}

	// 写入标签头、视频标签头、AVCPacketType和CompositionTime
	if err := p.doWrite(ctx, p.i.AllBytes()); err != nil {
		return nil, err
	}
	p.i.Reset()
	// 写入内容
	if err := p.doCopy(ctx, l); err != nil {
		return nil, err
	}

	return tag, nil
}
