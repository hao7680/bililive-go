package flv

import "context"

type DataType uint8

const (
	Number          DataType = 0
	Boolean         DataType = 1
	String          DataType = 2
	Object          DataType = 3
	Null            DataType = 5
	Undefined       DataType = 6
	Reference       DataType = 7
	ECMAArray       DataType = 8
	ObjectEndMarker DataType = 9
	StrictArray     DataType = 10
	Date            DataType = 11
	LongString      DataType = 12
)

func (p *Parser) parseScriptTag(ctx context.Context, length uint32) error {
	// TODO: 解析脚本标签内容
	// 写入标签头
	if err := p.doWrite(ctx, p.i.AllBytes()); err != nil {
		return err
	}
	p.i.Reset()
	// 写入内容
	if err := p.doCopy(ctx, length); err != nil {
		return err
	}
	return nil
}
