package insight

import (
	"context"
	"errors"
	"fmt"
	"go-noah/api"
	"go-noah/internal/das/dao"
	"go-noah/internal/handler"
	"go-noah/internal/model/insight"
	"go-noah/internal/service"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DASHandlerApp 全局 Handler 实例
var DASHandlerApp = new(DASHandler)

// DASHandler DAS Handler
type DASHandler struct{}

// ExecuteQueryRequest 执行查询请求
type ExecuteQueryRequest struct {
	InstanceID string            `json:"instance_id" binding:"required"`
	Schema     string            `json:"schema" binding:"required"`
	SQLText    string            `json:"sqltext" binding:"required"`
	Params     map[string]string `json:"params"`
}

// ExecuteQueryResponse 执行查询响应
type ExecuteQueryResponse struct {
	Columns  []string                 `json:"columns"`
	Data     []map[string]interface{} `json:"data"`
	Duration int64                    `json:"duration"`
	RowCount int                      `json:"row_count"`
	SQLText  string                   `json:"sqltext"`
}

// ExecuteQuery 执行SQL查询
// @Summary 执行SQL查询
// @Tags DAS
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body ExecuteQueryRequest true "查询请求"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/das/query [post]
func (h *DASHandler) ExecuteQuery(c *gin.Context) {
	var req ExecuteQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.HandleError(c, http.StatusBadRequest, err, nil)
		return
	}

	// 获取当前用户
	userId := handler.GetUserIdFromCtx(c)
	if userId == 0 {
		api.HandleError(c, http.StatusUnauthorized, api.ErrUnauthorized, nil)
		return
	}

	// 获取用户名
	username, err := h.getUsernameByID(c, userId)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	// 获取数据库配置
	config, err := service.InsightServiceApp.GetDBConfigByInstanceID(c.Request.Context(), req.InstanceID)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	// 检查用户权限（基于角色权限）
	effectivePerms, err := service.InsightServiceApp.GetUserEffectivePermissions(c.Request.Context(), username)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	// 检查是否有该库的权限
	hasPermission := false
	for _, perm := range effectivePerms {
		if perm.InstanceID == req.InstanceID && perm.Schema == req.Schema {
			// 如果有表权限限制，需要检查表权限
			if perm.Table != "" {
				// 表权限检查（暂时跳过，因为当前只检查库权限）
				// TODO: 实现表权限检查
			}
			hasPermission = true
			break
		}
	}
	if !hasPermission {
		api.HandleError(c, http.StatusForbidden, api.ErrForbidden, nil)
		return
	}

	// 创建数据库连接
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	db := &dao.MySQLDB{
		User:     config.UserName,
		Password: config.Password,
		Host:     config.Hostname,
		Port:     config.Port,
		Database: req.Schema,
		Params:   req.Params,
		Ctx:      ctx,
	}

	// 执行查询
	startTime := time.Now()
	columns, data, err := db.Query(req.SQLText)
	duration := time.Since(startTime).Milliseconds()

	// 记录执行日志
	record := &insight.DASRecord{
		Username:   username,
		InstanceID: config.InstanceID,
		Schema:     req.Schema,
		SQL:        req.SQLText,
		Duration:   duration,
		RowCount:   int64(len(data)),
	}
	if err != nil {
		record.Error = err.Error()
	}
	_ = service.InsightServiceApp.CreateDASRecord(c.Request.Context(), record)

	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	api.HandleSuccess(c, ExecuteQueryResponse{
		Columns:  columns,
		Data:     data,
		Duration: duration,
		RowCount: len(data),
		SQLText:  req.SQLText,
	})
}

// UserSchemaResult 用户授权的 schema 返回结构
type UserSchemaResult struct {
	InstanceID uuid.UUID `json:"instance_id"`
	Schema     string    `json:"schema"`
	DbType     string    `json:"db_type"`
	Hostname   string    `json:"hostname"`
	Port       int       `json:"port"`
	IsDeleted  bool      `json:"is_deleted"`
	Remark     string    `json:"remark"`
}

// GetUserSchemas 获取用户有权限的所有Schema（用于SQL查询页面）
// @Summary 获取用户授权的Schema列表
// @Tags DAS
// @Security Bearer
// @Accept json
// @Produce json
// @Success 200 {object} api.Response
// @Router /api/v1/insight/das/schemas [get]
func (h *DASHandler) GetUserSchemas(c *gin.Context) {
	// 获取当前用户
	userId := handler.GetUserIdFromCtx(c)
	if userId == 0 {
		api.HandleError(c, http.StatusUnauthorized, api.ErrUnauthorized, nil)
		return
	}

	// 获取用户名
	username, err := h.getUsernameByID(c, userId)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	// 获取用户授权的 schemas
	results, err := service.InsightServiceApp.GetUserAuthorizedSchemas(c.Request.Context(), username)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	api.HandleSuccess(c, results)
}

// GetSchemas 获取实例下的数据库列表
// @Summary 获取数据库列表
// @Tags DAS
// @Security Bearer
// @Accept json
// @Produce json
// @Param instance_id path string true "实例ID"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/das/schemas/{instance_id} [get]
func (h *DASHandler) GetSchemas(c *gin.Context) {
	instanceID := c.Param("instance_id")
	if instanceID == "" {
		api.HandleError(c, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}

	// 获取数据库配置
	config, err := service.InsightServiceApp.GetDBConfigByInstanceID(c.Request.Context(), instanceID)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	db := &dao.MySQLDB{
		User:     config.UserName,
		Password: config.Password,
		Host:     config.Hostname,
		Port:     config.Port,
		Ctx:      ctx,
	}

	schemas, err := db.GetSchemas()
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	api.HandleSuccess(c, schemas)
}

// GetTables 获取指定库的表列表
// @Summary 获取表列表
// @Tags DAS
// @Security Bearer
// @Accept json
// @Produce json
// @Param instance_id path string true "实例ID"
// @Param schema path string true "数据库名"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/das/tables/{instance_id}/{schema} [get]
func (h *DASHandler) GetTables(c *gin.Context) {
	instanceID := c.Param("instance_id")
	schema := c.Param("schema")
	if instanceID == "" || schema == "" {
		api.HandleError(c, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}

	config, err := service.InsightServiceApp.GetDBConfigByInstanceID(c.Request.Context(), instanceID)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	db := &dao.MySQLDB{
		User:     config.UserName,
		Password: config.Password,
		Host:     config.Hostname,
		Port:     config.Port,
		Ctx:      ctx,
		Params:   map[string]string{"group_concat_max_len": "4194304"},
	}

	tables, err := db.GetTables(schema)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	api.HandleSuccess(c, tables)
}

// GetTableColumns 获取表结构
// @Summary 获取表结构
// @Tags DAS
// @Security Bearer
// @Accept json
// @Produce json
// @Param instance_id path string true "实例ID"
// @Param schema path string true "数据库名"
// @Param table path string true "表名"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/das/columns/{instance_id}/{schema}/{table} [get]
func (h *DASHandler) GetTableColumns(c *gin.Context) {
	instanceID := c.Param("instance_id")
	schema := c.Param("schema")
	table := c.Param("table")
	if instanceID == "" || schema == "" || table == "" {
		api.HandleError(c, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}

	config, err := service.InsightServiceApp.GetDBConfigByInstanceID(c.Request.Context(), instanceID)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	db := &dao.MySQLDB{
		User:     config.UserName,
		Password: config.Password,
		Host:     config.Hostname,
		Port:     config.Port,
		Ctx:      ctx,
	}

	columns, err := db.GetTableColumns(schema, table)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	api.HandleSuccess(c, columns)
}

// GetRecords 获取执行记录
// @Summary 获取执行记录
// @Tags DAS
// @Security Bearer
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/das/records [get]
func (h *DASHandler) GetRecords(c *gin.Context) {
	userId := handler.GetUserIdFromCtx(c)
	if userId == 0 {
		api.HandleError(c, http.StatusUnauthorized, api.ErrUnauthorized, nil)
		return
	}

	username, err := h.getUsernameByID(c, userId)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	records, total, err := service.InsightServiceApp.GetDASRecords(c.Request.Context(), username, page, pageSize)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	api.HandleSuccess(c, gin.H{
		"list":  records,
		"total": total,
	})
}

// ============ 收藏夹 ============

// CreateFavoriteRequest 创建收藏请求
type CreateFavoriteRequest struct {
	Title string `json:"title" binding:"required"`
	SQL   string `json:"sql" binding:"required"`
}

// GetFavorites 获取收藏列表
// @Summary 获取收藏列表
// @Tags DAS
// @Security Bearer
// @Accept json
// @Produce json
// @Success 200 {object} api.Response
// @Router /api/v1/insight/das/favorites [get]
func (h *DASHandler) GetFavorites(c *gin.Context) {
	userId := handler.GetUserIdFromCtx(c)
	if userId == 0 {
		api.HandleError(c, http.StatusUnauthorized, api.ErrUnauthorized, nil)
		return
	}

	username, err := h.getUsernameByID(c, userId)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	favorites, err := service.InsightServiceApp.GetFavorites(c.Request.Context(), username)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	api.HandleSuccess(c, favorites)
}

// CreateFavorite 创建收藏
// @Summary 创建收藏
// @Tags DAS
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body CreateFavoriteRequest true "收藏信息"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/das/favorites [post]
func (h *DASHandler) CreateFavorite(c *gin.Context) {
	userId := handler.GetUserIdFromCtx(c)
	if userId == 0 {
		api.HandleError(c, http.StatusUnauthorized, api.ErrUnauthorized, nil)
		return
	}

	username, err := h.getUsernameByID(c, userId)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	var req CreateFavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.HandleError(c, http.StatusBadRequest, err, nil)
		return
	}

	fav := &insight.DASFavorite{
		Username: username,
		Title:    req.Title,
		SQL:      req.SQL,
	}

	if err := service.InsightServiceApp.CreateFavorite(c.Request.Context(), fav); err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	api.HandleSuccess(c, fav)
}

// DeleteFavorite 删除收藏
// @Summary 删除收藏
// @Tags DAS
// @Security Bearer
// @Accept json
// @Produce json
// @Param id path int true "收藏ID"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/das/favorites/{id} [delete]
func (h *DASHandler) DeleteFavorite(c *gin.Context) {
	userId := handler.GetUserIdFromCtx(c)
	if userId == 0 {
		api.HandleError(c, http.StatusUnauthorized, api.ErrUnauthorized, nil)
		return
	}

	username, err := h.getUsernameByID(c, userId)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		api.HandleError(c, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}

	if err := service.InsightServiceApp.DeleteFavorite(c.Request.Context(), uint(id), username); err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	api.HandleSuccess(c, nil)
}

// ============ 权限管理 ============

// GrantSchemaPermissionRequest 授权Schema权限请求
type GrantSchemaPermissionRequest struct {
	Username   string `json:"username" binding:"required"`
	InstanceID string `json:"instance_id" binding:"required"`
	Schema     string `json:"schema" binding:"required"`
}

// GetUserPermissions 获取用户权限
// @Summary 获取用户权限
// @Tags DAS
// @Security Bearer
// @Accept json
// @Produce json
// @Param username query string true "用户名"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/das/permissions [get]
func (h *DASHandler) GetUserPermissions(c *gin.Context) {
	username := c.Query("username")
	if username == "" {
		api.HandleError(c, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}

	schemaPerms, err := service.InsightServiceApp.GetUserSchemaPermissions(c.Request.Context(), username)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	tablePerms, err := service.InsightServiceApp.GetUserTablePermissions(c.Request.Context(), username)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	api.HandleSuccess(c, gin.H{
		"schema_permissions": schemaPerms,
		"table_permissions":  tablePerms,
	})
}

// ============ 权限模板管理 ============

// GetPermissionTemplates 获取权限模板列表
func (h *DASHandler) GetPermissionTemplates(c *gin.Context) {
	templates, err := service.InsightServiceApp.GetPermissionTemplates(c.Request.Context())
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}
	api.HandleSuccess(c, templates)
}

// GetPermissionTemplate 获取权限模板详情
func (h *DASHandler) GetPermissionTemplate(c *gin.Context) {
	id := c.Param("id")
	var templateID uint
	if _, err := fmt.Sscanf(id, "%d", &templateID); err != nil {
		api.HandleError(c, http.StatusBadRequest, err, nil)
		return
	}

	template, err := service.InsightServiceApp.GetPermissionTemplate(c.Request.Context(), templateID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.HandleError(c, http.StatusNotFound, err, nil)
			return
		}
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}
	api.HandleSuccess(c, template)
}

// CreatePermissionTemplate 创建权限模板
func (h *DASHandler) CreatePermissionTemplate(c *gin.Context) {
	var req struct {
		Name        string                     `json:"name" binding:"required"`
		Description string                     `json:"description"`
		Permissions []insight.PermissionObject `json:"permissions" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		api.HandleError(c, http.StatusBadRequest, err, nil)
		return
	}

	template := &insight.DASPermissionTemplate{
		Name:        req.Name,
		Description: req.Description,
		Permissions: req.Permissions,
	}

	if err := service.InsightServiceApp.CreatePermissionTemplate(c.Request.Context(), template); err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	api.HandleSuccess(c, template)
}

// UpdatePermissionTemplate 更新权限模板
func (h *DASHandler) UpdatePermissionTemplate(c *gin.Context) {
	id := c.Param("id")
	var templateID uint
	if _, err := fmt.Sscanf(id, "%d", &templateID); err != nil {
		api.HandleError(c, http.StatusBadRequest, err, nil)
		return
	}

	var req struct {
		Name        string                     `json:"name" binding:"required"`
		Description string                     `json:"description"`
		Permissions []insight.PermissionObject `json:"permissions" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		api.HandleError(c, http.StatusBadRequest, err, nil)
		return
	}

	template := &insight.DASPermissionTemplate{
		Model:       gorm.Model{ID: templateID},
		Name:        req.Name,
		Description: req.Description,
		Permissions: req.Permissions,
	}

	if err := service.InsightServiceApp.UpdatePermissionTemplate(c.Request.Context(), template); err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	api.HandleSuccess(c, template)
}

// DeletePermissionTemplate 删除权限模板
func (h *DASHandler) DeletePermissionTemplate(c *gin.Context) {
	id := c.Param("id")
	var templateID uint
	if _, err := fmt.Sscanf(id, "%d", &templateID); err != nil {
		api.HandleError(c, http.StatusBadRequest, err, nil)
		return
	}

	if err := service.InsightServiceApp.DeletePermissionTemplate(c.Request.Context(), templateID); err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	api.HandleSuccess(c, nil)
}

// ============ 角色权限管理 ============

// GetRolePermissions 获取角色权限列表
func (h *DASHandler) GetRolePermissions(c *gin.Context) {
	role := c.Param("role")
	if role == "" {
		api.HandleError(c, http.StatusBadRequest, fmt.Errorf("role is required"), nil)
		return
	}

	perms, err := service.InsightServiceApp.GetRolePermissions(c.Request.Context(), role)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}
	api.HandleSuccess(c, perms)
}

// CreateRolePermission 创建角色权限
func (h *DASHandler) CreateRolePermission(c *gin.Context) {
	var req struct {
		Role           string `json:"role" binding:"required"`
		PermissionType string `json:"permission_type" binding:"required,oneof=object template"`
		PermissionID   uint   `json:"permission_id" binding:"required"`
		InstanceID     string `json:"instance_id,omitempty"`
		Schema         string `json:"schema,omitempty"`
		Table          string `json:"table,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		api.HandleError(c, http.StatusBadRequest, err, nil)
		return
	}

	perm := &insight.DASRolePermission{
		Role:           req.Role,
		PermissionType: insight.PermissionType(req.PermissionType),
		PermissionID:   req.PermissionID,
		InstanceID:     req.InstanceID,
		Schema:         req.Schema,
		Table:          req.Table,
	}

	if err := service.InsightServiceApp.CreateRolePermission(c.Request.Context(), perm); err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	api.HandleSuccess(c, perm)
}

// DeleteRolePermission 删除角色权限
func (h *DASHandler) DeleteRolePermission(c *gin.Context) {
	id := c.Param("id")
	var permID uint
	if _, err := fmt.Sscanf(id, "%d", &permID); err != nil {
		api.HandleError(c, http.StatusBadRequest, err, nil)
		return
	}

	if err := service.InsightServiceApp.DeleteRolePermission(c.Request.Context(), permID); err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	api.HandleSuccess(c, nil)
}

// GetUserEffectivePermissions 获取用户实际生效的权限
func (h *DASHandler) GetUserEffectivePermissions(c *gin.Context) {
	username := c.Query("username")
	if username == "" {
		api.HandleError(c, http.StatusBadRequest, fmt.Errorf("username is required"), nil)
		return
	}

	perms, err := service.InsightServiceApp.GetUserEffectivePermissions(c.Request.Context(), username)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}
	api.HandleSuccess(c, perms)
}

// GrantSchemaPermission 授权Schema权限
// @Summary 授权Schema权限
// @Tags DAS
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body GrantSchemaPermissionRequest true "授权信息"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/das/permissions/schema [post]
func (h *DASHandler) GrantSchemaPermission(c *gin.Context) {
	var req GrantSchemaPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.HandleError(c, http.StatusBadRequest, err, nil)
		return
	}

	instanceUUID, err := uuid.Parse(req.InstanceID)
	if err != nil {
		api.HandleError(c, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}

	perm := &insight.DASUserSchemaPermission{
		Username:   req.Username,
		InstanceID: instanceUUID,
		Schema:     req.Schema,
	}

	if err := service.InsightServiceApp.CreateSchemaPermission(c.Request.Context(), perm); err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	api.HandleSuccess(c, perm)
}

// RevokeSchemaPermission 撤销Schema权限
// @Summary 撤销Schema权限
// @Tags DAS
// @Security Bearer
// @Accept json
// @Produce json
// @Param id path int true "权限ID"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/das/permissions/schema/{id} [delete]
func (h *DASHandler) RevokeSchemaPermission(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		api.HandleError(c, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}

	if err := service.InsightServiceApp.DeleteSchemaPermission(c.Request.Context(), uint(id)); err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	api.HandleSuccess(c, nil)
}

// getUsernameByID 通过用户ID获取用户名
func (h *DASHandler) getUsernameByID(c *gin.Context, userId uint) (string, error) {
	user, err := service.AdminServiceApp.GetAdminUser(c, userId)
	if err != nil {
		return "", fmt.Errorf("获取用户信息失败: %w", err)
	}
	return user.Username, nil
}
