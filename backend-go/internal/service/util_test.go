package service

import (
	"testing"
)

func TestMustJSON(t *testing.T) {
	t.Run("正常数组", func(t *testing.T) {
		result := mustJSON([]string{"/api/users/", "/api/roles/"})
		if len(result) == 0 {
			t.Fatal("结果为空")
		}
		s := string(result)
		if s != `["/api/users/","/api/roles/"]` {
			t.Errorf("result = %s", s)
		}
	})
	t.Run("空数组", func(t *testing.T) {
		result := mustJSON([]string{})
		if string(result) != `[]` {
			t.Errorf("result = %s, want []", string(result))
		}
	})
	t.Run("nil 切片", func(t *testing.T) {
		result := mustJSON(nil)
		if string(result) != `[]` {
			t.Errorf("result = %s, want []", string(result))
		}
	})
	t.Run("含特殊字符的路径", func(t *testing.T) {
		result := mustJSON([]string{"/api/config/", "/api/menu?type=dir"})
		if len(result) == 0 {
			t.Fatal("结果为空")
		}
	})
}
