// Package repository — LIKE 查询的搜索词转义。
package repository

import "strings"

// EscapeLike 转义 LIKE 通配符（% _ \），防止用户搜索词干扰匹配范围。
// 使用时 LIKE 模式写作 "?", 传参 "%" + EscapeLike(s) + "%"，
// **并且 SQL 里必须附上 ESCAPE '\'**。
//
// 为什么要显式写：PostgreSQL 的 LIKE 默认把反斜杠当转义符，不写也能工作；
// SQLite 的 LIKE 没有默认转义符，会把 \% \_ 当字面量。于是同一个搜索词
// （典型如 super_admin 这种带下划线的）在 PG 上正常，本地 SQLite 上一条都搜不到。
func EscapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "%", `\%`)
	s = strings.ReplaceAll(s, "_", `\_`)
	return s
}
