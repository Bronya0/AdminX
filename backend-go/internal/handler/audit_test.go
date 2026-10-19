package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"adminx/internal/database"
	"adminx/internal/model"
	"adminx/internal/repository"
	"adminx/internal/service"
)

func auditTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	for _, stmt := range database.SQLiteStatements() {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func auditTestRouter() *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("username", "auditor"); c.Set("user_id", "operator-id"); c.Next() })
	return r
}

func auditRequest(t *testing.T, r *gin.Engine, method, path, body string, code int) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var env struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.Code != code {
		t.Fatalf("%s %s: %s", method, path, w.Body.String())
	}
}

func auditLogs(t *testing.T, db *gorm.DB) []model.AuditLog {
	t.Helper()
	var logs []model.AuditLog
	if err := db.Order("created_at ASC").Find(&logs).Error; err != nil {
		t.Fatal(err)
	}
	return logs
}

func TestConfigAuditSnapshotsAndFailedWrite(t *testing.T) {
	db := auditTestDB(t)
	audit := service.NewAuditService(repository.NewAuditRepo(db))
	h := NewConfigHandler(service.NewConfigService(db, repository.NewConfigRepo(db), nil, nil), audit, slog.New(slog.NewTextHandler(io.Discard, nil)))
	r := auditTestRouter()
	r.POST("/config/", h.Create)
	r.PATCH("/config/:id/", h.Update)
	r.DELETE("/config/:id/", h.Delete)
	auditRequest(t, r, "POST", "/config/", `{"key":"test.name","value":"before"}`, 201)
	logs := auditLogs(t, db)
	if len(logs) != 1 || logs[0].Operator != "auditor" || !strings.Contains(string(logs[0].NewValues), `"value":"before"`) {
		t.Fatalf("missing create audit: %+v", logs)
	}
	path := fmt.Sprintf("/config/%s/", logs[0].ObjectID)
	auditRequest(t, r, "PATCH", path, `{"value":"after"}`, 200)
	logs = auditLogs(t, db)
	if len(logs) != 2 || !strings.Contains(string(logs[1].OldValues), `"value":"before"`) || !strings.Contains(string(logs[1].NewValues), `"value":"after"`) {
		t.Fatalf("missing update snapshots: %+v", logs)
	}
	auditRequest(t, r, "PATCH", path, `{`, 400)
	if len(auditLogs(t, db)) != 2 {
		t.Fatal("failed write produced success audit")
	}
	auditRequest(t, r, "DELETE", path, ``, 204)
	logs = auditLogs(t, db)
	if len(logs) != 3 || logs[2].Action != "delete" || !strings.Contains(string(logs[2].OldValues), `"value":"after"`) {
		t.Fatalf("missing delete snapshot: %+v", logs)
	}
}

func TestMenuRegistrationAuditsCreateAndUpdateExactlyOnce(t *testing.T) {
	db := auditTestDB(t)
	h := NewMenuHandler(service.NewMenuService(db, repository.NewMenuRepo(db)), service.NewAuditService(repository.NewAuditRepo(db)), slog.Default())
	r := auditTestRouter()
	r.POST("/menu/register/", h.Register)
	auditRequest(t, r, "POST", "/menu/register/", `{"code":"test","name":"first","path":"/test"}`, 200)
	auditRequest(t, r, "POST", "/menu/register/", `{"code":"test","name":"second","path":"/test"}`, 200)
	logs := auditLogs(t, db)
	if len(logs) != 2 || logs[0].Action != "create" || logs[1].Action != "update" {
		t.Fatalf("incorrect registration audit: %+v", logs)
	}
	if !strings.Contains(string(logs[1].OldValues), `"name":"first"`) || !strings.Contains(string(logs[1].NewValues), `"name":"second"`) {
		t.Fatal("registration missing before/after snapshots")
	}
}

func TestAuditRedactsNestedSecretsAndEncryptedConfig(t *testing.T) {
	raw := marshalAudit(gin.H{"extra_info": []interface{}{gin.H{"access_token": "TOKEN_LEAK", "secret": "SECRET_LEAK", "has_secret": true}}, "url": "https://user:PASS_LEAK@example.com/hook?token=QUERY_LEAK", "key": "db.password", "value": "VALUE_LEAK", "display_value": "DISPLAY_LEAK"})
	for _, secret := range []string{"TOKEN_LEAK", "SECRET_LEAK", "PASS_LEAK", "QUERY_LEAK", "VALUE_LEAK", "DISPLAY_LEAK"} {
		if strings.Contains(string(raw), secret) {
			t.Fatalf("credential leaked: %s", raw)
		}
	}
	if !strings.Contains(string(raw), `"has_secret":true`) {
		t.Fatal("safe boolean removed")
	}
	raw = marshalAudit(gin.H{"is_encrypted": true, "value": "CIPHERTEXT_LEAK", "display_value": "PLAINTEXT_LEAK"})
	if strings.Contains(string(raw), "LEAK") {
		t.Fatalf("encrypted config leaked: %s", raw)
	}
}

func TestConfigImportAuditsOnlyAffectedObjects(t *testing.T) {
	db := auditTestDB(t)
	svc := service.NewConfigService(db, repository.NewConfigRepo(db), nil, nil)
	existing, err := svc.Create(service.ConfigCreateInput{Key: "test.existing", Value: "before"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(service.ConfigCreateInput{Key: "test.untouched", Value: "keep"}); err != nil {
		t.Fatal(err)
	}
	h := NewConfigHandler(svc, service.NewAuditService(repository.NewAuditRepo(db)), slog.Default())
	r := auditTestRouter()
	r.POST("/config/import/", h.Import)
	body := `{"kind":"adminx.config","version":1,"items":[{"key":"test.existing","value":"after","value_type":"string","group":"default"},{"key":"test.created","value":"new","value_type":"string","group":"default"}]}`
	auditRequest(t, r, "POST", "/config/import/", body, 200)
	logs := auditLogs(t, db)
	if len(logs) != 2 {
		t.Fatalf("want two object audits, got %d", len(logs))
	}
	if logs[0].Action != "update" || logs[0].ObjectID != fmt.Sprint(existing.ID) || !strings.Contains(string(logs[0].OldValues), `"value":"before"`) || !strings.Contains(string(logs[0].NewValues), `"value":"after"`) {
		t.Fatalf("incorrect import update: %+v", logs[0])
	}
	if logs[1].Action != "create" || logs[1].ObjectRepr != "test.created" || !strings.Contains(string(logs[1].NewValues), `"is_active":true`) {
		t.Fatalf("missing persisted import defaults: %+v", logs[1])
	}
	auditRequest(t, r, "POST", "/config/import/", `{"kind":"adminx.config","version":1,"items":[{"key":"test.existing","value":"bad","value_type":"UNKNOWN"}]}`, 400)
	if len(auditLogs(t, db)) != 2 {
		t.Fatal("invalid import logged as success")
	}
}

func TestComponentCommandAndUnregisterAudits(t *testing.T) {
	db := auditTestDB(t)
	svc := service.NewClusterService(db, repository.NewClusterRepo(db))
	h := NewClusterHandler(svc, service.NewAuditService(repository.NewAuditRepo(db)), slog.Default())
	r := auditTestRouter()
	r.POST("/components/register/", h.Register)
	r.POST("/components/:id/set_upgrade/", h.SetUpgrade)
	r.POST("/components/:id/cancel_upgrade/", h.CancelUpgrade)
	r.POST("/components/:id/set_uninstall/", h.SetUninstall)
	r.POST("/components/unregister/", h.Unregister)
	auditRequest(t, r, "POST", "/components/register/", `{"app_label":"example","name":"Example","host":"localhost","version":"1.0"}`, 200)
	logs := auditLogs(t, db)
	if len(logs) != 1 || logs[0].Action != "create" || logs[0].Operator != "component:example" {
		t.Fatalf("incorrect component registration: %+v", logs)
	}
	base := "/components/" + logs[0].ObjectID
	auditRequest(t, r, "POST", base+"/set_upgrade/", `{"upgrade_version":"2.0"}`, 400)
	if len(auditLogs(t, db)) != 1 {
		t.Fatal("rejected command logged as success")
	}
	auditRequest(t, r, "POST", base+"/set_upgrade/", `{"upgrade_version":"2.0","upgrade_url":"https://example.com/update.zip","upgrade_checksum":"`+strings.Repeat("a", 64)+`"}`, 200)
	logs = auditLogs(t, db)
	if len(logs) != 2 || logs[1].Action != "execute" || !strings.Contains(string(logs[1].NewValues), `"upgrade_version":"2.0"`) || !strings.Contains(string(logs[1].OldValues), `"upgrade_version":""`) {
		t.Fatalf("incorrect command snapshots: %+v", logs)
	}
	auditRequest(t, r, "POST", base+"/cancel_upgrade/", `{}`, 200)
	auditRequest(t, r, "POST", base+"/set_uninstall/", `{}`, 200)
	auditRequest(t, r, "POST", "/components/unregister/", `{"app_label":"example"}`, 200)
	logs = auditLogs(t, db)
	last := logs[len(logs)-1]
	if len(logs) != 5 || last.Action != "delete" || last.Operator != "component:example" || !strings.Contains(string(last.OldValues), `"uninstall_pending":true`) {
		t.Fatalf("incorrect unregister audit: %+v", logs)
	}
	auditRequest(t, r, "POST", "/components/unregister/", `{"app_label":"example"}`, 200)
	if len(auditLogs(t, db)) != 5 {
		t.Fatal("idempotent unregister recorded another deletion")
	}
}
