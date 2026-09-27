// Package trigger 定义定时任务的触发配置契约（trigger_type + trigger_config）。
//
// 解析与校验只有这一处实现：scheduler 用它构建 gocron 定义，job service 用它做写入前校验。
// 分开实现过一次的教训：CRUD 只校验"非空"，而调度器侧解析失败仅打一行日志，
// 于是接口返回 201、任务永远不注册，页面上显示"启用"却从不执行。
package trigger

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

// 支持的触发类型。
const (
	KindCron     = "cron"
	KindInterval = "interval"
	KindDate     = "date"
)

// Spec 解析后的触发配置（各类型只填自己那几个字段）。
type Spec struct {
	Kind        string
	CronExpr    string        // cron：表达式
	WithSeconds bool          // cron：是否为 6 段（带秒）表达式
	Interval    time.Duration // interval：固定间隔
	RunAt       time.Time     // date：一次性执行时间（本地时区）
}

// Parse 解析并校验触发配置，返回可用的 Spec。
//
// 兼容的历史/旧前端形态（写入新数据请用文档里的规范形态）：
//   - cron:     {"expr":"*/5 * * * *"}、{"cron":"..."}，或裸表达式 "*/5 * * * *"
//   - interval: {"seconds"|"minutes"|"hours"|"days": n}
//   - date:     {"datetime":"<RFC3339>"} 或 {"run_date":"2006-01-02 15:04:05"}（本地时区）
func Parse(kind, raw string) (Spec, error) {
	switch strings.TrimSpace(kind) {
	case KindCron:
		return parseCron(raw)
	case KindInterval:
		return parseInterval(raw)
	case KindDate:
		return parseDate(raw)
	case "":
		return Spec{}, errors.New("触发类型不能为空（cron/interval/date）")
	default:
		return Spec{}, fmt.Errorf("未知触发类型 %q（仅支持 cron/interval/date）", kind)
	}
}

// parseCron 解析 cron 表达式。
// 5 段按"分 时 日 月 周"解析，6 段按"秒 分 时 日 月 周"解析 ——
// gocron.CronJob 的 withSeconds 必须与段数一致，否则 5 段表达式一律解析失败。
func parseCron(raw string) (Spec, error) {
	expr := ""
	if cfg := asObject(raw); cfg != nil {
		expr = pick(cfg, "expr", "cron")
	} else {
		expr = strings.TrimSpace(raw)
	}
	if expr == "" {
		return Spec{}, errors.New(`cron 触发需要表达式，如 {"expr":"*/5 * * * *"}`)
	}

	withSeconds := len(strings.Fields(expr)) == 6
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	if withSeconds {
		parser = cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	}
	if _, err := parser.Parse(expr); err != nil {
		return Spec{}, fmt.Errorf("cron 表达式非法: %w", err)
	}
	return Spec{Kind: KindCron, CronExpr: expr, WithSeconds: withSeconds}, nil
}

func parseInterval(raw string) (Spec, error) {
	cfg := asObject(raw)
	if cfg == nil {
		return Spec{}, errors.New(`interval 触发需要 JSON 对象，如 {"minutes":5}`)
	}
	units := []struct {
		key  string
		unit time.Duration
	}{
		{"seconds", time.Second},
		{"minutes", time.Minute},
		{"hours", time.Hour},
		{"days", 24 * time.Hour},
	}
	for _, u := range units {
		if n := asInt(cfg[u.key]); n > 0 {
			return Spec{Kind: KindInterval, Interval: time.Duration(n) * u.unit}, nil
		}
	}
	return Spec{}, errors.New(`interval 触发需要 seconds/minutes/hours/days 之一（正整数），如 {"minutes":5}`)
}

// dateLayouts 兼容旧前端提交的 "2006-01-02 15:04:05"（无时区，按本地时区解释）。
var dateLayouts = []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05"}

func parseDate(raw string) (Spec, error) {
	dt := ""
	if cfg := asObject(raw); cfg != nil {
		dt = pick(cfg, "datetime", "run_date")
	} else {
		dt = strings.TrimSpace(raw)
	}
	if dt == "" {
		return Spec{}, errors.New(`date 触发需要执行时间，如 {"datetime":"2026-01-02T15:04:05+08:00"}`)
	}
	for _, layout := range dateLayouts {
		if t, err := time.ParseInLocation(layout, dt, time.Local); err == nil {
			// 时间已过的一次性任务会被 gocron 静默丢弃（不注册、不报错），
			// 在写入前就拒绝，避免又出现"接口成功但任务永不执行"。
			if t.Before(time.Now()) {
				return Spec{}, fmt.Errorf("date 执行时间已过去（%s），一次性任务需要未来的时间", t.Format("2006-01-02 15:04:05"))
			}
			return Spec{Kind: KindDate, RunAt: t}, nil
		}
	}
	return Spec{}, fmt.Errorf("date 时间格式非法 %q（需 RFC3339，或 2006-01-02 15:04:05 本地时间）", dt)
}

// asObject 把 trigger_config 解析为 JSON 对象；不是 JSON 对象（裸串/空/非法 JSON）时返回 nil。
func asObject(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "{") {
		return nil
	}
	var cfg map[string]any
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil
	}
	return cfg
}

// pick 按顺序取第一个非空字符串字段。
func pick(cfg map[string]any, keys ...string) string {
	for _, k := range keys {
		if s, ok := cfg[k].(string); ok {
			if s = strings.TrimSpace(s); s != "" {
				return s
			}
		}
	}
	return ""
}

// asInt JSON 数字在 map[string]any 里是 float64。
func asInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	}
	return 0
}
