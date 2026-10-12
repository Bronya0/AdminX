package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

type auditCall struct {
	action, model, objectID, objectRepr, operator, diff string
}

// fakeRecorder 记录调用参数的假审计写入器。
type fakeRecorder struct {
	calls []auditCall
	err   error
}

func (f *fakeRecorder) Record(action, modelName, objectID, objectRepr, operator, ip string,
	oldVals, newVals datatypes.JSON, diff string) error {
	f.calls = append(f.calls, auditCall{action, modelName, objectID, objectRepr, operator, diff})
	return f.err
}

// auditRouter 构造一条带 Audited 中间件的写路由（:id 用于验证路由参数兜底）。
func auditRouter(rec AuditRecorder, handler gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/jobs/:id/",
		func(c *gin.Context) { c.Set("username", "tester") },
		Audited(rec, AuditCreate, "Job"),
		handler)
	return r
}

func doPost(r *gin.Engine, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, path, nil))
	return w
}

func doPostJSON(r *gin.Engine, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func TestAudited_RecordsOnSuccess(t *testing.T) {
	rec := &fakeRecorder{}
	r := auditRouter(rec, func(c *gin.Context) {
		c.JSON(200, gin.H{"code": 201, "msg": "success",
			"data": gin.H{"id": "job-1", "name": "备份任务"}})
	})

	w := doPost(r, "/jobs/abc/")

	if len(rec.calls) != 1 {
		t.Fatalf("期望写入 1 条审计，实际 %d 条", len(rec.calls))
	}
	got := rec.calls[0]
	if got.action != "create" || got.model != "Job" {
		t.Errorf("动作/模型错误: %+v", got)
	}
	if got.objectID != "job-1" || got.objectRepr != "备份任务" {
		t.Errorf("对象应取自响应体 data: %+v", got)
	}
	if got.operator != "tester" {
		t.Errorf("操作人应取自 context username: %+v", got)
	}

	// 响应体必须原样透传：包装 ResponseWriter 不能吞掉 body
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应体不是合法 JSON: %v (%q)", err, w.Body.String())
	}
	if body["code"] != float64(201) {
		t.Errorf("响应体被改写: %v", body)
	}
}

func TestAudited_SkipsOnBusinessFailure(t *testing.T) {
	rec := &fakeRecorder{}
	r := auditRouter(rec, func(c *gin.Context) {
		c.JSON(200, gin.H{"code": 400, "msg": "参数错误", "data": nil})
	})

	doPost(r, "/jobs/abc/")

	if len(rec.calls) != 0 {
		t.Errorf("失败的写操作不应落审计，实际 %d 条", len(rec.calls))
	}
}

func TestAudited_FallsBackToRouteID(t *testing.T) {
	rec := &fakeRecorder{}
	r := auditRouter(rec, func(c *gin.Context) {
		// delete 类响应只回空 data，对象标识只能从路由 :id 取
		c.JSON(200, gin.H{"code": 204, "msg": "success", "data": nil})
	})

	doPost(r, "/jobs/job-9/")

	if len(rec.calls) != 1 {
		t.Fatalf("期望写入 1 条审计，实际 %d 条", len(rec.calls))
	}
	if got := rec.calls[0]; got.objectID != "job-9" || got.objectRepr != "job-9" {
		t.Errorf("应回落到路由 :id: %+v", got)
	}
}

func TestAudited_DiffSummaryUsesRequestFieldNames(t *testing.T) {
	rec := &fakeRecorder{}
	var seen string
	r := auditRouter(rec, func(c *gin.Context) {
		// handler 必须仍能读到完整请求体（中间件读完要复原）
		var in map[string]interface{}
		if err := c.ShouldBindJSON(&in); err != nil {
			t.Errorf("handler 读不到请求体: %v", err)
		}
		seen, _ = in["site_name"].(string)
		c.JSON(200, gin.H{"code": 200, "msg": "success", "data": nil})
	})

	doPostJSON(r, "/jobs/abc/", `{"site_name":"X","login_captcha_required":false}`)

	if seen != "X" {
		t.Errorf("请求体被中间件吞掉: %q", seen)
	}
	if len(rec.calls) != 1 {
		t.Fatalf("期望 1 条审计，实际 %d 条", len(rec.calls))
	}
	// 只记字段名、不记值：改密/建 webhook 的 body 含明文密码与密钥
	if got := rec.calls[0].diff; got != "login_captcha_required, site_name" {
		t.Errorf("变更摘要应为请求体字段名: %q", got)
	}
}

func TestAudited_RecorderErrorDoesNotBreakResponse(t *testing.T) {
	rec := &fakeRecorder{err: errors.New("db down")}
	r := auditRouter(rec, func(c *gin.Context) {
		c.JSON(200, gin.H{"code": 200, "msg": "success", "data": gin.H{"id": "job-1"}})
	})

	w := doPost(r, "/jobs/abc/")

	if w.Code != 200 {
		t.Errorf("审计写失败不应影响业务响应，HTTP=%d", w.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应体不是合法 JSON: %v", err)
	}
	if body["code"] != float64(200) {
		t.Errorf("业务响应被改写: %v", body)
	}
}
