package servers

import (
	"encoding/json"
	"net/http"
)

const (
	contentType     = "Content-Type"     // 定义 Content-Type 常量
	contentTypeJSON = "application/json" // 定义 JSON 内容类型常量
)

// writeMsg 函数用于向 ResponseWriter 写入指定的 HTTP 响应消息和状态码。
func writeMsg(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	_, _ = w.Write([]byte(msg))
}

// writeJSON 函数用于将对象 obj 转换为 JSON 格式并写入 ResponseWriter。
func writeJSON(w http.ResponseWriter, obj interface{}) {
	b, err := json.Marshal(obj)
	if err != nil {
		writeMsg(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set(contentType, contentTypeJSON)
	_, _ = w.Write(b)
}

// writeJsonWithStatusCode 函数用于将对象 obj 转换为 JSON 格式，并写入带有指定状态码的 ResponseWriter。
func writeJsonWithStatusCode(w http.ResponseWriter, code int, obj interface{}) {
	b, err := json.Marshal(obj)
	if err != nil {
		writeMsg(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(code)
	w.Header().Set(contentType, contentTypeJSON)
	_, _ = w.Write(b)
}
