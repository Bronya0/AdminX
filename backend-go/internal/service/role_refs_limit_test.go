package service

import (
	"errors"
	"strconv"
	"testing"

	apperr "adminx/pkg/errors"

	"adminx/internal/repository"
)

// roles 数组必须有长度上限：几万个元素会拼出超长 SQL（PG 参数上限 65535 = 直接 500）。
// 校验点遍布各 service 入口，这里逐个确认，避免以后新增入口时漏掉。
func TestRoleRefsLengthLimit(t *testing.T) {
	db := setupTestDB(t)
	roleSvc := NewRoleService(db, repository.NewRoleRepo(db))
	userSvc := NewUserService(db, repository.NewUserRepo(db))

	refs := make([]string, maxRoleRefs+1)
	for i := range refs {
		refs[i] = "role-" + strconv.Itoa(i)
	}

	want400 := func(name string, err error) {
		t.Helper()
		var appErr *apperr.AppError
		if !errors.As(err, &appErr) || appErr.Code != 400 {
			t.Fatalf("%s: 期望 400 参数错误，实际 %v", name, err)
		}
	}

	_, err := roleSvc.AccessibleMenus(refs)
	want400("AccessibleMenus", err)

	_, err = userSvc.Create(CreateInput{Username: "toolarge", Password: "Pass123!", Roles: refs})
	want400("Create", err)

	_, err = userSvc.Update("00000000-0000-0000-0000-000000000000", UpdateInput{Roles: refs})
	want400("Update", err)

	// 超管也要受限：长度校验在 superuser 放行分支之前
	want400("CheckRoleAssignment(superuser)", userSvc.CheckRoleAssignment(refs, "", true))

	// 上限之内不受影响
	if _, err := roleSvc.AccessibleMenus([]string{superAdminRoleName}); err != nil {
		t.Fatalf("正常入参被拒绝: %v", err)
	}
}
