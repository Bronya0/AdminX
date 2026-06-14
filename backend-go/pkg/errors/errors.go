// Package errors 定义应用级错误类型与哨兵错误。
//
// AppError 携带 HTTP 业务状态码（对齐 Django 的 {code,msg,data} 中的 code 值），
// handler 层通过 errors.As 提取 code 决定返回的业务码。
package errors

import (
	"errors"
	"fmt"
)

// AppError 应用错误，Code 对齐 Django StandardJsonRenderer 的 code 值。
type AppError struct {
	Code    int    // 业务状态码（200/400/401/403/404/409/423/500...）
	Message string // 用户可见的错误信息
	Err     error  // 原始错误（可选，不暴露给客户端）
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Err }

// Is 支持 errors.Is 比较 AppError 的 Code。
func (e *AppError) Is(target error) bool {
	t, ok := target.(*AppError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// New 构造一个 AppError。
func New(code int, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

// Wrap 包装一个底层错误为 AppError。
func Wrap(code int, message string, err error) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}

// ── 哨兵错误（常用，handler 可 errors.Is 匹配）──

var (
	ErrBadRequest      = New(400, "请求参数错误")
	ErrUnauthorized    = New(401, "未授权")
	ErrForbidden       = New(403, "禁止访问")
	ErrNotFound        = New(404, "资源不存在")
	ErrConflict        = New(409, "资源已存在")
	ErrLocked          = New(423, "账号已锁定")
	ErrInternal        = New(500, "服务器内部错误")
	ErrServiceDisabled = New(503, "服务不可用")
)

// FromError 从任意 error 推断 AppError；非 AppError 则视为 500。
func FromError(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return Wrap(500, "服务器内部错误", err)
}
