package service

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	apperr "adminx/pkg/errors"
	"adminx/pkg/crypto"

	"adminx/internal/model"
	"adminx/internal/repository"
)

// defaultHomePage 未指定首页时的默认落点（系统内置首页）。
const defaultHomePage = "/dashboard"

// UserService 用户业务逻辑。
type UserService struct {
	db       *gorm.DB
	userRepo *repository.UserRepo
}

func NewUserService(db *gorm.DB, userRepo *repository.UserRepo) *UserService {
	return &UserService{db: db, userRepo: userRepo}
}

// List 分页查询用户。
func (s *UserService) List(offset, limit int, search, role string, isActive *bool) ([]model.User, int64, error) {
	return s.userRepo.List(offset, limit, search, role, isActive)
}

// GetByID 查询单个用户。
func (s *UserService) GetByID(id string) (*model.User, error) {
	u, err := s.userRepo.FindByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrNotFound
		}
		return nil, apperr.ErrInternal
	}
	return u, nil
}

// CreateInput 创建用户入参。
type CreateInput struct {
	Username  string   `json:"username" binding:"required,min=2,max=150"`
	Password  string   `json:"password" binding:"required,min=6"`
	Email     string   `json:"email"`
	Phone     string   `json:"phone"`
	Avatar    string   `json:"avatar"`
	Desc      string   `json:"desc"`
	HomePage  string   `json:"home_page"`
	IsActive  *bool    `json:"is_active"`
	IsSuperuser bool   `json:"is_superuser"`
	Roles     []string `json:"roles"` // role IDs
}

// Create 创建用户（含密码哈希 + 角色关联）。
// 建用户与绑角色包在同一事务内：AssignRoles 失败时整体回滚，避免留下无角色的半份用户。
func (s *UserService) Create(in CreateInput) (*model.User, error) {
	// 检查用户名唯一（含软删除账号：username 唯一索引不区分软删除，
	// 只查未删除记录会在 INSERT 时撞索引、接口报 500）
	if _, err := s.userRepo.FindAnyByUsername(in.Username); err == nil {
		return nil, apperr.New(409, "用户名已存在（可能属于已删除账号，请先恢复或更换用户名）")
	}

	// 非超管必须有角色：没角色的账号能登录，但 RBAC 查不到任何菜单，进去是一片空白
	if !in.IsSuperuser && len(in.Roles) == 0 {
		return nil, apperr.New(400, "请至少为用户选择一个角色")
	}

	// 首页决定登录后的落点：没填就给系统默认首页，避免账号落点随菜单顺序漂移
	if strings.TrimSpace(in.HomePage) == "" {
		in.HomePage = defaultHomePage
	}

	// 密码策略校验（管理员创建用户同样受策略约束）
	if err := validatePasswordPolicy(s.db, in.Password); err != nil {
		return nil, err
	}

	// 密码哈希
	hashed, err := crypto.HashPassword(in.Password)
	if err != nil {
		return nil, apperr.ErrInternal
	}

	isActive := true
	if in.IsActive != nil {
		isActive = *in.IsActive
	}

	user := &model.User{
		Username:    in.Username,
		Password:    hashed,
		Email:       in.Email,
		Phone:       in.Phone,
		Avatar:      in.Avatar,
		Desc:        in.Desc,
		HomePage:    in.HomePage,
		IsActive:    isActive,
		IsSuperuser: in.IsSuperuser,
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		txRepo := s.userRepo.WithTx(tx)
		if err := txRepo.Create(user); err != nil {
			if isDupKey(err) {
				return apperr.New(409, "用户名已存在")
			}
			return apperr.ErrInternal
		}

		// 角色关联（同事务：失败整体回滚）
		if len(in.Roles) > 0 {
			if err := txRepo.AssignRoles(user.ID, in.Roles); err != nil {
				return apperr.ErrInternal
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// 重新加载（含角色）
	return s.userRepo.FindByID(user.ID)
}

// UpdateInput 更新用户入参（指针字段区分"未传"和"清空"）。
type UpdateInput struct {
	Email       *string  `json:"email"`
	Phone       *string  `json:"phone"`
	Avatar      *string  `json:"avatar"`
	Desc       *string  `json:"desc"`
	HomePage   *string  `json:"home_page"`
	IsActive   *bool    `json:"is_active"`
	IsSuperuser *bool   `json:"is_superuser"`
	Password    string   `json:"password"` // 非空则更新密码
	Roles       []string `json:"roles"`
}

// Update 更新用户。
func (s *UserService) Update(id string, in UpdateInput) (*model.User, error) {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrNotFound
		}
		return nil, apperr.ErrInternal
	}

	// 非超管账号不允许清空角色：角色为空的账号登录后看不到任何菜单。
	// 超管不在此列，它的权限来自 is_superuser，roles 本来就会被忽略。
	if in.Roles != nil && len(in.Roles) == 0 && !user.IsSuperuser {
		return nil, apperr.New(400, "请至少为用户选择一个角色")
	}

	// 首页传空则落回默认值：清空后不应留下一个登录无确定落点的账号
	if in.HomePage != nil && strings.TrimSpace(*in.HomePage) == "" {
		page := defaultHomePage
		in.HomePage = &page
	}

	// 收集本次变更列（只更新传了的字段）。
	// 不做“读整行改完再 Save 全行”：Save 会把 last_activity/last_login 等并发单列更新覆盖回去。
	cols := map[string]interface{}{}
	if in.Email != nil {
		cols["email"] = *in.Email
	}
	if in.Phone != nil {
		cols["phone"] = *in.Phone
	}
	if in.Avatar != nil {
		cols["avatar"] = *in.Avatar
	}
	if in.Desc != nil {
		cols["desc"] = *in.Desc
	}
	if in.HomePage != nil {
		cols["home_page"] = *in.HomePage
	}
	if in.IsActive != nil {
		cols["is_active"] = *in.IsActive
	}
	if in.IsSuperuser != nil {
		cols["is_superuser"] = *in.IsSuperuser
	}

	// 密码更新（重置密码同样受策略约束）
	passwordChanged := false
	if in.Password != "" {
		if err := validatePasswordPolicy(s.db, in.Password); err != nil {
			return nil, err
		}
		hashed, err := crypto.HashPassword(in.Password)
		if err != nil {
			return nil, apperr.ErrInternal
		}
		cols["password"] = hashed
		passwordChanged = true
	}

	// 本次变更会让一个启用中的超管失去身份或被停用：交给事务在写完之后确认还有别的超管。
	// 必须是“写完再查”，否则两个超管同时互相降级时各自都能看到对方还在而双双放行。
	losingSuperuser := user.IsSuperuser && user.IsActive &&
		((in.IsSuperuser != nil && !*in.IsSuperuser) || (in.IsActive != nil && !*in.IsActive))

	// 重置密码即吊销该用户全部旧 token（与 policy 改密同标准）：
	// 否则被盗 token 在管理员重置密码后仍可继续使用。
	// 密码写入与 last_logout 更新同事务，避免只改一半。
	if err := s.userRepo.SaveAndMaybeRevoke(user.ID, cols, passwordChanged, losingSuperuser); err != nil {
		if errors.Is(err, repository.ErrLastSuperuser) {
			return nil, apperr.New(403, "系统至少保留一个启用状态的超级管理员，不能降级或停用最后一个超管")
		}
		return nil, apperr.ErrInternal
	}

	// 角色关联（仅当传入 roles 字段时更新）。
	// 超管账号的角色只影响界面展示，权限来自 is_superuser，改了没有任何效果；
	// 这里静默忽略而不是报错：前端编辑超管时会一并提交 roles，报错会让改个描述也保存不了。
	if in.Roles != nil && !user.IsSuperuser {
		if err := s.userRepo.AssignRoles(user.ID, in.Roles); err != nil {
			return nil, apperr.ErrInternal
		}
	}

	return s.userRepo.FindByID(user.ID)
}

// superAdminRoleName 拥有全量权限的系统角色名；对其授予/撤销仅限超级管理员。
// 这是权限体系的提权闸门：否则"用户管理"权限持有者可给自己授 super_admin 完成提权。
const superAdminRoleName = "super_admin"

// CheckRoleAssignment 校验角色赋权边界（Create/Update 保存前调用）：
//   - 调用方为超级管理员：放行；
//   - 请求的角色列表包含 super_admin：拒绝；
//   - 目标用户当前持有 super_admin（任何角色变更都可能撤掉它）：拒绝。
func (s *UserService) CheckRoleAssignment(roleRefs []string, targetUserID string, callerIsSuperuser bool) error {
	if callerIsSuperuser {
		return nil
	}
	if len(roleRefs) > 0 {
		// 只在"请求的角色里确实包含 super_admin"时拒绝。
		// 注意不能只写 Where(name = super_admin)：那样统计的是库里叫 super_admin 的角色条数
		// （恒为 1），任何非超管提交 roles 都会被 403，连普通角色也授不了。
		ids, names := repository.SplitRoleRefs(roleRefs)
		q := s.db.Model(&model.Role{}).Where("name = ?", superAdminRoleName)
		switch {
		case len(ids) > 0 && len(names) > 0:
			q = q.Where("id IN ? OR name IN ?", ids, names)
		case len(ids) > 0:
			q = q.Where("id IN ?", ids)
		default:
			q = q.Where("name IN ?", names)
		}
		var n int64
		if err := q.Count(&n).Error; err != nil {
			return apperr.ErrInternal
		}
		if n > 0 {
			return apperr.New(403, "super_admin 角色的授予仅限超级管理员操作")
		}
	}
	if targetUserID != "" {
		var held int64
		if err := s.db.Table("user_roles").
			Joins("JOIN roles ON roles.id = user_roles.role_id").
			Where("user_roles.user_id = ? AND roles.name = ?", targetUserID, superAdminRoleName).
			Count(&held).Error; err != nil {
			return apperr.ErrInternal
		}
		if held > 0 {
			return apperr.New(403, "super_admin 用户的角色变更仅限超级管理员操作")
		}
	}
	return nil
}

// Logout 强制用户下线：更新 last_logout，该用户已签发的 token 立即失效
// （JWTAuth、Refresh、WS 三处都按 jwt.RevokedByLogout 判定，与改密吊销共用一套机制）。
func (s *UserService) Logout(id string) error {
	if _, err := s.GetByID(id); err != nil {
		return err
	}
	if err := s.userRepo.UpdateLogout(id); err != nil {
		return apperr.ErrInternal
	}
	return nil
}

// Delete 软删除用户。
func (s *UserService) Delete(id string) error {
	user, err := s.GetByID(id)
	if err != nil {
		return err
	}
	guard := user.IsSuperuser && user.IsActive
	if err := s.userRepo.SoftDelete(id, guard); err != nil {
		if errors.Is(err, repository.ErrLastSuperuser) {
			return apperr.New(403, "系统至少保留一个启用状态的超级管理员，不能删除最后一个超管")
		}
		return apperr.ErrInternal
	}
	return nil
}
