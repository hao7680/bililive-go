package counter

import (
	"io"
)

// Counter 定义了一个计数器接口，用于返回当前计数值。
type Counter interface {
	Count() uint
}

// CountReader 定义了一个接口，将计数器和 io.Reader 结合在一起。
type CountReader interface {
	Counter
	io.Reader
}

// CountWriter 定义了一个接口，将计数器和 io.Writer 结合在一起。
type CountWriter interface {
	Counter
	io.Writer
}

// countReader 结构实现了 CountReader 接口。
type countReader struct {
	r     io.Reader // 嵌入的 io.Reader 接口
	total uint      // 计数器总数
}

// NewCountReader 创建一个新的 CountReader。
func NewCountReader(r io.Reader) CountReader {
	return &countReader{r: r}
}

// Count 返回当前计数值。
func (r *countReader) Count() uint {
	return r.total
}

// Read 从嵌入的 io.Reader 中读取数据，并更新计数值。
func (r *countReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p) // 调用嵌入的 io.Reader 的 Read 方法
	r.total += uint(n)    // 更新计数器
	return n, err         // 返回读取的字节数和错误
}

// countWriter 结构实现了 CountWriter 接口。
type countWriter struct {
	w     io.Writer // 嵌入的 io.Writer 接口
	total uint      // 计数器总数
}

// NewCountWriter 创建一个新的 CountWriter。
func NewCountWriter(w io.Writer) CountWriter {
	return &countWriter{w: w}
}

// Count 返回当前计数值。
func (w *countWriter) Count() uint {
	return w.total
}

// Write 将数据写入嵌入的 io.Writer，并更新计数值。
func (w *countWriter) Write(p []byte) (int, error) {
	n, err := w.w.Write(p) // 调用嵌入的 io.Writer 的 Write 方法
	w.total += uint(n)     // 更新计数器
	return n, err          // 返回写入的字节数和错误
}
