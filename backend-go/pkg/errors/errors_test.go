package errors

import (
	"errors"
	"testing"
)

func TestNew(t *testing.T) {
	err := New(400, "参数错误")
	if err.Code != 400 {
		t.Errorf("Code = %d, want 400", err.Code)
	}
	if err.Message != "参数错误" {
		t.Errorf("Message = %s, want 参数错误", err.Message)
	}
}

func TestWrap(t *testing.T) {
	root := errors.New("底层错误")
	err := Wrap(500, "内部错误", root)
	if err.Code != 500 {
		t.Errorf("Code = %d, want 500", err.Code)
	}
	if !errors.Is(err, root) {
		t.Error("errors.Is 应匹配底层错误")
	}
}

func TestAppError_Is(t *testing.T) {
	a := New(404, "找不到")
	b := New(404, "资源不存在")
	if !errors.Is(a, b) {
		t.Error("相同 Code 的 AppError 应互相匹配")
	}
	c := New(500, "内部错误")
	if errors.Is(a, c) {
		t.Error("不同 Code 的 AppError 不应匹配")
	}
	// 非 AppError
	if errors.Is(a, errors.New("普通错误")) {
		t.Error("AppError 不应匹配非 AppError")
	}
}

func TestAppError_Unwrap(t *testing.T) {
	root := errors.New("root")
	err := Wrap(500, "wrapped", root)
	unwrapped := err.Unwrap()
	if unwrapped != root {
		t.Error("Unwrap 应返回底层错误")
	}
}

func TestAppError_Error(t *testing.T) {
	t.Run("无底层错误", func(t *testing.T) {
		err := New(404, "找不到")
		if err.Error() != "找不到" {
			t.Errorf("Error() = %s, want 找不到", err.Error())
		}
	})
	t.Run("有底层错误", func(t *testing.T) {
		root := errors.New("timeout")
		err := Wrap(500, "内部错误", root)
		if err.Error() != "内部错误: timeout" {
			t.Errorf("Error() = %s, want 内部错误: timeout", err.Error())
		}
	})
}

func TestFromError(t *testing.T) {
	t.Run("AppError 原样返回", func(t *testing.T) {
		app := New(423, "锁定")
		result := FromError(app)
		if result.Code != 423 {
			t.Errorf("Code = %d, want 423", result.Code)
		}
	})
	t.Run("标准 error 包装为 500", func(t *testing.T) {
		result := FromError(errors.New("未知错误"))
		if result.Code != 500 {
			t.Errorf("Code = %d, want 500", result.Code)
		}
	})
	t.Run("wrapped AppError", func(t *testing.T) {
		app := New(400, "参数错误")
		wrapped := Wrap(400, "重包", app)
		result := FromError(wrapped)
		if result.Code != 400 {
			t.Errorf("Code = %d, want 400", result.Code)
		}
	})
}

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  *AppError
		code int
	}{
		{"ErrBadRequest", ErrBadRequest, 400},
		{"ErrUnauthorized", ErrUnauthorized, 401},
		{"ErrForbidden", ErrForbidden, 403},
		{"ErrNotFound", ErrNotFound, 404},
		{"ErrConflict", ErrConflict, 409},
		{"ErrLocked", ErrLocked, 423},
		{"ErrInternal", ErrInternal, 500},
		{"ErrServiceDisabled", ErrServiceDisabled, 503},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.code {
				t.Errorf("%s.Code = %d, want %d", tt.name, tt.err.Code, tt.code)
			}
		})
	}
}
