// Package middleware — 引擎级请求体大小上限。
//
// encoding/json 读没有上限的 body 时会整份分配：一个 {"username":"<500MB 字符串>"}
// 打到登录接口就能把进程内存打爆（限流只管速率，管不住并发大包）。
// 所以在引擎层用 http.MaxBytesReader 兜底，任何 JSON 端点都不会无限读。
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// MaxBodyBytes JSON 接口的请求体上限（8MB）。
// 配配置/主题导入在 handler 层另有更严的 2MB 限制，这里只做兜底。
const MaxBodyBytes int64 = 8 << 20

// bodyLimitSkipPaths 必须跳过体积上限的路径（按子串匹配，兼容 base_path 前缀）：
//   - /files/upload/：文件上传用 MultipartReader 流式落盘，大小由 FileService 自己把关
//     （100MB 上限），套上 8MB 会让大文件上传直接失败；
//   - /ws/：WebSocket 升级请求（没有请求体）。
var bodyLimitSkipPaths = []string{"/files/upload/", "/ws/"}

// BodyLimit 把请求体包成 http.MaxBytesReader（只包 reader，不读取内容，不干扰解析）。
// 超限后 ShouldBindJSON 返回 *http.MaxBytesError，handler 走既有的参数错误分支（400），
// 不新增错误码。
func BodyLimit(maxBytes int64) gin.HandlerFunc {
	if maxBytes <= 0 {
		maxBytes = MaxBodyBytes
	}
	return func(c *gin.Context) {
		if c.Request != nil && c.Request.Body != nil && c.Request.URL != nil &&
			!skipBodyLimit(c.Request.URL.Path) {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}

// skipBodyLimit 判断该路径是否需要绕过体积上限。
func skipBodyLimit(path string) bool {
	for _, sub := range bodyLimitSkipPaths {
		if strings.Contains(path, sub) {
			return true
		}
	}
	return false
}
