package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// 超过上限的 JSON 请求体必须在 bind 阶段失败（handler 统一走 400），不能整份进内存。
func TestBodyLimit_RejectsOversizedJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(BodyLimit(1 << 10))
	r.POST("/adminx/api/v1/accounts/login/", func(c *gin.Context) {
		var in struct {
			Username string `json:"username"`
		}
		if err := c.ShouldBindJSON(&in); err != nil {
			c.String(http.StatusBadRequest, "bind failed")
			return
		}
		c.String(http.StatusOK, "ok")
	})

	body := `{"username":"` + strings.Repeat("a", 8<<10) + `"}`
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/adminx/api/v1/accounts/login/", strings.NewReader(body)))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("超限请求体应 bind 失败返回 400，实际 %d", w.Code)
	}
}

// 正常大小的请求体不能被影响（中间件只包 reader，不读内容）。
func TestBodyLimit_NormalBodyPasses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(BodyLimit(1 << 10))
	r.POST("/adminx/api/v1/jobs/", func(c *gin.Context) {
		var in struct {
			Name string `json:"name"`
		}
		if err := c.ShouldBindJSON(&in); err != nil {
			c.String(http.StatusBadRequest, "bind failed")
			return
		}
		c.String(http.StatusOK, in.Name)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/adminx/api/v1/jobs/", strings.NewReader(`{"name":"job"}`)))
	if w.Code != http.StatusOK || w.Body.String() != "job" {
		t.Fatalf("正常请求应通过，实际 code=%d body=%q", w.Code, w.Body.String())
	}
}

// 文件上传走 MultipartReader 流式落盘（100MB 上限由 FileService 把关），
// 必须被白名单跳过：否则 8MB 限制会把大文件上传直接打断。
func TestBodyLimit_SkipsUploadAndWSPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const payload = 64 << 10 // 远超下面的 1KB 上限
	reader := func(c *gin.Context) {
		n, err := io.Copy(io.Discard, c.Request.Body)
		if err != nil {
			c.String(http.StatusBadRequest, "read error")
			return
		}
		c.String(http.StatusOK, strconv.FormatInt(n, 10))
	}

	r := gin.New()
	r.Use(BodyLimit(1 << 10))
	r.POST("/adminx/api/v1/files/upload/", reader)
	r.GET("/adminx/ws/log/", reader)

	for _, tc := range []struct{ method, path string }{
		{http.MethodPost, "/adminx/api/v1/files/upload/"},
		{http.MethodGet, "/adminx/ws/log/"},
	} {
		path := tc.path
		w := httptest.NewRecorder()
		body := strings.Repeat("a", payload)
		r.ServeHTTP(w, httptest.NewRequest(tc.method, path, strings.NewReader(body)))
		if w.Code != http.StatusOK {
			t.Fatalf("%s 应跳过体积上限，实际 code=%d body=%q", path, w.Code, w.Body.String())
		}
		if w.Body.String() != strconv.Itoa(payload) {
			t.Fatalf("%s 应能读完整个请求体，实际 %s", path, w.Body.String())
		}
	}
}
