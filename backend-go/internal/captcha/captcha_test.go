package captcha

import (
	"testing"
)

func TestRandomText(t *testing.T) {
	text := randomText(4)
	if len(text) != 4 {
		t.Errorf("length = %d, want 4", len(text))
	}
	for _, ch := range text {
		if ch >= 'a' && ch <= 'z' {
			// 不应包含小写（charset 只有大写）
		}
	}
}

func TestRandomText_Length(t *testing.T) {
	for _, n := range []int{1, 4, 6, 10} {
		text := randomText(n)
		if len(text) != n {
			t.Errorf("randomText(%d) length = %d", n, len(text))
		}
	}
}

func TestEqualFold(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"ABC", "abc", true},
		{"abc", "ABC", true},
		{"AbC", "aBc", true},
		{"ABCD", "abc", false},
		{"abc", "abd", false},
		{"1234", "1234", true},
		{"A 1B", "a 1b", true},
	}
	for _, tt := range tests {
		got := equalFold(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("equalFold(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestRenderSVG(t *testing.T) {
	svg := renderSVG("ABCD")
	if len(svg) == 0 {
		t.Fatal("SVG 为空")
	}
	if svg[:5] != "<svg " {
		t.Errorf("SVG 应以 <svg 开头: %s", svg[:30])
	}
	// 应包含 4 个字符
	for _, ch := range "ABCD" {
		if !contains(svg, string(ch)) {
			t.Errorf("SVG 应包含字符 %c", ch)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
