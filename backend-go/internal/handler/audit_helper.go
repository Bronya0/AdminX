package handler

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"

	"adminx/internal/service"
)

// auditRecord 记录一条操作审计（create/update/delete）。
// 从 gin context 提取操作人与 IP；审计失败不影响主流程（只留痕，不阻断）。
func auditRecord(c *gin.Context, svc *service.AuditService, action, modelName, objectID, objectRepr string) {
	if svc == nil {
		return
	}
	operator, _ := c.Get("username")
	op, _ := operator.(string)
	if err := svc.Record(action, modelName, objectID, objectRepr, op, c.ClientIP(), nil, nil, ""); err != nil {
		// 审计失败仅记录，不阻断业务
		return
	}
}

// auditRecordDiff 记录带前后快照的审计（update/delete）。
// oldObj/newObj 经 JSON 序列化存储（model 敏感字段需 json:"-"，密码字段均满足），
// 并生成变更字段名摘要写入 diff_summary。读取失败不阻断审计与业务。
func auditRecordDiff(c *gin.Context, svc *service.AuditService, action, modelName, objectID, objectRepr string, oldObj, newObj interface{}) {
	if svc == nil {
		return
	}
	operator, _ := c.Get("username")
	op, _ := operator.(string)

	oldJSON := marshalAudit(oldObj)
	newJSON := marshalAudit(newObj)
	diff := diffSummary(oldJSON, newJSON)

	if err := svc.Record(action, modelName, objectID, objectRepr, op, c.ClientIP(), oldJSON, newJSON, diff); err != nil {
		return
	}
}

func marshalAudit(v interface{}) datatypes.JSON {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
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
