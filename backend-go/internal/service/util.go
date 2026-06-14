package service

import (
	"encoding/json"

	"gorm.io/datatypes"
)

// datatypesJSON 是 datatypes.JSON 的别名，简化 service 层引用。
type datatypesJSON = datatypes.JSON

// jsonMarshal 封装 json.Marshal。
func jsonMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// mustJSON 将字符串切片转为 datatypes.JSON，失败时返回空数组 []。
// 用于菜单的 allowed_paths 字段。
func mustJSON(items []string) datatypes.JSON {
	if len(items) == 0 {
		return datatypes.JSON([]byte("[]"))
	}
	b, err := json.Marshal(items)
	if err != nil {
		return datatypes.JSON([]byte("[]"))
	}
	return datatypes.JSON(b)
}
