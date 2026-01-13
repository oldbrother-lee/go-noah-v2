package insight

import (
	"go-noah/api"
	"go-noah/internal/model/insight"
	"go-noah/internal/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// DBConfigHandlerApp 全局 Handler 实例
var DBConfigHandlerApp = new(DBConfigHandler)

// DBConfigHandler 数据库配置Handler
type DBConfigHandler struct{}

// GetDBConfigsRequest 获取数据库配置列表请求
type GetDBConfigsRequest struct {
	UseType     string `form:"use_type"`     // 用途：查询/工单
	Environment int    `form:"environment"`  // 环境ID
}

// GetDBConfigs 获取数据库配置列表
// @Summary 获取数据库配置列表
// @Tags 数据库配置
// @Security Bearer
// @Accept json
// @Produce json
// @Param use_type query string false "用途：查询/工单"
// @Param environment query int false "环境ID"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/dbconfigs [get]
func (h *DBConfigHandler) GetDBConfigs(c *gin.Context) {
	var req GetDBConfigsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.HandleError(c, http.StatusBadRequest, err, nil)
		return
	}

	configs, err := service.InsightServiceApp.GetDBConfigs(c.Request.Context(), insight.UseType(req.UseType), req.Environment)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	// 隐藏密码
	for i := range configs {
		configs[i].Password = "******"
	}

	api.HandleSuccess(c, configs)
}

// GetDBConfig 获取单个数据库配置
// @Summary 获取单个数据库配置
// @Tags 数据库配置
// @Security Bearer
// @Accept json
// @Produce json
// @Param instance_id path string true "实例ID"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/dbconfigs/{instance_id} [get]
func (h *DBConfigHandler) GetDBConfig(c *gin.Context) {
	instanceID := c.Param("instance_id")
	if instanceID == "" {
		api.HandleError(c, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}

	config, err := service.InsightServiceApp.GetDBConfigByInstanceID(c.Request.Context(), instanceID)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	// 隐藏密码
	config.Password = "******"

	api.HandleSuccess(c, config)
}

// CreateDBConfigRequest 创建数据库配置请求
type CreateDBConfigRequest struct {
	Hostname         string `json:"hostname" binding:"required"`
	Port             int    `json:"port" binding:"required"`
	UserName         string `json:"user_name" binding:"required"`
	Password         string `json:"password" binding:"required"`
	UseType          string `json:"use_type" binding:"required"` // 查询/工单
	DbType           string `json:"db_type" binding:"required"`  // MySQL/TiDB/ClickHouse
	Environment      int    `json:"environment"`
	OrganizationKey  string `json:"organization_key"`
	Remark           string `json:"remark"`
}

// CreateDBConfig 创建数据库配置
// @Summary 创建数据库配置
// @Tags 数据库配置
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body CreateDBConfigRequest true "数据库配置信息"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/dbconfigs [post]
func (h *DBConfigHandler) CreateDBConfig(c *gin.Context) {
	var req CreateDBConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.HandleError(c, http.StatusBadRequest, err, nil)
		return
	}

	// 显式生成 instance_id，与老系统保持一致
	instanceID, err := uuid.NewUUID()
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	config := &insight.DBConfig{
		InstanceID:      instanceID,
		Hostname:        req.Hostname,
		Port:            req.Port,
		UserName:        req.UserName,
		Password:        req.Password,
		UseType:         insight.UseType(req.UseType),
		DbType:          insight.DbType(req.DbType),
		Environment:     req.Environment,
		OrganizationKey: req.OrganizationKey,
		Remark:          req.Remark,
	}

	if err := service.InsightServiceApp.CreateDBConfig(c.Request.Context(), config); err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	// 隐藏密码
	config.Password = "******"
	api.HandleSuccess(c, config)
}

// UpdateDBConfigRequest 更新数据库配置请求
type UpdateDBConfigRequest struct {
	Hostname         string `json:"hostname"`
	Port             int    `json:"port"`
	UserName         string `json:"user_name"`
	Password         string `json:"password"`
	UseType          string `json:"use_type"`
	DbType           string `json:"db_type"`
	Environment      int    `json:"environment"`
	OrganizationKey  string `json:"organization_key"`
	Remark           string `json:"remark"`
}

// UpdateDBConfig 更新数据库配置
// @Summary 更新数据库配置
// @Tags 数据库配置
// @Security Bearer
// @Accept json
// @Produce json
// @Param id path int true "配置ID"
// @Param request body UpdateDBConfigRequest true "数据库配置信息"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/dbconfigs/{id} [put]
func (h *DBConfigHandler) UpdateDBConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		api.HandleError(c, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}

	var req UpdateDBConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.HandleError(c, http.StatusBadRequest, err, nil)
		return
	}

	config := &insight.DBConfig{}
	config.ID = uint(id)
	config.Hostname = req.Hostname
	config.Port = req.Port
	config.UserName = req.UserName
	config.Password = req.Password
	config.UseType = insight.UseType(req.UseType)
	config.DbType = insight.DbType(req.DbType)
	config.Environment = req.Environment
	config.OrganizationKey = req.OrganizationKey
	config.Remark = req.Remark

	if err := service.InsightServiceApp.UpdateDBConfig(c.Request.Context(), config); err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}
	api.HandleSuccess(c, nil)
}

// DeleteDBConfig 删除数据库配置
// @Summary 删除数据库配置
// @Tags 数据库配置
// @Security Bearer
// @Accept json
// @Produce json
// @Param id path int true "配置ID"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/dbconfigs/{id} [delete]
func (h *DBConfigHandler) DeleteDBConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		api.HandleError(c, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}

	if err := service.InsightServiceApp.DeleteDBConfig(c.Request.Context(), uint(id)); err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}
	api.HandleSuccess(c, nil)
}

// GetSchemas 获取实例下的Schema列表
// @Summary 获取Schema列表
// @Tags 数据库配置
// @Security Bearer
// @Accept json
// @Produce json
// @Param instance_id path string true "实例ID"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/dbconfigs/{instance_id}/schemas [get]
func (h *DBConfigHandler) GetSchemas(c *gin.Context) {
	instanceID := c.Param("instance_id")
	if instanceID == "" {
		api.HandleError(c, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}

	schemas, err := service.InsightServiceApp.GetSchemasByInstanceID(c.Request.Context(), instanceID)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}
	api.HandleSuccess(c, schemas)
}

