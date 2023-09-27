package servers

import (
	"net/http"

	"github.com/yuhaohwang/bililive-go/src/instance"
)

// log 函数是一个中间件，用于记录 HTTP 请求的日志信息。
func log(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 获取应用程序实例，并使用日志记录 HTTP 请求的相关信息。
		instance.GetInstance(r.Context()).Logger.WithFields(map[string]interface{}{
			"Method":     r.Method,     // 请求方法，如 GET、POST 等。
			"Path":       r.RequestURI, // 请求路径，如 "/api/v1/user"。
			"RemoteAddr": r.RemoteAddr, // 请求的远程地址。
		}).Debug("Http Request") // 记录 DEBUG 级别的日志消息。
		handler.ServeHTTP(w, r) // 调用下一个处理程序来处理请求。
	})
}
