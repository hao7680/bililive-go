package flv

import "context"

type (
	SoundFormat   uint8
	SoundRate     uint8
	SoundSize     uint8
	SoundType     uint8
	AACPacketType uint8

	AudioTagHeader struct {
		SoundFormat   SoundFormat
		SoundRate     SoundRate
		SoundSize     SoundSize
		SoundType     SoundType
		AACPacketType AACPacketType
	}
)

const (
	// SoundFormat
	LPCM_PE  SoundFormat = 0 // 线性PCM，平台字节序
	ADPCM    SoundFormat = 1
	MP3      SoundFormat = 2
	LPCM_LE  SoundFormat = 3 // 线性PCM，小端字节序
	AAC      SoundFormat = 10
	Speex    SoundFormat = 11
	MP3_8kHz SoundFormat = 14 // MP3 8千赫兹

	// SoundRate
	Rate5kHz  SoundRate = 0 // 5.5千赫兹
	Rate11kHz SoundRate = 1 // 11千赫兹
	Rate22kHz SoundRate = 2 // 22千赫兹
	Rate44kHz SoundRate = 3 // 44千赫兹

	// SoundSize
	Sample8  uint8 = 0 // 8位样本
	Sample16 uint8 = 1 // 16位样本

	// SoundType
	Mono   SoundType = 0 // 单声道音频
	Stereo SoundType = 1 // 立体声音频

	// AACPacketType
	AACSeqHeader AACPacketType = 0
	AACRaw       AACPacketType = 1
)

// parseAudioTag 解析音频标签
func (p *Parser) parseAudioTag(ctx context.Context, length, timestamp uint32) (*AudioTagHeader, error) {
	b, err := p.i.ReadByte()
	l := length - 1
	if err != nil {
		return nil, err
	}
	tag := new(AudioTagHeader)

	tag.SoundFormat = SoundFormat(b >> 4 & 15)
	tag.SoundRate = SoundRate(b >> 2 & 3)
	tag.SoundSize = SoundSize(b >> 1 & 1)
	tag.SoundType = SoundType(b & 1)

	if tag.SoundFormat == AAC {
		b, err := p.i.ReadByte()
		l -= 1
		if err != nil {
			return nil, err
		}
		tag.AACPacketType = AACPacketType(b)
	}

	// 写入标签头 && 音频标签头 && AACPacketType
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
