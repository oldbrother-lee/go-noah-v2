package insight

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go-noah/api"
	"go-noah/internal/inspect/checker"
	"go-noah/internal/inspect/config"
	"go-noah/internal/inspect/parser"
	"go-noah/internal/model/insight"
	"go-noah/internal/service"
	"go-noah/pkg/global"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// InspectHandlerApp 全局 Handler 实例
var InspectHandlerApp = new(InspectHandler)

// InspectHandler SQL审核 Handler
type InspectHandler struct{}

// InspectSQLRequest 审核SQL请求
type InspectSQLRequest struct {
	Content    string `json:"content" binding:"required"` // SQL内容
	DBType     string `json:"db_type"`                    // MySQL/TiDB
	SQLType    string `json:"sql_type"`                   // DDL/DML
	InstanceID string `json:"instance_id"`                // 可选，用于获取实例配置的审核参数
	Schema     string `json:"schema"`                     // 数据库名
}

// InspectSQL 审核SQL
// @Summary 审核SQL
// @Tags SQL审核
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body InspectSQLRequest true "审核请求"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/inspect/sql [post]
func (h *InspectHandler) InspectSQL(c *gin.Context) {
	var req InspectSQLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.HandleError(c, http.StatusBadRequest, err, nil)
		return
	}

	// 获取审核参数和数据库配置
	var params *config.InspectParams
	var dbConfig *insight.DBConfig
	if req.InstanceID != "" {
		// 从数据库配置中获取审核参数
		var err error
		dbConfig, err = service.InsightServiceApp.GetDBConfigByInstanceID(c.Request.Context(), req.InstanceID)
		if err != nil {
			api.HandleError(c, http.StatusOK, fmt.Errorf("获取数据库配置失败: %s", err.Error()), nil)
			return
		}
		if dbConfig.InspectParams != nil && len(dbConfig.InspectParams) > 0 {
			// 先创建默认参数作为基础
			params = config.DefaultInspectParams()
			// 从数据库配置中反序列化参数，覆盖默认值（使用 Decoder，保留默认值，只覆盖存在的字段）
			r := bytes.NewReader(dbConfig.InspectParams)
			decoder := json.NewDecoder(r)
			if err := decoder.Decode(params); err != nil {
				global.Logger.Warn("反序列化审核参数失败，使用默认参数",
					zap.Error(err),
					zap.String("instance_id", req.InstanceID),
					zap.String("inspect_params", string(dbConfig.InspectParams)),
				)
				// 反序列化失败，使用默认参数（params 已经是默认参数）
			} else {
				// 验证参数有效性：检查关键字段是否为零值（Decoder 会保留默认值，但以防万一还是验证）
				if params.MAX_VARCHAR_LENGTH == 0 {
					global.Logger.Warn("审核参数MAX_VARCHAR_LENGTH为0，使用默认值16383",
						zap.String("instance_id", req.InstanceID),
					)
					params.MAX_VARCHAR_LENGTH = 16383
				}
				// 验证 TABLE_SUPPORT_CHARSET 是否为空（如果数据库配置中没有该字段，Decoder 会保留默认值）
				if len(params.TABLE_SUPPORT_CHARSET) == 0 {
					global.Logger.Warn("审核参数TABLE_SUPPORT_CHARSET为空，使用默认值",
						zap.String("instance_id", req.InstanceID),
					)
					params.TABLE_SUPPORT_CHARSET = []map[string]string{
						{"charset": "utf8", "recommend": "utf8_general_ci"},
						{"charset": "utf8mb4", "recommend": "utf8mb4_general_ci"},
					}
				}
			}
		}
	}

	// 使用默认参数
	if params == nil {
		params = config.DefaultInspectParams()
	}

	// 判断SQL类型是否匹配，DML工单仅允许提交DML语句，DDL工单仅允许提交DDL语句
	if req.SQLType != "" && req.SQLType != "EXPORT" {
		if err := parser.CheckSqlType(req.Content, req.SQLType); err != nil {
			api.HandleError(c, http.StatusOK, err, nil)
			return
		}
	}

	// 导出工单和 ClickHouse 不审核
	if req.SQLType == "EXPORT" || req.DBType == "ClickHouse" {
		api.HandleSuccess(c, []interface{}{})
		return
	}

	// 创建审核器并执行审核
	chk := checker.NewChecker(params, req.DBType)
	
	// 设置数据库连接信息（用于表存在性检查）
	// 注意：即使 req.Schema 为空，也应该设置连接信息，因为 SQL 中可能指定了数据库名
	if dbConfig != nil {
		chk.SetDBInfo(dbConfig.Hostname, dbConfig.Port, dbConfig.UserName, dbConfig.Password, req.Schema)
		global.Logger.Info("设置数据库连接信息",
			zap.String("instance_id", req.InstanceID),
			zap.String("host", dbConfig.Hostname),
			zap.Int("port", dbConfig.Port),
			zap.String("user", dbConfig.UserName),
			zap.String("password", "***"), // 不打印真实密码
			zap.String("req_schema", req.Schema),
		)
	} else {
		global.Logger.Warn("数据库配置为空，无法进行表存在性检查", zap.String("instance_id", req.InstanceID))
	}
	
	results, err := chk.Check(req.Content)
	if err != nil {
		// SQL 语法解析错误，HTTP 200 但 code=1
		api.HandleError(c, http.StatusOK, fmt.Errorf("SQL语法错误: %s", err.Error()), []*checker.AuditResult{
			{
				SQL:           req.Content,
				Type:          "ERROR",
				Level:         checker.LevelError,
				AffectedRows:  0,
				Messages:      []string{err.Error()},
				FixSuggestion: "",
			},
		})
		return
	}

	// 检查审核结果是否有错误（与老服务逻辑一致）
	// status: 0表示语法检查通过，1表示语法检查不通过
	status := 0
	for _, r := range results {
		if r.Level != checker.LevelInfo {
			status = 1
			break
		}
	}

	// 返回格式与老服务保持一致：{status: 0/1, data: [...]}
	api.HandleSuccess(c, gin.H{
		"status": status,
		"data":   results,
	})
}

// GetInspectParams 获取审核参数列表
// @Summary 获取审核参数列表
// @Tags SQL审核
// @Security Bearer
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/inspect/params [get]
func (h *InspectHandler) GetInspectParams(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var params []insight.InspectParams
	var total int64

	query := global.DB.Model(&insight.InspectParams{})
	query.Count(&total)

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&params).Error; err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	api.HandleSuccess(c, gin.H{
		"list":  params,
		"total": total,
	})
}

// GetInspectParam 获取审核参数详情
// @Summary 获取审核参数详情
// @Tags SQL审核
// @Security Bearer
// @Accept json
// @Produce json
// @Param id path int true "参数ID"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/inspect/params/{id} [get]
func (h *InspectHandler) GetInspectParam(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		api.HandleError(c, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}

	var param insight.InspectParams
	if err := global.DB.First(&param, id).Error; err != nil {
		api.HandleError(c, http.StatusNotFound, err, nil)
		return
	}

	api.HandleSuccess(c, param)
}

// CreateInspectParamsRequest 创建审核参数请求
type CreateInspectParamsRequest struct {
	Params map[string]interface{} `json:"params" binding:"required"`
	Remark string                 `json:"remark" binding:"required"`
}

// CreateInspectParams 创建审核参数
// @Summary 创建审核参数
// @Tags SQL审核
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body CreateInspectParamsRequest true "参数信息"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/inspect/params [post]
func (h *InspectHandler) CreateInspectParams(c *gin.Context) {
	var req CreateInspectParamsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.HandleError(c, http.StatusBadRequest, err, nil)
		return
	}

	paramsJSON, err := json.Marshal(req.Params)
	if err != nil {
		api.HandleError(c, http.StatusBadRequest, err, nil)
		return
	}

	param := &insight.InspectParams{
		Params: paramsJSON,
		Remark: req.Remark,
	}

	if err := global.DB.Create(param).Error; err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	api.HandleSuccess(c, param)
}

// UpdateInspectParamsRequest 更新审核参数请求
type UpdateInspectParamsRequest struct {
	Params map[string]interface{} `json:"params" binding:"required"`
	Remark string                 `json:"remark"`
}

// UpdateInspectParams 更新审核参数
// @Summary 更新审核参数
// @Tags SQL审核
// @Security Bearer
// @Accept json
// @Produce json
// @Param id path int true "参数ID"
// @Param request body UpdateInspectParamsRequest true "参数信息"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/inspect/params/{id} [put]
func (h *InspectHandler) UpdateInspectParams(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		api.HandleError(c, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}

	var req UpdateInspectParamsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.HandleError(c, http.StatusBadRequest, err, nil)
		return
	}

	paramsJSON, err := json.Marshal(req.Params)
	if err != nil {
		api.HandleError(c, http.StatusBadRequest, err, nil)
		return
	}

	updates := map[string]interface{}{
		"params": paramsJSON,
	}
	if req.Remark != "" {
		updates["remark"] = req.Remark
	}

	if err := global.DB.Model(&insight.InspectParams{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	api.HandleSuccess(c, nil)
}

// DeleteInspectParams 删除审核参数
// @Summary 删除审核参数
// @Tags SQL审核
// @Security Bearer
// @Accept json
// @Produce json
// @Param id path int true "参数ID"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/inspect/params/{id} [delete]
func (h *InspectHandler) DeleteInspectParams(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		api.HandleError(c, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}

	if err := global.DB.Delete(&insight.InspectParams{}, id).Error; err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	api.HandleSuccess(c, nil)
}

// GetDefaultInspectParams 获取默认审核参数
// @Summary 获取默认审核参数
// @Tags SQL审核
// @Security Bearer
// @Accept json
// @Produce json
// @Success 200 {object} api.Response
// @Router /api/v1/insight/inspect/params/default [get]
func (h *InspectHandler) GetDefaultInspectParams(c *gin.Context) {
	api.HandleSuccess(c, config.DefaultInspectParams())
}

