package flv

import "context"

// parseTag 解析FLV文件中的标签。
func (p *Parser) parseTag(ctx context.Context) error {
	p.tagCount += 1

	// 读取标签头部数据
	b, err := p.i.ReadN(15)
	if err != nil {
		return err
	}

	// 解析标签类型、长度和时间戳
	tagType := uint8(b[4])
	length := uint32(b[5])<<16 | uint32(b[6])<<8 | uint32(b[7])
	timeStamp := uint32(b[8])<<16 | uint32(b[9])<<8 | uint32(b[10]) | uint32(b[11])<<24

	// 根据标签类型进行不同的处理
	switch tagType {
	case audioTag:
		if _, err := p.parseAudioTag(ctx, length, timeStamp); err != nil {
			return err
		}
	case videoTag:
		if _, err := p.parseVideoTag(ctx, length, timeStamp); err != nil {
			return err
		}
	case scriptTag:
		return p.parseScriptTag(ctx, length)
	default:
		return ErrUnknownTag
	}

	return nil
}
