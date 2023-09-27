package live

import (
	"github.com/yuhaohwang/requests"
)

// CommonUserAgent 是一个常用的 HTTP User-Agent 头，模拟 Chrome 浏览器的用户代理。
var CommonUserAgent = requests.UserAgent(userAgent)

const (
	// userAgent 包含了一个常用的浏览器 User-Agent 字符串，用于 HTTP 请求中。
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/59.0.3071.115 Safari/537.36"
)
