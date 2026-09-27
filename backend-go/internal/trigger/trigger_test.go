package trigger

import (
	"strings"
	"testing"
	"time"
)

// 5 段 cron（分 时 日 月 周）是前端预设与文档里的常见写法，必须能解析。
func TestParseCron_FiveFields(t *testing.T) {
	spec, err := Parse(KindCron, `{"expr":"*/5 * * * *"}`)
	if err != nil {
		t.Fatalf("5 段 cron 应解析成功: %v", err)
	}
	if spec.CronExpr != "*/5 * * * *" || spec.WithSeconds {
		t.Errorf("解析结果不对: %+v", spec)
	}

	// 裸表达式（旧前端/历史数据）
	spec, err = Parse(KindCron, "0 2 * * *")
	if err != nil {
		t.Fatalf("裸 cron 表达式应兼容: %v", err)
	}
	if spec.CronExpr != "0 2 * * *" || spec.WithSeconds {
		t.Errorf("裸表达式解析结果不对: %+v", spec)
	}

	// {"cron": ...} 别名
	if _, err := Parse(KindCron, `{"cron":"0 12 * * *"}`); err != nil {
		t.Errorf(`{"cron":...} 应兼容: %v`, err)
	}
}

func TestParseCron_SixFields(t *testing.T) {
	spec, err := Parse(KindCron, `{"expr":"*/30 * * * * *"}`)
	if err != nil {
		t.Fatalf("6 段 cron 应解析成功: %v", err)
	}
	// 段数必须与 gocron.CronJob 的 withSeconds 一致，否则注册必然失败
	if !spec.WithSeconds {
		t.Errorf("6 段表达式应标记 WithSeconds: %+v", spec)
	}
}

func TestParseCron_Invalid(t *testing.T) {
	cases := []string{
		`{"expr":"not a cron"}`,
		`{"expr":"*/5 * *"}`,   // 字段数不足
		`{"expr":"99 * * * *"}`, // 分钟越界
		``,
		`{"expr":""}`,
	}
	for _, raw := range cases {
		if _, err := Parse(KindCron, raw); err == nil {
			t.Errorf("非法 cron 配置应报错: %q", raw)
		}
	}
}

func TestParseInterval(t *testing.T) {
	cases := []struct {
		raw  string
		want time.Duration
	}{
		{`{"seconds":30}`, 30 * time.Second},
		{`{"minutes":5}`, 5 * time.Minute},
		{`{"hours":1}`, time.Hour},
		{`{"days":2}`, 48 * time.Hour},
	}
	for _, c := range cases {
		spec, err := Parse(KindInterval, c.raw)
		if err != nil {
			t.Fatalf("interval %s 应解析成功: %v", c.raw, err)
		}
		if spec.Interval != c.want {
			t.Errorf("interval %s = %s, want %s", c.raw, spec.Interval, c.want)
		}
	}

	for _, raw := range []string{``, `{}`, `{"minutes":0}`, `{"minutes":-1}`, `5`, `{"minutes":"5"}`} {
		if _, err := Parse(KindInterval, raw); err == nil {
			t.Errorf("非法 interval 配置应报错: %q", raw)
		}
	}
}

func TestParseDate(t *testing.T) {
	// 规范形态：RFC3339（带时区）
	spec, err := Parse(KindDate, `{"datetime":"2099-01-02T15:04:05+08:00"}`)
	if err != nil {
		t.Fatalf("RFC3339 应解析成功: %v", err)
	}
	want := time.Date(2099, 1, 2, 15, 4, 5, 0, time.FixedZone("", 8*3600))
	if !spec.RunAt.Equal(want) {
		t.Errorf("RunAt = %s, want %s", spec.RunAt, want)
	}

	// 旧前端形态：无时区，按本地时区解释
	spec, err = Parse(KindDate, `{"datetime":"2099-01-02 15:04:05"}`)
	if err != nil {
		t.Fatalf("本地时间格式应兼容: %v", err)
	}
	if !spec.RunAt.Equal(time.Date(2099, 1, 2, 15, 4, 5, 0, time.Local)) {
		t.Errorf("本地时间解析结果不对: %s", spec.RunAt)
	}

	// run_date 别名
	if _, err := Parse(KindDate, `{"run_date":"2099-01-02 15:04:05"}`); err != nil {
		t.Errorf("run_date 应兼容: %v", err)
	}

	for _, raw := range []string{``, `{}`, `{"datetime":"02/01/2026"}`, `{"datetime":"2020-01-01 00:00:00"}`} {
		if _, err := Parse(KindDate, raw); err == nil {
			t.Errorf("非法 date 配置应报错: %q", raw)
		}
	}
}

func TestParseUnknownKind(t *testing.T) {
	for _, kind := range []string{"", "weekly", "INTERVAL"} {
		if _, err := Parse(kind, `{"minutes":5}`); err == nil {
			t.Errorf("未知触发类型应报错: %q", kind)
		}
	}
	// 错误信息要能指导用户改正
	_, err := Parse("weekly", ``)
	if err == nil || !strings.Contains(err.Error(), "cron/interval/date") {
		t.Errorf("错误信息应列出支持的类型: %v", err)
	}
}
