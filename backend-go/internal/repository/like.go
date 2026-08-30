// Package repository — LIKE 查询的搜索词转义。
package repository

import "strings"

// EscapeLike 转义 LIKE 通配符（% _ \），防止用户搜索词干扰匹配范围。
// 使用时 LIKE 模式写作 "?", 传参 "%" + EscapeLike(s) + "%"。
// Postgres 默认 LIKE 不支持 ESCAPE 语法变化：反斜杠是默认转义符，无需附加 ESCAPE 子句。
func EscapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "%", `\%`)
	s = strings.ReplaceAll(s, "_", `\_`)
	return s
}
