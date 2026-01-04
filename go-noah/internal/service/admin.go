package service

import (
	"context"
	"errors"
	"go-noah/api"
	"go-noah/internal/model"
	"go-noah/internal/repository"
	"go-noah/pkg/global"
	"strings"
	"time"

	"github.com/duke-git/lancet/v2/convertor"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AdminService 管理员业务逻辑层
type AdminService struct{}

var AdminServiceApp = new(AdminService)

// getAdminRepo 获取 AdminRepository（在方法内部创建）
func (s *AdminService) getAdminRepo() *repository.AdminRepository {
	return repository.NewAdminRepository(global.Repo)
}

func (s *AdminService) GetAdminUser(ctx context.Context, uid uint) (*api.GetAdminUserResponseData, error) {
	repo := s.getAdminRepo()
	user, err := repo.GetAdminUser(ctx, uid)
	if err != nil {
		return nil, err
	}
	roles, _ := repo.GetUserRoles(ctx, uid)

	return &api.GetAdminUserResponseData{
		Email:     user.Email,
		ID:        user.ID,
		Username:  user.Username,
		Nickname:  user.Nickname,
		Phone:     user.Phone,
		Roles:     roles,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (s *AdminService) Login(ctx context.Context, req *api.LoginRequest) (string, error) {
	repo := s.getAdminRepo()
	user, err := repo.GetAdminUserByUsername(ctx, req.Username)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", api.ErrUnauthorized
		}
		return "", api.ErrInternalServerError
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return "", err
	}
	token, err := global.JWT.GenToken(user.ID, time.Now().Add(time.Hour*24*90))
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *AdminService) GetAdminUsers(ctx context.Context, req *api.GetAdminUsersRequest) (*api.GetAdminUsersResponseData, error) {
	repo := s.getAdminRepo()
	list, total, err := repo.GetAdminUsers(ctx, req)
	if err != nil {
		return nil, err
	}
	data := &api.GetAdminUsersResponseData{
		List:  make([]api.AdminUserDataItem, 0),
		Total: total,
	}
	for _, user := range list {
		roles, err := repo.GetUserRoles(ctx, user.ID)
		if err != nil {
			global.Logger.Error("GetUserRoles error", zap.Error(err))
			continue
		}
		data.List = append(data.List, api.AdminUserDataItem{
			Email:     user.Email,
			ID:        user.ID,
			Nickname:  user.Nickname,
			Username:  user.Username,
			Phone:     user.Phone,
			Roles:     roles,
			CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: user.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return data, nil
}

func (s *AdminService) AdminUserUpdate(ctx context.Context, req *api.AdminUserUpdateRequest) error {
	repo := s.getAdminRepo()
	old, _ := repo.GetAdminUser(ctx, req.ID)
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		req.Password = string(hash)
	} else {
		req.Password = old.Password
	}
	err := repo.UpdateUserRoles(ctx, req.ID, req.Roles)
	if err != nil {
		return err
	}
	return repo.AdminUserUpdate(ctx, &model.AdminUser{
		Model: gorm.Model{
			ID: req.ID,
		},
		Email:    req.Email,
		Nickname: req.Nickname,
		Phone:    req.Phone,
		Username: req.Username,
	})

}

func (s *AdminService) AdminUserCreate(ctx context.Context, req *api.AdminUserCreateRequest) error {
	repo := s.getAdminRepo()
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	req.Password = string(hash)
	err = repo.AdminUserCreate(ctx, &model.AdminUser{
		Email:    req.Email,
		Nickname: req.Nickname,
		Phone:    req.Phone,
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		return err
	}
	user, err := repo.GetAdminUserByUsername(ctx, req.Username)
	if err != nil {
		return err
	}
	err = repo.UpdateUserRoles(ctx, user.ID, req.Roles)
	if err != nil {
		return err
	}
	return err

}

func (s *AdminService) AdminUserDelete(ctx context.Context, id uint) error {
	repo := s.getAdminRepo()
	// 删除用户角色
	err := repo.DeleteUserRoles(ctx, id)
	if err != nil {
		return err
	}
	return repo.AdminUserDelete(ctx, id)
}

func (s *AdminService) UpdateRolePermission(ctx context.Context, req *api.UpdateRolePermissionRequest) error {
	repo := s.getAdminRepo()
	permissions := map[string]struct{}{}
	for _, v := range req.List {
		perm := strings.Split(v, model.PermSep)
		if len(perm) == 2 {
			permissions[v] = struct{}{}
		}

	}
	return repo.UpdateRolePermission(ctx, req.Role, permissions)
}

func (s *AdminService) GetApis(ctx context.Context, req *api.GetApisRequest) (*api.GetApisResponseData, error) {
	repo := s.getAdminRepo()
	list, total, err := repo.GetApis(ctx, req)
	if err != nil {
		return nil, err
	}
	groups, err := repo.GetApiGroups(ctx)
	if err != nil {
		return nil, err
	}
	data := &api.GetApisResponseData{
		List:   make([]api.ApiDataItem, 0),
		Total:  total,
		Groups: groups,
	}
	for _, item := range list {
		data.List = append(data.List, api.ApiDataItem{
			CreatedAt: item.CreatedAt.Format("2006-01-02 15:04:05"),
			Group:     item.Group,
			ID:        item.ID,
			Method:    item.Method,
			Name:      item.Name,
			Path:      item.Path,
			UpdatedAt: item.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return data, nil
}

func (s *AdminService) ApiUpdate(ctx context.Context, req *api.ApiUpdateRequest) error {
	repo := s.getAdminRepo()
	return repo.ApiUpdate(ctx, &model.Api{
		Group:  req.Group,
		Method: req.Method,
		Name:   req.Name,
		Path:   req.Path,
		Model: gorm.Model{
			ID: req.ID,
		},
	})
}

func (s *AdminService) ApiCreate(ctx context.Context, req *api.ApiCreateRequest) error {
	repo := s.getAdminRepo()
	return repo.ApiCreate(ctx, &model.Api{
		Group:  req.Group,
		Method: req.Method,
		Name:   req.Name,
		Path:   req.Path,
	})
}

func (s *AdminService) ApiDelete(ctx context.Context, id uint) error {
	repo := s.getAdminRepo()
	return repo.ApiDelete(ctx, id)
}

func (s *AdminService) GetUserPermissions(ctx context.Context, uid uint) (*api.GetUserPermissionsData, error) {
	repo := s.getAdminRepo()
	data := &api.GetUserPermissionsData{
		List: []string{},
	}
	list, err := repo.GetUserPermissions(ctx, uid)
	if err != nil {
		return nil, err
	}
	for _, v := range list {
		if len(v) == 3 {
			data.List = append(data.List, strings.Join([]string{v[1], v[2]}, model.PermSep))
		}
	}
	return data, nil
}
func (s *AdminService) GetRolePermissions(ctx context.Context, role string) (*api.GetRolePermissionsData, error) {
	repo := s.getAdminRepo()
	data := &api.GetRolePermissionsData{
		List: []string{},
	}
	list, err := repo.GetRolePermissions(ctx, role)
	if err != nil {
		return nil, err
	}
	for _, v := range list {
		if len(v) == 3 {
			data.List = append(data.List, strings.Join([]string{v[1], v[2]}, model.PermSep))
		}
	}
	return data, nil
}

func (s *AdminService) MenuUpdate(ctx context.Context, req *api.MenuUpdateRequest) error {
	repo := s.getAdminRepo()
	return repo.MenuUpdate(ctx, &model.Menu{
		Component:  req.Component,
		Icon:       req.Icon,
		KeepAlive:  req.KeepAlive,
		HideInMenu: req.HideInMenu,
		Locale:     req.Locale,
		Weight:     req.Weight,
		Name:       req.Name,
		ParentID:   req.ParentID,
		Path:       req.Path,
		Redirect:   req.Redirect,
		Title:      req.Title,
		URL:        req.URL,
		Model: gorm.Model{
			ID: req.ID,
		},
	})
}

func (s *AdminService) MenuCreate(ctx context.Context, req *api.MenuCreateRequest) error {
	repo := s.getAdminRepo()
	return repo.MenuCreate(ctx, &model.Menu{
		Component:  req.Component,
		Icon:       req.Icon,
		KeepAlive:  req.KeepAlive,
		HideInMenu: req.HideInMenu,
		Locale:     req.Locale,
		Weight:     req.Weight,
		Name:       req.Name,
		ParentID:   req.ParentID,
		Path:       req.Path,
		Redirect:   req.Redirect,
		Title:      req.Title,
		URL:        req.URL,
	})
}

func (s *AdminService) MenuDelete(ctx context.Context, id uint) error {
	repo := s.getAdminRepo()
	return repo.MenuDelete(ctx, id)
}

func (s *AdminService) GetMenus(ctx context.Context, uid uint) (*api.GetMenuResponseData, error) {
	repo := s.getAdminRepo()
	menuList, err := repo.GetMenuList(ctx)
	if err != nil {
		global.Logger.WithContext(ctx).Error("GetMenuList error", zap.Error(err))
		return nil, err
	}
	data := &api.GetMenuResponseData{
		List: make([]api.MenuDataItem, 0),
	}
	// 获取权限的菜单
	permissions, err := repo.GetUserPermissions(ctx, uid)
	if err != nil {
		return nil, err
	}
	menuPermMap := map[string]struct{}{}
	for _, permission := range permissions {
		// 防呆设置，超管可以看到所有菜单
		if convertor.ToString(uid) == model.AdminUserID {
			menuPermMap[strings.TrimPrefix(permission[1], model.MenuResourcePrefix)] = struct{}{}
		} else {
			if len(permission) == 3 && strings.HasPrefix(permission[1], model.MenuResourcePrefix) {
				menuPermMap[strings.TrimPrefix(permission[1], model.MenuResourcePrefix)] = struct{}{}
			}
		}
	}

	for _, menu := range menuList {
		if _, ok := menuPermMap[menu.Path]; ok {
			data.List = append(data.List, api.MenuDataItem{
				ID:         menu.ID,
				Name:       menu.Name,
				Title:      menu.Title,
				Path:       menu.Path,
				Component:  menu.Component,
				Redirect:   menu.Redirect,
				KeepAlive:  menu.KeepAlive,
				HideInMenu: menu.HideInMenu,
				Locale:     menu.Locale,
				Weight:     menu.Weight,
				Icon:       menu.Icon,
				ParentID:   menu.ParentID,
				UpdatedAt:  menu.UpdatedAt.Format("2006-01-02 15:04:05"),
				URL:        menu.URL,
			})
		}
	}
	return data, nil
}
func (s *AdminService) GetAdminMenus(ctx context.Context) (*api.GetMenuResponseData, error) {
	repo := s.getAdminRepo()
	menuList, err := repo.GetMenuList(ctx)
	if err != nil {
		global.Logger.WithContext(ctx).Error("GetMenuList error", zap.Error(err))
		return nil, err
	}
	data := &api.GetMenuResponseData{
		List: make([]api.MenuDataItem, 0),
	}
	for _, menu := range menuList {
		data.List = append(data.List, api.MenuDataItem{
			ID:         menu.ID,
			Name:       menu.Name,
			Title:      menu.Title,
			Path:       menu.Path,
			Component:  menu.Component,
			Redirect:   menu.Redirect,
			KeepAlive:  menu.KeepAlive,
			HideInMenu: menu.HideInMenu,
			Locale:     menu.Locale,
			Weight:     menu.Weight,
			Icon:       menu.Icon,
			ParentID:   menu.ParentID,
			UpdatedAt:  menu.UpdatedAt.Format("2006-01-02 15:04:05"),
			URL:        menu.URL,
		})
	}
	return data, nil
}

func (s *AdminService) RoleUpdate(ctx context.Context, req *api.RoleUpdateRequest) error {
	repo := s.getAdminRepo()
	return repo.RoleUpdate(ctx, &model.Role{
		Name: req.Name,
		Sid:  req.Sid,
		Model: gorm.Model{
			ID: req.ID,
		},
	})
}

func (s *AdminService) RoleCreate(ctx context.Context, req *api.RoleCreateRequest) error {
	repo := s.getAdminRepo()
	_, err := repo.GetRoleBySid(ctx, req.Sid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return repo.RoleCreate(ctx, &model.Role{
				Name: req.Name,
				Sid:  req.Sid,
			})
		} else {
			return err
		}
	}
	return nil
}

func (s *AdminService) RoleDelete(ctx context.Context, id uint) error {
	repo := s.getAdminRepo()
	old, err := repo.GetRole(ctx, id)
	if err != nil {
		return err
	}
	if err := repo.CasbinRoleDelete(ctx, old.Sid); err != nil {
		return err
	}
	return repo.RoleDelete(ctx, id)
}

func (s *AdminService) GetRoles(ctx context.Context, req *api.GetRoleListRequest) (*api.GetRolesResponseData, error) {
	repo := s.getAdminRepo()
	list, total, err := repo.GetRoles(ctx, req)
	if err != nil {
		return nil, err
	}
	data := &api.GetRolesResponseData{
		List:  make([]api.RoleDataItem, 0),
		Total: total,
	}
	for _, role := range list {
		data.List = append(data.List, api.RoleDataItem{
			ID:        role.ID,
			Name:      role.Name,
			Sid:       role.Sid,
			UpdatedAt: role.UpdatedAt.Format("2006-01-02 15:04:05"),
			CreatedAt: role.CreatedAt.Format("2006-01-02 15:04:05"),
		})

	}
	return data, nil
}
