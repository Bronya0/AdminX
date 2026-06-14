// Package response 提供统一的 {code, msg, data} 响应封装。
//
// 对齐 Django 的 StandardJsonRenderer：
//   - 所有响应 HTTP 状态码恒为 200
//   - 业务状态在 body.code 字段（200/201/400/401/403/404/423/500...）
//   - 结构: {"code": int, "msg": string, "data": any}
package response

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	apperr "djangoadminx/pkg/errors"
)

// Body 统一响应结构。
type Body struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// PaginatedData 分页响应的 data 字段结构（对齐 Django StandardPagination）。
type PaginatedData struct {
	Count    int64        `json:"count"`
	Next     string       `json:"next"`
	Previous string       `json:"previous"`
	Results  interface{}  `json:"results"`
}

// OK 返回成功响应: {code:200, msg:"success", data:...}
func OK(c *gin.Context, data interface{}) {
	c.JSON(200, Body{Code: 200, Msg: "success", Data: data})
}

// Created 返回创建成功响应: {code:201, msg:"success", data:...}
func Created(c *gin.Context, data interface{}) {
	c.JSON(200, Body{Code: 201, Msg: "success", Data: data})
}

// NoContent 返回无内容响应: {code:204, msg:"success", data:null}
func NoContent(c *gin.Context) {
	c.JSON(200, Body{Code: 204, Msg: "success", Data: nil})
}

// Fail 返回失败响应: {code:code, msg:msg, data:null}
// HTTP 状态码恒为 200（对齐 Django StandardJsonRenderer）。
func Fail(c *gin.Context, code int, msg string) {
	c.JSON(200, Body{Code: code, Msg: msg, Data: nil})
}

// FailWithData 返回失败响应但携带 data（用于字段级校验错误等）。
func FailWithData(c *gin.Context, code int, msg string, data interface{}) {
	c.JSON(200, Body{Code: code, Msg: msg, Data: data})
}

// Paginated 返回分页列表响应。
func Paginated(c *gin.Context, count int64, next, previous string, results interface{}) {
	OK(c, PaginatedData{
		Count:    count,
		Next:     next,
		Previous: previous,
		Results:  results,
	})
}

// Error 统一错误响应入口。
// 从 err 提取 AppError 的 Code/Message；非 AppError 视为 500。
// 5xx 错误会记录日志（含 request_id）。
func Error(c *gin.Context, logger *slog.Logger, err error) {
	appErr := apperr.FromError(err)
	if appErr.Code >= 500 && logger != nil {
		reqID := c.GetString("request_id")
		if reqID == "" {
			reqID = uuid.NewString()
		}
		logger.ErrorContext(c.Request.Context(), "服务异常",
			"error", err,
			"request_id", reqID,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
		)
	}
	Fail(c, appErr.Code, appErr.Message)
}

// BindingError 处理 gin ShouldBind* 的参数绑定错误，
// 返回 400 + 错误详情。
func BindingError(c *gin.Context, err error) {
	FailWithData(c, 400, "请求参数错误: "+err.Error(), gin.H{
		"binding_error": err.Error(),
	})
}
