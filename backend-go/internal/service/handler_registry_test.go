package service

import (
	"testing"
)

func TestRegisterAndGetHandler(t *testing.T) {
	handler := func() (string, error) {
		return "test result", nil
	}
	RegisterHandler("test.ExampleHandler", handler)

	got := GetHandler("test.ExampleHandler")
	if got == nil {
		t.Fatal("GetHandler 返回 nil")
	}

	result, err := got()
	if err != nil {
		t.Fatalf("handler 执行错误: %v", err)
	}
	if result != "test result" {
		t.Errorf("result = %s, want test result", result)
	}
}

func TestGetHandler_NotFound(t *testing.T) {
	got := GetHandler("nonexistent.Handler")
	if got != nil {
		t.Error("找不到的 handler 应返回 nil")
	}
}

func TestSampleTask(t *testing.T) {
	result, err := SampleTask()
	if err != nil {
		t.Fatalf("SampleTask 失败: %v", err)
	}
	if result == "" {
		t.Error("SampleTask 返回值为空")
	}
}

func TestBuiltinHandlers(t *testing.T) {
	t.Run("jobs.SampleTask 已注册", func(t *testing.T) {
		h := GetHandler("jobs.SampleTask")
		if h == nil {
			t.Fatal("内置 handler jobs.SampleTask 未注册")
		}
		result, err := h()
		if err != nil {
			t.Errorf("执行失败: %v", err)
		}
		if result == "" {
			t.Error("结果为空")
		}
	})
	t.Run("jobs.NtpSyncPlaceholder 已注册", func(t *testing.T) {
		h := GetHandler("jobs.NtpSyncPlaceholder")
		if h == nil {
			t.Fatal("内置 handler jobs.NtpSyncPlaceholder 未注册")
		}
		_, err := h()
		if err != nil {
			t.Errorf("执行失败: %v", err)
		}
	})
}
