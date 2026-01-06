package handler

import (
	"go-noah/api"
	"go-noah/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AdminHandler 管理员 Handler
type AdminHandler struct{}

// AdminHandlerApp 全局 Handler 实例
var AdminHandlerApp = new(AdminHandler)

// Login godoc
// @Summary 账号登录
// @Schemes
// @Description
// @Tags 用户模块
// @Accept json
// @Produce json
// @Param request body api.LoginRequest true "params"
// @Success 200 {object} api.LoginResponse
// @Router /v1/login [post]
func (h *AdminHandler) Login(ctx *gin.Context) {
	var req api.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleError(ctx, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}

	token, err := service.AdminServiceApp.Login(ctx, &req)
	if err != nil {
		api.HandleError(ctx, http.StatusUnauthorized, api.ErrUnauthorized, nil)
		return
	}
	api.HandleSuccess(ctx, api.LoginResponseData{
		AccessToken: token,
	})
}

// GetMenus godoc
// @Summary 获取用户菜单
// @Schemes
// @Description 获取当前用户的菜单列表
// @Tags 菜单模块
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} api.GetMenuResponse
// @Router /v1/menus [get]
func (h *AdminHandler) GetMenus(ctx *gin.Context) {
	data, err := service.AdminServiceApp.GetMenus(ctx, GetUserIdFromCtx(ctx))
	if err != nil {
		api.HandleError(ctx, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}
	// 过滤权限菜单
	api.HandleSuccess(ctx, data)
}

// GetAdminMenus godoc
// @Summary 获取管理员菜单
// @Schemes
// @Description 获取管理员菜单列表（Soybean-admin格式）
// @Tags 菜单模块
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} api.GetSoybeanMenuResponse
// @Router /v1/admin/menus [get]
func (h *AdminHandler) GetAdminMenus(ctx *gin.Context) {
	data, err := service.AdminServiceApp.GetAdminMenusSoybean(ctx)
	if err != nil {
		api.HandleError(ctx, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}
	api.HandleSuccess(ctx, data)
}

// GetUserPermissions godoc
// @Summary 获取用户权限
// @Schemes
// @Description 获取当前用户的权限列表
// @Tags 权限模块
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} api.GetUserPermissionsData
// @Router /v1/admin/user/permissions [get]
func (h *AdminHandler) GetUserPermissions(ctx *gin.Context) {
	data, err := service.AdminServiceApp.GetUserPermissions(ctx, GetUserIdFromCtx(ctx))
	if err != nil {
		api.HandleError(ctx, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}
	// 过滤权限菜单
	api.HandleSuccess(ctx, data)
}

// GetRolePermissions godoc
// @Summary 获取角色权限
// @Schemes
// @Description 获取指定角色的权限列表
// @Tags 权限模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param role query string true "角色名称"
// @Success 200 {object} api.GetRolePermissionsData
// @Router /v1/admin/role/permissions [get]
func (h *AdminHandler) GetRolePermissions(ctx *gin.Context) {
	var req api.GetRolePermissionsRequest
	if err := ctx.ShouldBind(&req); err != nil {
		api.HandleError(ctx, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}
	data, err := service.AdminServiceApp.GetRolePermissions(ctx, req.Role)
	if err != nil {
		api.HandleError(ctx, http.StatusInternalServerError, api.ErrInternalServerError, nil)
		return
	}
	api.HandleSuccess(ctx, data)
}

// UpdateRolePermission godoc
// @Summary 更新角色权限
// @Schemes
// @Description 更新指定角色的权限列表
// @Tags 权限模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body api.UpdateRolePermissionRequest true "参数"
// @Success 200 {object} api.Response
// @Router /v1/admin/role/permissions [put]
func (h *AdminHandler) UpdateRolePermission(ctx *gin.Context) {
	var req api.UpdateRolePermissionRequest
	if err := ctx.ShouldBind(&req); err != nil {
		api.HandleError(ctx, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}
	err := service.AdminServiceApp.UpdateRolePermission(ctx, &req)
	if err != nil {
		api.HandleError(ctx, http.StatusInternalServerError, api.ErrInternalServerError, nil)
		return
	}
	api.HandleSuccess(ctx, nil)
}

// MenuUpdate godoc
// @Summary 更新菜单
// @Schemes
// @Description 更新菜单信息
// @Tags 菜单模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body api.MenuUpdateRequest true "参数"
// @Success 200 {object} api.Response
// @Router /v1/admin/menu [put]
func (h *AdminHandler) MenuUpdate(ctx *gin.Context) {
	var req api.MenuUpdateRequest
	if err := ctx.ShouldBind(&req); err != nil {
		api.HandleError(ctx, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}
	if err := service.AdminServiceApp.MenuUpdate(ctx, &req); err != nil {
		api.HandleError(ctx, http.StatusInternalServerError, api.ErrInternalServerError, nil)
		return
	}
	api.HandleSuccess(ctx, nil)
}

// MenuCreate godoc
// @Summary 创建菜单
// @Schemes
// @Description 创建新的菜单
// @Tags 菜单模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body api.MenuCreateRequest true "参数"
// @Success 200 {object} api.Response
// @Router /v1/admin/menu [post]
func (h *AdminHandler) MenuCreate(ctx *gin.Context) {
	var req api.MenuCreateRequest
	if err := ctx.ShouldBind(&req); err != nil {
		api.HandleError(ctx, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}
	if err := service.AdminServiceApp.MenuCreate(ctx, &req); err != nil {
		api.HandleError(ctx, http.StatusInternalServerError, api.ErrInternalServerError, nil)
		return
	}
	api.HandleSuccess(ctx, nil)
}

// MenuDelete godoc
// @Summary 删除菜单
// @Schemes
// @Description 删除指定菜单
// @Tags 菜单模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param id query uint true "菜单ID"
// @Success 200 {object} api.Response
// @Router /v1/admin/menu [delete]
func (h *AdminHandler) MenuDelete(ctx *gin.Context) {
	var req api.MenuDeleteRequest
	if err := ctx.ShouldBind(&req); err != nil {
		api.HandleError(ctx, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}
	if err := service.AdminServiceApp.MenuDelete(ctx, req.ID); err != nil {
		api.HandleError(ctx, http.StatusInternalServerError, api.ErrInternalServerError, nil)
		return

	}
	api.HandleSuccess(ctx, nil)
}

// GetRoles godoc
// @Summary 获取角色列表
// @Schemes
// @Description 获取角色列表
// @Tags 角色模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int true "页码"
// @Param pageSize query int true "每页数量"
// @Param sid query string false "角色ID"
// @Param name query string false "角色名称"
// @Success 200 {object} api.GetRolesResponse
// @Router /v1/admin/roles [get]
func (h *AdminHandler) GetRoles(ctx *gin.Context) {
	var req api.GetRoleListRequest
	if err := ctx.ShouldBind(&req); err != nil {
		api.HandleError(ctx, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}
	data, err := service.AdminServiceApp.GetRoles(ctx, &req)
	if err != nil {
		api.HandleError(ctx, http.StatusInternalServerError, api.ErrInternalServerError, nil)
		return
	}

	api.HandleSuccess(ctx, data)
}

// RoleCreate godoc
// @Summary 创建角色
// @Schemes
// @Description 创建新的角色
// @Tags 角色模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body api.RoleCreateRequest true "参数"
// @Success 200 {object} api.Response
// @Router /v1/admin/role [post]
func (h *AdminHandler) RoleCreate(ctx *gin.Context) {
	var req api.RoleCreateRequest
	if err := ctx.ShouldBind(&req); err != nil {
		api.HandleError(ctx, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}
	if err := service.AdminServiceApp.RoleCreate(ctx, &req); err != nil {
		api.HandleError(ctx, http.StatusInternalServerError, api.ErrInternalServerError, nil)
		return
	}
	api.HandleSuccess(ctx, nil)
}

// RoleUpdate godoc
// @Summary 更新角色
// @Schemes
// @Description 更新角色信息
// @Tags 角色模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body api.RoleUpdateRequest true "参数"
// @Success 200 {object} api.Response
// @Router /v1/admin/role [put]
func (h *AdminHandler) RoleUpdate(ctx *gin.Context) {
	var req api.RoleUpdateRequest
	if err := ctx.ShouldBind(&req); err != nil {
		api.HandleError(ctx, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}
	if err := service.AdminServiceApp.RoleUpdate(ctx, &req); err != nil {
		api.HandleError(ctx, http.StatusInternalServerError, api.ErrInternalServerError, nil)
		return
	}
	api.HandleSuccess(ctx, nil)
}

// RoleDelete godoc
// @Summary 删除角色
// @Schemes
// @Description 删除指定角色
// @Tags 角色模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param id query uint true "角色ID"
// @Success 200 {object} api.Response
// @Router /v1/admin/role [delete]
func (h *AdminHandler) RoleDelete(ctx *gin.Context) {
	var req api.RoleDeleteRequest
	if err := ctx.ShouldBind(&req); err != nil {
		api.HandleError(ctx, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}
	if err := service.AdminServiceApp.RoleDelete(ctx, req.ID); err != nil {
		api.HandleError(ctx, http.StatusInternalServerError, api.ErrInternalServerError, nil)
		return
	}
	api.HandleSuccess(ctx, nil)
}

// GetApis godoc
// @Summary 获取API列表
// @Schemes
// @Description 获取API列表
// @Tags API模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int true "页码"
// @Param pageSize query int true "每页数量"
// @Param group query string false "API分组"
// @Param name query string false "API名称"
// @Param path query string false "API路径"
// @Param method query string false "请求方法"
// @Success 200 {object} api.GetApisResponse
// @Router /v1/admin/apis [get]
func (h *AdminHandler) GetApis(ctx *gin.Context) {
	var req api.GetApisRequest
	if err := ctx.ShouldBind(&req); err != nil {
		api.HandleError(ctx, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}
	data, err := service.AdminServiceApp.GetApis(ctx, &req)
	if err != nil {
		api.HandleError(ctx, http.StatusInternalServerError, api.ErrInternalServerError, nil)
		return
	}

	api.HandleSuccess(ctx, data)
}

// ApiCreate godoc
// @Summary 创建API
// @Schemes
// @Description 创建新的API
// @Tags API模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body api.ApiCreateRequest true "参数"
// @Success 200 {object} api.Response
// @Router /v1/admin/api [post]
func (h *AdminHandler) ApiCreate(ctx *gin.Context) {
	var req api.ApiCreateRequest
	if err := ctx.ShouldBind(&req); err != nil {
		api.HandleError(ctx, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}
	if err := service.AdminServiceApp.ApiCreate(ctx, &req); err != nil {
		api.HandleError(ctx, http.StatusInternalServerError, api.ErrInternalServerError, nil)
		return
	}
	api.HandleSuccess(ctx, nil)
}

// ApiUpdate godoc
// @Summary 更新API
// @Schemes
// @Description 更新API信息
// @Tags API模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body api.ApiUpdateRequest true "参数"
// @Success 200 {object} api.Response
// @Router /v1/admin/api [put]
func (h *AdminHandler) ApiUpdate(ctx *gin.Context) {
	var req api.ApiUpdateRequest
	if err := ctx.ShouldBind(&req); err != nil {
		api.HandleError(ctx, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}
	if err := service.AdminServiceApp.ApiUpdate(ctx, &req); err != nil {
		api.HandleError(ctx, http.StatusInternalServerError, api.ErrInternalServerError, nil)
		return
	}
	api.HandleSuccess(ctx, nil)
}

// ApiDelete godoc
// @Summary 删除API
// @Schemes
// @Description 删除指定API
// @Tags API模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param id query uint true "API ID"
// @Success 200 {object} api.Response
// @Router /v1/admin/api [delete]
func (h *AdminHandler) ApiDelete(ctx *gin.Context) {
	var req api.ApiDeleteRequest
	if err := ctx.ShouldBind(&req); err != nil {
		api.HandleError(ctx, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}
	if err := service.AdminServiceApp.ApiDelete(ctx, req.ID); err != nil {
		api.HandleError(ctx, http.StatusInternalServerError, api.ErrInternalServerError, nil)
		return
	}
	api.HandleSuccess(ctx, nil)
}

// AdminUserUpdate godoc
// @Summary 更新管理员用户
// @Schemes
// @Description 更新管理员用户信息
// @Tags 用户模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body api.AdminUserUpdateRequest true "参数"
// @Success 200 {object} api.Response
// @Router /v1/admin/user [put]
func (h *AdminHandler) AdminUserUpdate(ctx *gin.Context) {
	var req api.AdminUserUpdateRequest
	if err := ctx.ShouldBind(&req); err != nil {
		api.HandleError(ctx, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}
	if err := service.AdminServiceApp.AdminUserUpdate(ctx, &req); err != nil {
		api.HandleError(ctx, http.StatusInternalServerError, api.ErrInternalServerError, nil)
		return
	}
	api.HandleSuccess(ctx, nil)
}

// AdminUserCreate godoc
// @Summary 创建管理员用户
// @Schemes
// @Description 创建新的管理员用户
// @Tags 用户模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body api.AdminUserCreateRequest true "参数"
// @Success 200 {object} api.Response
// @Router /v1/admin/user [post]
func (h *AdminHandler) AdminUserCreate(ctx *gin.Context) {
	var req api.AdminUserCreateRequest
	if err := ctx.ShouldBind(&req); err != nil {
		api.HandleError(ctx, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}
	if err := service.AdminServiceApp.AdminUserCreate(ctx, &req); err != nil {
		api.HandleError(ctx, http.StatusInternalServerError, api.ErrInternalServerError, nil)
		return
	}
	api.HandleSuccess(ctx, nil)
}

// AdminUserDelete godoc
// @Summary 删除管理员用户
// @Schemes
// @Description 删除指定管理员用户
// @Tags 用户模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param id query uint true "用户ID"
// @Success 200 {object} api.Response
// @Router /v1/admin/user [delete]
func (h *AdminHandler) AdminUserDelete(ctx *gin.Context) {
	var req api.AdminUserDeleteRequest
	if err := ctx.ShouldBind(&req); err != nil {
		api.HandleError(ctx, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}
	if err := service.AdminServiceApp.AdminUserDelete(ctx, req.ID); err != nil {
		api.HandleError(ctx, http.StatusInternalServerError, api.ErrInternalServerError, nil)
		return

	}
	api.HandleSuccess(ctx, nil)
}

// GetAdminUsers godoc
// @Summary 获取管理员用户列表
// @Schemes
// @Description 获取管理员用户列表
// @Tags 用户模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int true "页码"
// @Param pageSize query int true "每页数量"
// @Param username query string false "用户名"
// @Param nickname query string false "昵称"
// @Param phone query string false "手机号"
// @Param email query string false "邮箱"
// @Success 200 {object} api.GetAdminUsersResponse
// @Router /v1/admin/users [get]
func (h *AdminHandler) GetAdminUsers(ctx *gin.Context) {
	var req api.GetAdminUsersRequest
	if err := ctx.ShouldBind(&req); err != nil {
		api.HandleError(ctx, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}
	data, err := service.AdminServiceApp.GetAdminUsers(ctx, &req)
	if err != nil {
		api.HandleError(ctx, http.StatusInternalServerError, api.ErrInternalServerError, nil)
		return
	}

	api.HandleSuccess(ctx, data)
}

// GetAdminUser godoc
// @Summary 获取管理用户信息
// @Schemes
// @Description
// @Tags 用户模块
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} api.GetAdminUserResponse
// @Router /v1/admin/user [get]
func (h *AdminHandler) GetAdminUser(ctx *gin.Context) {
	data, err := service.AdminServiceApp.GetAdminUser(ctx, GetUserIdFromCtx(ctx))
	if err != nil {
		api.HandleError(ctx, http.StatusInternalServerError, api.ErrInternalServerError, nil)
		return
	}

	api.HandleSuccess(ctx, data)
}
