package service

import (
	"context"
	"errors"
	"go-noah/api"
	"go-noah/internal/model"
	"go-noah/internal/repository"
	"go-noah/pkg/global"
	"go-noah/pkg/ldap"
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
	// 直接创建 Repository，避免循环导入
	repo := repository.NewRepository(global.Logger, global.DB, global.Enforcer)
	return repository.NewAdminRepository(repo)
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

	// LDAP 认证（如果启用）
	if global.Conf != nil && global.Conf.GetBool("ldap.enable") {
		ldapUser, err := ldap.Auth(global.Conf, req.Username, req.Password)
		if err == nil {
			// LDAP 认证成功，进行 JIT 用户同步
			var user model.AdminUser
			result := global.DB.WithContext(ctx).Where("username = ?", req.Username).First(&user)
			if result.Error == gorm.ErrRecordNotFound {
				// 创建新用户
				user = model.AdminUser{
					Username: ldapUser.Username,
					Nickname: ldapUser.Nickname,
					Email:    ldapUser.Email,
					Phone:    ldapUser.Mobile,
					Password: "", // LDAP 用户不需要本地密码
				}
				if err := repo.AdminUserCreate(ctx, &user); err != nil {
					global.Logger.WithContext(ctx).Error("LDAP JIT create user failed", zap.Error(err))
					return "", api.ErrInternalServerError
				}
				// 重新查询获取完整用户信息（包含 ID）
				if err := global.DB.WithContext(ctx).Where("username = ?", req.Username).First(&user).Error; err != nil {
					global.Logger.WithContext(ctx).Error("LDAP JIT query user failed", zap.Error(err))
					return "", api.ErrInternalServerError
				}
			} else if result.Error == nil {
				// 更新现有用户信息
				user.Nickname = ldapUser.Nickname
				user.Email = ldapUser.Email
				user.Phone = ldapUser.Mobile
				if err := repo.AdminUserUpdate(ctx, &user); err != nil {
					global.Logger.WithContext(ctx).Warn("LDAP JIT update user failed", zap.Error(err))
					// 继续执行，不阻断登录
				}
			} else {
				global.Logger.WithContext(ctx).Error("LDAP JIT query user failed", zap.Error(result.Error))
				return "", api.ErrInternalServerError
			}

			// 生成 JWT Token
			token, err := global.JWT.GenToken(user.ID, time.Now().Add(time.Hour*24*90))
			if err != nil {
				return "", err
			}
			return token, nil
		}
		// LDAP 认证失败，记录日志并回退到本地认证
		global.Logger.WithContext(ctx).Warn("LDAP login failed, falling back to local", zap.String("username", req.Username), zap.Error(err))
	}

	// 本地密码认证
	user, err := repo.GetAdminUserByUsername(ctx, req.Username)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", api.ErrUnauthorized
		}
		return "", api.ErrInternalServerError
	}

	// 如果用户是 LDAP 用户（密码为空），不允许本地密码认证
	if user.Password == "" {
		return "", api.ErrUnauthorized
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
	menu := &model.Menu{
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
	}

	// 映射 Soybean-admin 格式字段
	if req.MenuType != "" {
		menu.MenuType = req.MenuType
	}
	if req.MenuName != "" {
		menu.MenuName = req.MenuName
	} else if req.Title != "" {
		menu.MenuName = req.Title
	}
	if req.RouteName != "" {
		menu.RouteName = req.RouteName
	} else if req.Name != "" {
		menu.RouteName = req.Name
	}
	if req.RoutePath != "" {
		menu.RoutePath = req.RoutePath
	} else if req.Path != "" {
		menu.RoutePath = req.Path
	}
	if req.I18nKey != "" {
		menu.I18nKey = req.I18nKey
	} else if req.Locale != "" {
		menu.I18nKey = req.Locale
	}
	if req.IconType != "" {
		menu.IconType = req.IconType
	}
	if req.Order > 0 {
		menu.Order = req.Order
	} else {
		menu.Order = req.Weight
	}
	if req.Status != "" {
		menu.Status = req.Status
	}
	menu.MultiTab = req.MultiTab
	menu.ActiveMenu = req.ActiveMenu
	menu.Constant = req.Constant
	menu.Href = req.Href

	return repo.MenuUpdate(ctx, menu)
}

func (s *AdminService) MenuCreate(ctx context.Context, req *api.MenuCreateRequest) error {
	repo := s.getAdminRepo()
	menu := &model.Menu{
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
	}

	// 映射 Soybean-admin 格式字段
	if req.MenuType != "" {
		menu.MenuType = req.MenuType
	} else {
		menu.MenuType = "2" // 默认菜单
	}
	if req.MenuName != "" {
		menu.MenuName = req.MenuName
	} else if req.Title != "" {
		menu.MenuName = req.Title
	}
	if req.RouteName != "" {
		menu.RouteName = req.RouteName
	} else if req.Name != "" {
		menu.RouteName = req.Name
	}
	if req.RoutePath != "" {
		menu.RoutePath = req.RoutePath
	} else if req.Path != "" {
		menu.RoutePath = req.Path
	}
	if req.I18nKey != "" {
		menu.I18nKey = req.I18nKey
	} else if req.Locale != "" {
		menu.I18nKey = req.Locale
	}
	if req.IconType != "" {
		menu.IconType = req.IconType
	} else {
		menu.IconType = "1" // 默认 iconify
	}
	if req.Order > 0 {
		menu.Order = req.Order
	} else {
		menu.Order = req.Weight
	}
	if req.Status != "" {
		menu.Status = req.Status
	} else {
		menu.Status = "1" // 默认启用
	}
	menu.MultiTab = req.MultiTab
	menu.ActiveMenu = req.ActiveMenu
	menu.Constant = req.Constant
	menu.Href = req.Href

	return repo.MenuCreate(ctx, menu)
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

// GetAdminMenusSoybean 获取管理员菜单
func (s *AdminService) GetAdminMenusSoybean(ctx context.Context) (*api.GetSoybeanMenuResponseData, error) {
	repo := s.getAdminRepo()
	menuList, err := repo.GetMenuList(ctx)
	if err != nil {
		global.Logger.WithContext(ctx).Error("GetMenuList error", zap.Error(err))
		return nil, err
	}

	// 转换为map以便快速查找
	menuMap := make(map[uint]*model.Menu)
	for i := range menuList {
		menuMap[menuList[i].ID] = &menuList[i]
	}

	// 构建树形结构
	var rootMenus []*api.SoybeanMenuDataItem
	for i := range menuList {
		if menuList[i].ParentID == 0 {
			rootMenus = append(rootMenus, s.convertMenuToSoybean(&menuList[i], menuMap))
		}
	}

	// 排序根菜单
	for i := 0; i < len(rootMenus)-1; i++ {
		for j := i + 1; j < len(rootMenus); j++ {
			if rootMenus[i].Order > rootMenus[j].Order {
				rootMenus[i], rootMenus[j] = rootMenus[j], rootMenus[i]
			}
		}
	}

	// 扁平化所有菜单（包括子菜单）用于分页
	allMenus := s.flattenMenuTree(rootMenus)

	return &api.GetSoybeanMenuResponseData{
		Records: allMenus,
		Current: 1,
		Size:    len(allMenus),
		Total:   len(allMenus),
	}, nil
}

// convertMenuToSoybean 将Menu转换为Soybean格式
func (s *AdminService) convertMenuToSoybean(menu *model.Menu, menuMap map[uint]*model.Menu) *api.SoybeanMenuDataItem {
	// 确定menuType：如果有子菜单，则为目录(1)，否则为菜单(2)
	menuType := menu.MenuType
	if menuType == "" {
		// 检查是否有子菜单
		hasChildren := false
		for _, m := range menuMap {
			if m.ParentID == menu.ID {
				hasChildren = true
				break
			}
		}
		if hasChildren {
			menuType = "1" // 目录
		} else {
			menuType = "2" // 菜单
		}
	}

	// 确定routeName和routePath
	routeName := menu.RouteName
	if routeName == "" {
		routeName = menu.Name
	}
	routePath := menu.RoutePath
	if routePath == "" {
		routePath = menu.Path
	}

	// 确定menuName
	menuName := menu.MenuName
	if menuName == "" {
		menuName = menu.Title
	}

	// 确定i18nKey
	i18nKey := menu.I18nKey
	if i18nKey == "" && menu.Locale != "" {
		i18nKey = menu.Locale
	}

	// 确定iconType
	iconType := menu.IconType
	if iconType == "" {
		iconType = "1" // 默认iconify
	}

	// 确定status
	status := menu.Status
	if status == "" {
		status = "1" // 默认启用
	}

	// 确定order
	order := menu.Order
	if order == 0 {
		order = menu.Weight
	}

	item := &api.SoybeanMenuDataItem{
		ID:         menu.ID,
		CreateBy:   "",
		CreateTime: menu.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdateBy:   "",
		UpdateTime: menu.UpdatedAt.Format("2006-01-02 15:04:05"),
		Status:     status,
		ParentID:   menu.ParentID,
		MenuType:   menuType,
		MenuName:   menuName,
		RouteName:  routeName,
		RoutePath:  routePath,
		Component:  menu.Component,
		Order:      order,
		I18nKey:    i18nKey,
		Icon:       menu.Icon,
		IconType:   iconType,
		MultiTab:   menu.MultiTab,
		HideInMenu: menu.HideInMenu,
		ActiveMenu: menu.ActiveMenu,
		KeepAlive:  menu.KeepAlive,
		Constant:   menu.Constant,
		Href:       menu.Href,
		Query:      []map[string]string{},
		Buttons:    []map[string]string{},
		Children:   []*api.SoybeanMenuDataItem{},
	}

	// 查找并添加子菜单
	for _, m := range menuMap {
		if m.ParentID == menu.ID {
			child := s.convertMenuToSoybean(m, menuMap)
			item.Children = append(item.Children, child)
		}
	}

	// 排序子菜单
	for i := 0; i < len(item.Children)-1; i++ {
		for j := i + 1; j < len(item.Children); j++ {
			if item.Children[i].Order > item.Children[j].Order {
				item.Children[i], item.Children[j] = item.Children[j], item.Children[i]
			}
		}
	}

	return item
}

// flattenMenuTree 扁平化菜单树
func (s *AdminService) flattenMenuTree(menus []*api.SoybeanMenuDataItem) []*api.SoybeanMenuDataItem {
	var result []*api.SoybeanMenuDataItem
	for _, menu := range menus {
		// 创建菜单副本（不包含children）
		menuCopy := *menu
		menuCopy.Children = nil
		result = append(result, &menuCopy)
		// 递归添加子菜单
		if len(menu.Children) > 0 {
			result = append(result, s.flattenMenuTree(menu.Children)...)
		}
	}
	return result
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

// GetUserRoutes 获取用户动态路由
func (s *AdminService) GetUserRoutes(ctx context.Context, uid uint) (*api.UserRouteData, error) {
	repo := s.getAdminRepo()
	menuList, err := repo.GetMenuList(ctx)
	if err != nil {
		global.Logger.WithContext(ctx).Error("GetMenuList error", zap.Error(err))
		return nil, err
	}

	// 获取用户权限的菜单
	permissions, err := repo.GetUserPermissions(ctx, uid)
	if err != nil {
		return nil, err
	}

	menuPermMap := map[string]struct{}{}
	for _, permission := range permissions {
		// 超管可以看到所有菜单
		if convertor.ToString(uid) == model.AdminUserID {
			menuPermMap[strings.TrimPrefix(permission[1], model.MenuResourcePrefix)] = struct{}{}
		} else {
			if len(permission) == 3 && strings.HasPrefix(permission[1], model.MenuResourcePrefix) {
				menuPermMap[strings.TrimPrefix(permission[1], model.MenuResourcePrefix)] = struct{}{}
			}
		}
	}

	// 转换为map以便快速查找
	menuMap := make(map[uint]*model.Menu)
	for i := range menuList {
		menuMap[menuList[i].ID] = &menuList[i]
	}

	// 构建树形结构（只包含有权限的菜单）
	rootRoutes := make([]api.ElegantRoute, 0)

	// 超管直接获取所有顶级菜单
	isAdmin := convertor.ToString(uid) == model.AdminUserID

	for i := range menuList {
		menu := &menuList[i]

		// 超管可以看到所有菜单
		if !isAdmin {
			// 检查菜单权限
			routePath := menu.RoutePath
			if routePath == "" {
				routePath = menu.Path
			}
			if _, ok := menuPermMap[routePath]; !ok {
				continue
			}
		}

		if menu.ParentID == 0 {
			route := s.convertMenuToElegantRoute(menu, menuMap, menuPermMap, isAdmin)
			rootRoutes = append(rootRoutes, route)
		}
	}

	// 排序根路由
	for i := 0; i < len(rootRoutes)-1; i++ {
		for j := i + 1; j < len(rootRoutes); j++ {
			if rootRoutes[i].Meta.Order > rootRoutes[j].Meta.Order {
				rootRoutes[i], rootRoutes[j] = rootRoutes[j], rootRoutes[i]
			}
		}
	}

	// 确定首页路由
	home := "home"
	if len(rootRoutes) > 0 {
		// 优先查找名为 home 的路由
		foundHome := false
		for _, route := range rootRoutes {
			if route.Name == "home" {
				home = route.Name
				foundHome = true
				break
			}
		}
		// 如果没有 home 路由，查找第一个一级页面（有 layout 且有 view 的）
		if !foundHome {
			for _, route := range rootRoutes {
				if strings.Contains(route.Component, "$view.") {
					home = route.Name
					foundHome = true
					break
				}
			}
		}
		// 如果还是没有，用第一个路由
		if !foundHome && len(rootRoutes) > 0 {
			home = rootRoutes[0].Name
		}
	}

	return &api.UserRouteData{
		Routes: rootRoutes,
		Home:   home,
	}, nil
}

// convertMenuToElegantRoute 将菜单转换为 ElegantRoute 格式
func (s *AdminService) convertMenuToElegantRoute(menu *model.Menu, menuMap map[uint]*model.Menu, menuPermMap map[string]struct{}, isAdmin bool) api.ElegantRoute {
	// 确定基础 routeName
	baseRouteName := menu.RouteName
	if baseRouteName == "" {
		baseRouteName = menu.Name
	}

	// 确定最终的 routeName（子菜单需要加上父级前缀）
	routeName := baseRouteName
	if menu.ParentID != 0 {
		if parent, ok := menuMap[menu.ParentID]; ok {
			parentRouteName := parent.RouteName
			if parentRouteName == "" {
				parentRouteName = parent.Name
			}
			// 如果 routeName 还没有父级前缀，添加它
			if !strings.HasPrefix(routeName, parentRouteName+"_") {
				routeName = parentRouteName + "_" + baseRouteName
			}
		}
	}

	// 确定routePath
	routePath := menu.RoutePath
	if routePath == "" {
		routePath = menu.Path
	}

	// 确定title
	title := menu.MenuName
	if title == "" {
		title = menu.Title
	}

	// 确定i18nKey - 使用 routeName 生成
	i18nKey := menu.I18nKey
	if i18nKey == "" && menu.Locale != "" {
		i18nKey = menu.Locale
	}
	if i18nKey == "" {
		i18nKey = "route." + routeName
	}

	// 确定order
	order := menu.Order
	if order == 0 {
		order = menu.Weight
	}

	// 检查是否有子菜单
	hasChildren := false
	for _, m := range menuMap {
		if m.ParentID == menu.ID {
			hasChildren = true
			break
		}
	}

	// 智能生成 component
	component := menu.Component
	if component == "" || !strings.Contains(component, ".") {
		if menu.ParentID == 0 {
			// 顶级菜单
			if hasChildren {
				// 有子菜单，只需要 layout
				component = "layout.base"
			} else {
				// 没有子菜单，一级页面
				component = "layout.base$view." + routeName
			}
		} else {
			// 子菜单，使用 view.{routeName} 格式
			component = "view." + routeName
		}
	}

	route := api.ElegantRoute{
		Name:      routeName,
		Path:      routePath,
		Component: component,
		Redirect:  menu.Redirect,
		Meta: api.ElegantRouteMeta{
			Title:      title,
			I18nKey:    i18nKey,
			Icon:       menu.Icon,
			Order:      order,
			KeepAlive:  menu.KeepAlive,
			Constant:   menu.Constant,
			HideInMenu: menu.HideInMenu,
			ActiveMenu: menu.ActiveMenu,
			MultiTab:   menu.MultiTab,
			Href:       menu.Href,
		},
		Children: []api.ElegantRoute{},
	}

	// 查找并添加子菜单
	for _, m := range menuMap {
		if m.ParentID == menu.ID {
			// 超管跳过权限检查
			if !isAdmin {
				childRoutePath := m.RoutePath
				if childRoutePath == "" {
					childRoutePath = m.Path
				}
				if _, ok := menuPermMap[childRoutePath]; !ok {
					continue
				}
			}
			child := s.convertMenuToElegantRoute(m, menuMap, menuPermMap, isAdmin)
			route.Children = append(route.Children, child)
		}
	}

	// 排序子菜单
	for i := 0; i < len(route.Children)-1; i++ {
		for j := i + 1; j < len(route.Children); j++ {
			if route.Children[i].Meta.Order > route.Children[j].Meta.Order {
				route.Children[i], route.Children[j] = route.Children[j], route.Children[i]
			}
		}
	}

	return route
}

// SyncRoutesToDB 同步 Gin 路由到数据库
// routes 是从 gin.Engine.Routes() 获取的路由信息
func (s *AdminService) SyncRoutesToDB(ctx context.Context, routes []api.RouteInfo) error {
	repo := s.getAdminRepo()

	// 定义路由分组规则
	groupRules := map[string]string{
		"/v1/login":       "基础API",
		"/v1/menus":       "基础API",
		"/v1/user":        "基础API",
		"/v1/admin/menu":  "菜单管理",
		"/v1/admin/menus": "菜单管理",
		"/v1/admin/user":  "用户管理",
		"/v1/admin/users": "用户管理",
		"/v1/admin/role":  "角色管理",
		"/v1/admin/roles": "角色管理",
		"/v1/admin/api":   "API管理",
		"/v1/admin/apis":  "API管理",
		"/route":          "路由管理",
	}

	// 需要排除的路由（不需要权限管理的）
	excludePaths := map[string]bool{
		"/swagger/*any":     true,
		"/api/swagger/*any": true,
		"/":                 true,
	}

	syncCount := 0
	for _, route := range routes {
		// 跳过排除的路由
		if excludePaths[route.Path] {
			continue
		}

		// 只处理 /v1 或 /api/v1 开头的路由
		path := route.Path
		if strings.HasPrefix(path, "/api") {
			path = strings.TrimPrefix(path, "/api")
		}
		if !strings.HasPrefix(path, "/v1") && !strings.HasPrefix(path, "/route") {
			continue
		}

		// 检查是否已存在
		exists, err := repo.CheckApiExists(ctx, path, route.Method)
		if err != nil {
			global.Logger.Warn("检查API是否存在失败", zap.String("path", path), zap.Error(err))
			continue
		}
		if exists {
			continue
		}

		// 确定分组
		group := "其他"
		for prefix, g := range groupRules {
			if strings.HasPrefix(path, prefix) {
				group = g
				break
			}
		}

		// 生成名称
		name := generateApiName(path, route.Method)

		// 创建 API 记录
		newApi := &model.Api{
			Group:  group,
			Name:   name,
			Path:   path,
			Method: route.Method,
		}
		if err := repo.ApiCreate(ctx, newApi); err != nil {
			global.Logger.Warn("创建API失败", zap.String("path", path), zap.Error(err))
			continue
		}
		syncCount++
		global.Logger.Info("同步新API", zap.String("path", path), zap.String("method", route.Method))
	}

	global.Logger.Info("路由同步完成", zap.Int("syncCount", syncCount))
	return nil
}

// generateApiName 根据路径和方法生成 API 名称
func generateApiName(path, method string) string {
	// 方法对应的动作
	methodAction := map[string]string{
		"GET":    "获取",
		"POST":   "创建",
		"PUT":    "更新",
		"DELETE": "删除",
	}

	action := methodAction[method]
	if action == "" {
		action = method
	}

	// 从路径提取资源名
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return action + "资源"
	}

	// 取最后一个非参数部分
	resource := ""
	for i := len(parts) - 1; i >= 0; i-- {
		if !strings.HasPrefix(parts[i], ":") && parts[i] != "v1" && parts[i] != "admin" {
			resource = parts[i]
			break
		}
	}

	// 资源名映射
	resourceNames := map[string]string{
		"menus":             "菜单列表",
		"menu":              "菜单",
		"users":             "用户列表",
		"user":              "用户",
		"roles":             "角色列表",
		"role":              "角色",
		"apis":              "API列表",
		"api":               "API",
		"permissions":       "权限",
		"permission":        "权限",
		"login":             "登录",
		"getUserRoutes":     "用户路由",
		"getConstantRoutes": "常量路由",
		"isRouteExist":      "路由存在性",
	}

	if name, ok := resourceNames[resource]; ok {
		return action + name
	}

	return action + resource
}
