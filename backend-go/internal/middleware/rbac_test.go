package middleware

import "testing"

func TestInvalidRuleCannotGrantPermission(t *testing.T) {
	for _, rule := range []string{":/api/v1/users/*", "GET: /api/v1/users/*", "GET:\n/api/v1/users/*", "GET:/api/v1/users/\u00a0*"} {
		if matchPath("GET", "/api/v1/users/1", rule) {
			t.Errorf("invalid rule granted access: %q", rule)
		}
	}
}
