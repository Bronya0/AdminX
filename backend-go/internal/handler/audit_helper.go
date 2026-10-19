package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"

	"adminx/internal/service"
)

// auditRecordDiff 在业务成功后记录实际对象快照，不读取/缓存 HTTP 请求响应体。
func auditRecordDiff(c *gin.Context, svc *service.AuditService, action, modelName, objectID, objectRepr string, oldObj, newObj interface{}) {
	auditOperation(c, svc, action, modelName, objectID, objectRepr, "", oldObj, newObj)
}

// auditOperation 为非 CRUD 动作补充业务语义；异步任务只记录提交结果。
func auditOperation(c *gin.Context, svc *service.AuditService, action, modelName, objectID, objectRepr, operation string, oldObj, newObj interface{}) {
	if svc == nil {
		return
	}
	oldJSON, newJSON := marshalAudit(oldObj), marshalAudit(newObj)
	parts := []string{c.Request.Method + " " + c.FullPath()}
	if operation != "" {
		parts = append(parts, operation)
	}
	if diff := diffSummary(oldJSON, newJSON); diff != "" {
		parts = append(parts, diff)
	}
	summary := strings.Join(parts, " | ")
	operator := c.GetString("username")
	if operator == "" {
		operator = c.GetString("user_id")
		if operator == "" {
			operator = "system"
		}
	}
	if err := svc.Record(action, modelName, objectID, objectRepr, operator, c.ClientIP(), oldJSON, newJSON, summary); err != nil {
		slog.Default().Error("操作审计写入失败", "action", action, "model", modelName, "object_id", objectID, "error", err)
	}
}

func marshalAudit(v interface{}) datatypes.JSON {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		slog.Default().Error("操作审计快照序列化失败", "error", err)
		return datatypes.JSON(`{"snapshot_unavailable":true}`)
	}
	var value interface{}
	if err := json.Unmarshal(b, &value); err != nil {
		return nil
	}
	redactAudit(value)
	b, err = json.Marshal(value)
	if err != nil {
		return nil
	}
	return datatypes.JSON(b)
}

// diffSummary 比较两份 JSON 对象，返回发生变更的字段名列表（逗号分隔）。
func diffSummary(oldJSON, newJSON datatypes.JSON) string {
	var oldMap, newMap map[string]interface{}
	_ = json.Unmarshal(oldJSON, &oldMap)
	_ = json.Unmarshal(newJSON, &newMap)
	if oldMap == nil && newMap == nil {
		return ""
	}

	changed := make([]string, 0, 8)
	seen := map[string]bool{}
	for k, ov := range oldMap {
		nv, exists := newMap[k]
		if !exists {
			changed = append(changed, k+"(removed)")
		} else if !jsonEqual(ov, nv) {
			changed = append(changed, k)
		}
		seen[k] = true
	}
	for k := range newMap {
		if !seen[k] {
			changed = append(changed, k+"(added)")
		}
	}
	if len(changed) == 0 {
		return ""
	}
	sort.Strings(changed)
	return strings.Join(changed, ", ")
}

func jsonEqual(a, b interface{}) bool {
	ab, errA := json.Marshal(a)
	bb, errB := json.Marshal(b)
	if errA != nil || errB != nil {
		return fmt.Sprint(a) == fmt.Sprint(b)
	}
	return string(ab) == string(bb)
}

// DTO 是第一道脱敏边界；递归清理扩展 JSON，防止 extra_info 等字段夹带凭据。
func redactAudit(value interface{}) {
	switch v := value.(type) {
	case map[string]interface{}:
		sensitiveConfig := false
		if key, ok := v["key"].(string); ok {
			sensitiveConfig = auditSecretKey(key)
		}
		encrypted, _ := v["is_encrypted"].(bool)
		for key, item := range v {
			lower := strings.ToLower(key)
			if text, ok := item.(string); ok && (strings.HasPrefix(strings.TrimSpace(text), "{") || strings.HasPrefix(strings.TrimSpace(text), "[")) {
				var nested interface{}
				if json.Unmarshal([]byte(text), &nested) == nil {
					redactAudit(nested)
					if encoded, err := json.Marshal(nested); err == nil {
						v[key] = string(encoded)
					}
				}
			}
			if auditSecretKey(key) || ((sensitiveConfig || encrypted) && (lower == "value" || lower == "display_value")) {
				v[key] = "[REDACTED]"
				continue
			}
			if raw, ok := item.(string); ok && (lower == "url" || strings.HasSuffix(lower, "_url")) {
				if u, err := url.Parse(raw); err == nil && u.Host != "" {
					u.User = nil
					u.RawQuery = ""
					u.Fragment = ""
					v[key] = u.String()
				}
			}
			redactAudit(item)
		}
	case []interface{}:
		for _, item := range v {
			redactAudit(item)
		}
	}
}

func auditSecretKey(key string) bool {
	key = strings.ToLower(strings.NewReplacer("-", "_", ".", "_").Replace(key))
	switch key {
	case "password", "old_password", "new_password", "password_hash", "secret", "token", "access_token", "refresh_token", "authorization", "cookie", "api_key", "apikey", "accesstoken", "refreshtoken", "clientsecret", "private_key":
		return true
	}
	return (key != "has_secret" && strings.HasSuffix(key, "_secret")) || strings.HasSuffix(key, "_password") || strings.HasSuffix(key, "_token") || strings.HasSuffix(key, "_api_key")
}
