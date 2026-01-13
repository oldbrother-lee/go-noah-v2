package insight

import (
	"context"
	"encoding/json"
	"fmt"
	"go-noah/api"
	"go-noah/internal/handler"
	"go-noah/internal/inspect/parser"
	"go-noah/internal/model"
	"go-noah/internal/model/insight"
	"go-noah/internal/orders/executor"
	insightRepo "go-noah/internal/repository/insight"
	"go-noah/internal/service"
	"go-noah/pkg/global"
	"go-noah/pkg/utils"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// OrderHandlerApp 全局 Handler 实例
var OrderHandlerApp = new(OrderHandler)

// OrderHandler 工单管理 Handler
type OrderHandler struct{}

// GetOrdersRequest 获取工单列表请求
type GetOrdersRequest struct {
	Page        int    `form:"current"` // 前端用 current
	PageSize    int    `form:"size"`    // 前端用 size
	OnlyMyOrder int    `form:"only_my_orders"`
	Applicant   string `form:"applicant"`
	Progress    string `form:"progress"`
	Environment int    `form:"environment"`
	SQLType     string `form:"sql_type"`
	DBType      string `form:"db_type"`
	Title       string `form:"title"`
}

// GetOrders 获取工单列表
// @Summary 获取工单列表
// @Tags 工单管理
// @Security Bearer
// @Accept json
// @Produce json
// @Param page query int true "页码"
// @Param page_size query int true "每页数量"
// @Param applicant query string false "申请人"
// @Param progress query string false "进度"
// @Param environment query int false "环境"
// @Param sql_type query string false "SQL类型"
// @Param db_type query string false "DB类型"
// @Param title query string false "标题"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/orders [get]
func (h *OrderHandler) GetOrders(c *gin.Context) {
	var req GetOrdersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.HandleError(c, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}

	// 设置默认分页值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}

	// 如果 only_my_orders=1，获取当前用户名
	var applicant string
	if req.OnlyMyOrder == 1 {
		userId := handler.GetUserIdFromCtx(c)
		if userId > 0 {
			user, err := service.AdminServiceApp.GetAdminUser(c.Request.Context(), userId)
			if err == nil && user != nil {
				applicant = user.Username
			}
		}
	} else {
		applicant = req.Applicant
	}

	params := &insightRepo.OrderQueryParams{
		Page:        req.Page,
		PageSize:    req.PageSize,
		Applicant:   applicant,
		Progress:    req.Progress,
		Environment: req.Environment,
		SQLType:     req.SQLType,
		DBType:      req.DBType,
		Title:       req.Title,
	}

	orders, total, err := service.InsightServiceApp.GetOrders(c.Request.Context(), params)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	api.HandleSuccess(c, gin.H{
		"list":  orders,
		"total": total,
	})
}

// GetOrder 获取工单详情
// @Summary 获取工单详情
// @Tags 工单管理
// @Security Bearer
// @Accept json
// @Produce json
// @Param order_id path string true "工单ID"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/orders/{order_id} [get]
func (h *OrderHandler) GetOrder(c *gin.Context) {
	orderID := c.Param("order_id")
	if orderID == "" {
		api.HandleError(c, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}

	order, err := service.InsightServiceApp.GetOrderByID(c.Request.Context(), orderID)
	if err != nil {
		api.HandleError(c, http.StatusNotFound, err, nil)
		return
	}

	// 获取任务列表
	tasks, err := service.InsightServiceApp.GetOrderTasks(c.Request.Context(), orderID)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	// 获取操作日志
	logs, err := service.InsightServiceApp.GetOpLogs(c.Request.Context(), orderID)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	api.HandleSuccess(c, gin.H{
		"order": order,
		"tasks": tasks,
		"logs":  logs,
	})
}

// CreateOrderRequest 创建工单请求
type CreateOrderRequest struct {
	Title            string     `json:"title" binding:"required"`
	Remark           string     `json:"remark"`
	IsRestrictAccess bool       `json:"is_restrict_access"`
	DBType           string     `json:"db_type" binding:"required"`
	SQLType          string     `json:"sql_type" binding:"required"`
	Environment      int        `json:"environment" binding:"required"`
	InstanceID       string     `json:"instance_id" binding:"required"`
	Schema           string     `json:"schema" binding:"required"`
	Content          string     `json:"content" binding:"required"`
	Approver         []string   `json:"approver"`
	Executor         []string   `json:"executor"`
	Reviewer         []string   `json:"reviewer"`
	CC               []string   `json:"cc"`
	ScheduleTime     *time.Time `json:"schedule_time"`
	FixVersion       string     `json:"fix_version"`
	ExportFileFormat string     `json:"export_file_format"`
}

// CreateOrder 创建工单
// @Summary 创建工单
// @Tags 工单管理
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body CreateOrderRequest true "工单信息"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/orders [post]
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.HandleError(c, http.StatusBadRequest, err, nil)
		return
	}

	// 获取当前用户
	userId := handler.GetUserIdFromCtx(c)
	username := ""
	if userId > 0 {
		user, err := service.AdminServiceApp.GetAdminUser(c, userId)
		if err == nil {
			username = user.Username
		}
	}

	// 解析 InstanceID
	instanceUUID, err := uuid.Parse(req.InstanceID)
	if err != nil {
		api.HandleError(c, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}

	order := &insight.OrderRecord{
		Title:            req.Title,
		Remark:           req.Remark,
		IsRestrictAccess: req.IsRestrictAccess,
		DBType:           insight.DbType(req.DBType),
		SQLType:          insight.SQLType(req.SQLType),
		Environment:      req.Environment,
		InstanceID:       instanceUUID,
		Schema:           req.Schema,
		Content:          req.Content,
		Applicant:        username,
		Progress:         insight.ProgressPending,
		ScheduleTime:     req.ScheduleTime,
		FixVersion:       req.FixVersion,
		ExportFileFormat: insight.ExportFileFormat(req.ExportFileFormat),
	}

	// 转换 JSON 字段
	if len(req.Approver) > 0 {
		order.Approver, _ = jsonMarshal(req.Approver)
	}
	if len(req.Executor) > 0 {
		order.Executor, _ = jsonMarshal(req.Executor)
	}
	if len(req.Reviewer) > 0 {
		order.Reviewer, _ = jsonMarshal(req.Reviewer)
	}
	if len(req.CC) > 0 {
		order.CC, _ = jsonMarshal(req.CC)
	}

	if err := service.InsightServiceApp.CreateOrder(c.Request.Context(), order); err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	// 记录操作日志
	_ = service.InsightServiceApp.CreateOpLog(c.Request.Context(), &insight.OrderOpLog{
		Username: username,
		OrderID:  order.OrderID,
		Msg:      "创建工单",
	})

	api.HandleSuccess(c, order)
}

// UpdateOrderProgressRequest 更新工单进度请求
type UpdateOrderProgressRequest struct {
	OrderID  string `json:"order_id" binding:"required"`
	Progress string `json:"progress" binding:"required"`
	Remark   string `json:"remark"`
}

// UpdateOrderProgress 更新工单进度
// @Summary 更新工单进度
// @Tags 工单管理
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body UpdateOrderProgressRequest true "进度信息"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/orders/progress [put]
func (h *OrderHandler) UpdateOrderProgress(c *gin.Context) {
	var req UpdateOrderProgressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.HandleError(c, http.StatusBadRequest, err, nil)
		return
	}

	// 获取当前用户
	userId := handler.GetUserIdFromCtx(c)
	username := ""
	if userId > 0 {
		user, err := service.AdminServiceApp.GetAdminUser(c, userId)
		if err == nil {
			username = user.Username
		}
	}

	// 获取工单
	order, err := service.InsightServiceApp.GetOrderByID(c.Request.Context(), req.OrderID)
	if err != nil {
		api.HandleError(c, http.StatusNotFound, err, nil)
		return
	}

	// 更新进度
	if err := service.InsightServiceApp.UpdateOrderProgress(c.Request.Context(), req.OrderID, insight.Progress(req.Progress)); err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	// 记录操作日志
	msg := "更新工单进度为: " + req.Progress
	if req.Remark != "" {
		msg += ", 备注: " + req.Remark
	}
	_ = service.InsightServiceApp.CreateOpLog(c.Request.Context(), &insight.OrderOpLog{
		Username: username,
		OrderID:  order.OrderID,
		Msg:      msg,
	})

	api.HandleSuccess(c, nil)
}

// ApproveOrderRequest 审批工单请求
type ApproveOrderRequest struct {
	OrderID string `json:"order_id" binding:"required"`
	Status  string `json:"status" binding:"required,oneof=pass reject"` // pass: 通过, reject: 驳回
	Msg     string `json:"msg"`                                         // 审批意见
}

// ApproveOrder 审批工单
// @Summary 审批工单
// @Tags 工单管理
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body ApproveOrderRequest true "审批信息"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/orders/approve [post]
func (h *OrderHandler) ApproveOrder(c *gin.Context) {
	var req ApproveOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.HandleError(c, http.StatusBadRequest, err, nil)
		return
	}

	// 获取当前用户
	userId := handler.GetUserIdFromCtx(c)
	username := ""
	if userId > 0 {
		user, err := service.AdminServiceApp.GetAdminUser(c, userId)
		if err == nil {
			username = user.Username
		}
	}

	// 获取工单
	order, err := service.InsightServiceApp.GetOrderByID(c.Request.Context(), req.OrderID)
	if err != nil {
		api.HandleError(c, http.StatusNotFound, err, nil)
		return
	}

	// 检查工单状态
	if order.Progress != insight.ProgressPending {
		api.HandleError(c, http.StatusBadRequest, api.ErrBadRequest, "工单状态不允许审批")
		return
	}

	// 解析审批人列表
	var approverList []map[string]interface{}
	if order.Approver != nil {
		if err := json.Unmarshal(order.Approver, &approverList); err != nil {
			api.HandleError(c, http.StatusBadRequest, api.ErrBadRequest, "解析审批人列表失败")
			return
		}
	}

	// 检查当前用户是否在审批人列表中
	var foundApprover bool
	var passCount int
	for i, approver := range approverList {
		userValue, ok := approver["user"].(string)
		if !ok {
			// 兼容性处理：如果是其他类型，尝试转换
			continue
		}
		if userValue == username {
			foundApprover = true
			// 检查是否已审核过
			if status, ok := approver["status"].(string); ok && status != "pending" {
				api.HandleError(c, http.StatusBadRequest, api.ErrBadRequest, "您已审核过，请不要重复执行")
				return
			}
			// 更新当前审批人的状态
			approverList[i]["status"] = req.Status
			approverList[i]["user"] = username
		}
		// 统计已通过的审核人数
		if status, ok := approver["status"].(string); ok && status == "pass" {
			passCount++
		}
	}

	if !foundApprover {
		// 检查是否是 admin 用户
		isAdmin := userId == 1
		if !isAdmin {
			uidStr := strconv.FormatUint(uint64(userId), 10)
			roles, err := global.Enforcer.GetRolesForUser(uidStr)
			if err == nil {
				for _, role := range roles {
					if role == model.AdminRole {
						isAdmin = true
						break
					}
				}
			}
		}
		if !isAdmin {
			api.HandleError(c, http.StatusForbidden, api.ErrForbidden, "您没有当前工单的审核权限")
			return
		}
		// Admin 用户可以审核，但不添加到审批人列表（因为不在列表中）
		// 对于这种情况，如果只有一个审批人且是 admin，直接通过
		if len(approverList) == 0 {
			// 没有审批人列表，admin 可以直接审批通过
			passCount = 1
		}
	}

	// 如果驳回，直接更新工单状态
	if req.Status == "reject" {
		if err := service.InsightServiceApp.UpdateOrderProgress(c.Request.Context(), req.OrderID, insight.ProgressRejected); err != nil {
			api.HandleError(c, http.StatusInternalServerError, err, nil)
			return
		}
		// 更新审批人列表（如果当前用户在列表中）
		if foundApprover {
			approverJSON, _ := json.Marshal(approverList)
			order.Approver = approverJSON
			_ = service.InsightServiceApp.UpdateOrder(c.Request.Context(), &order.OrderRecord)
		}
		// 记录操作日志
		_ = service.InsightServiceApp.CreateOpLog(c.Request.Context(), &insight.OrderOpLog{
			Username: username,
			OrderID:  order.OrderID,
			Msg:      fmt.Sprintf("用户%s驳回了工单，审批意见：%s", username, req.Msg),
		})
		api.HandleSuccess(c, nil)
		return
	}

	// 更新审批人列表（如果是 pass 状态且当前用户在列表中）
	if foundApprover {
		// 重新统计通过人数（包含当前审批人）
		passCount = 0
		for _, approver := range approverList {
			if status, ok := approver["status"].(string); ok && status == "pass" {
				passCount++
			}
		}
		approverJSON, _ := json.Marshal(approverList)
		order.Approver = approverJSON
		_ = service.InsightServiceApp.UpdateOrder(c.Request.Context(), &order.OrderRecord)
	}

	// 检查是否所有审核人都通过了
	allApproved := false
	if foundApprover {
		// 如果当前用户在审批人列表中，检查所有审批人是否都通过
		allApproved = len(approverList) == passCount
	} else {
		// 如果当前用户不在审批人列表中（但可能是 admin），检查所有审批人是否都通过
		// 这种情况下，如果审批人列表为空，admin 可以直接通过
		if len(approverList) == 0 {
			allApproved = true
		} else {
			allApproved = len(approverList) == passCount
		}
	}

	// 如果所有审核人都通过，更新工单状态为"已批准"并自动生成任务
	if allApproved {
		if err := service.InsightServiceApp.UpdateOrderProgress(c.Request.Context(), req.OrderID, insight.ProgressApproved); err != nil {
			api.HandleError(c, http.StatusInternalServerError, err, nil)
			return
		}
		// UpdateOrderProgress 内部会自动生成任务，这里不需要再次调用
		// 记录操作日志
		_ = service.InsightServiceApp.CreateOpLog(c.Request.Context(), &insight.OrderOpLog{
			Username: username,
			OrderID:  order.OrderID,
			Msg:      fmt.Sprintf("用户%s审核通过了工单，审批意见：%s", username, req.Msg),
		})
		api.HandleSuccess(c, nil)
		return
	}

	// 部分审核人通过，记录操作日志但不更新工单状态
	_ = service.InsightServiceApp.CreateOpLog(c.Request.Context(), &insight.OrderOpLog{
		Username: username,
		OrderID:  order.OrderID,
		Msg:      fmt.Sprintf("用户%s审核通过了工单（等待其他审核人），审批意见：%s", username, req.Msg),
	})
	api.HandleSuccess(c, nil)
}

// GetOrderTasks 获取工单任务列表
// @Summary 获取工单任务列表
// @Tags 工单管理
// @Security Bearer
// @Accept json
// @Produce json
// @Param order_id path string true "工单ID"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/orders/{order_id}/tasks [get]
func (h *OrderHandler) GetOrderTasks(c *gin.Context) {
	orderID := c.Param("order_id")
	if orderID == "" {
		api.HandleError(c, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}

	tasks, err := service.InsightServiceApp.GetOrderTasks(c.Request.Context(), orderID)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	api.HandleSuccess(c, tasks)
}

// UpdateTaskProgressRequest 更新任务进度请求
type UpdateTaskProgressRequest struct {
	TaskID   string `json:"task_id" binding:"required"`
	Progress string `json:"progress" binding:"required"`
}

// UpdateTaskProgress 更新任务进度
// @Summary 更新任务进度
// @Tags 工单管理
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body UpdateTaskProgressRequest true "进度信息"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/orders/tasks/progress [put]
func (h *OrderHandler) UpdateTaskProgress(c *gin.Context) {
	var req UpdateTaskProgressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.HandleError(c, http.StatusBadRequest, err, nil)
		return
	}

	if err := service.InsightServiceApp.UpdateTaskProgress(c.Request.Context(), req.TaskID, insight.TaskProgress(req.Progress), nil); err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	api.HandleSuccess(c, nil)
}

// GetOrderLogs 获取工单操作日志
// @Summary 获取工单操作日志
// @Tags 工单管理
// @Security Bearer
// @Accept json
// @Produce json
// @Param order_id path string true "工单ID"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/orders/{order_id}/logs [get]
func (h *OrderHandler) GetOrderLogs(c *gin.Context) {
	orderID := c.Param("order_id")
	if orderID == "" {
		api.HandleError(c, http.StatusBadRequest, api.ErrBadRequest, nil)
		return
	}

	logs, err := service.InsightServiceApp.GetOpLogs(c.Request.Context(), orderID)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	api.HandleSuccess(c, logs)
}

// GetMyOrders 获取我的工单
// @Summary 获取我的工单
// @Tags 工单管理
// @Security Bearer
// @Accept json
// @Produce json
// @Param page query int true "页码"
// @Param page_size query int true "每页数量"
// @Param progress query string false "进度"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/orders/my [get]
func (h *OrderHandler) GetMyOrders(c *gin.Context) {
	userId := handler.GetUserIdFromCtx(c)
	if userId == 0 {
		api.HandleError(c, http.StatusUnauthorized, api.ErrUnauthorized, nil)
		return
	}

	username := ""
	user, err := service.AdminServiceApp.GetAdminUser(c, userId)
	if err == nil {
		username = user.Username
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	progress := c.Query("progress")

	params := &insightRepo.OrderQueryParams{
		Page:      page,
		PageSize:  pageSize,
		Applicant: username,
		Progress:  progress,
	}

	orders, total, err := service.InsightServiceApp.GetOrders(c.Request.Context(), params)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	api.HandleSuccess(c, gin.H{
		"list":  orders,
		"total": total,
	})
}

// jsonMarshal JSON序列化辅助函数
func jsonMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// ExecuteTaskRequest 执行任务请求
type ExecuteTaskRequest struct {
	TaskID  string `json:"task_id"`  // 执行单个任务时使用
	OrderID string `json:"order_id"` // 执行全部任务时使用
}

// ExecuteTask 执行工单任务（支持单个任务和全部任务）
// @Summary 执行工单任务
// @Tags 工单管理
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body ExecuteTaskRequest true "任务信息"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/orders/tasks/execute [post]
func (h *OrderHandler) ExecuteTask(c *gin.Context) {
	var req ExecuteTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.HandleError(c, http.StatusBadRequest, err, nil)
		return
	}

	// 获取当前用户
	userId := handler.GetUserIdFromCtx(c)
	username := ""
	if userId > 0 {
		user, err := service.AdminServiceApp.GetAdminUser(c, userId)
		if err == nil {
			username = user.Username
		}
	}

	// 如果提供了 order_id，执行全部任务
	if req.OrderID != "" {
		h.executeAllTasks(c, req.OrderID, username, userId)
		return
	}

	// 如果提供了 task_id，执行单个任务
	if req.TaskID == "" {
		api.HandleError(c, http.StatusBadRequest, api.ErrBadRequest, "task_id 或 order_id 必须提供其中一个")
		return
	}

	// 获取任务信息
	task, err := service.InsightServiceApp.GetTaskByID(c.Request.Context(), req.TaskID)
	if err != nil {
		api.HandleError(c, http.StatusNotFound, err, nil)
		return
	}

	// 获取工单信息
	order, err := service.InsightServiceApp.GetOrderByID(c.Request.Context(), task.OrderID.String())
	if err != nil {
		api.HandleError(c, http.StatusNotFound, err, nil)
		return
	}

	// 检查执行权限
	if err := h.checkOrderStatus(c.Request.Context(), task.OrderID.String(), username, userId); err != nil {
		api.HandleError(c, http.StatusForbidden, err, nil)
		return
	}

	// 检查任务状态（避免重复执行）
	if task.Progress == insight.TaskProgressCompleted {
		api.HandleError(c, http.StatusBadRequest, api.ErrBadRequest, "当前任务已完成，请勿重复执行")
		return
	}
	if task.Progress == insight.TaskProgressExecuting {
		api.HandleError(c, http.StatusBadRequest, api.ErrBadRequest, "当前任务正在执行中，请勿重复执行")
		return
	}

	// 检查是否有其他任务正在执行中
	noExecutingTasks, err := service.InsightServiceApp.CheckTasksProgressIsDoing(c.Request.Context(), task.OrderID.String())
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}
	if !noExecutingTasks {
		api.HandleError(c, http.StatusBadRequest, api.ErrBadRequest, "当前有任务正在执行中，请先等待执行完成")
		return
	}

	// 获取数据库配置
	dbConfig, err := service.InsightServiceApp.GetDBConfigByInstanceID(c.Request.Context(), order.InstanceID.String())
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	// 使用事务同时更新任务状态和工单状态
	err = service.InsightServiceApp.UpdateTaskAndOrderProgress(c.Request.Context(), req.TaskID, task.OrderID.String(), insight.TaskProgressExecuting, insight.ProgressExecuting)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	// 记录操作日志
	_ = service.InsightServiceApp.CreateOpLog(c.Request.Context(), &insight.OrderOpLog{
		Username: username,
		OrderID:  order.OrderID,
		Msg:      "开始执行任务: " + req.TaskID,
	})

	// 创建执行器配置
	execConfig := &executor.DBConfig{
		Hostname:         dbConfig.Hostname,
		Port:             dbConfig.Port,
		UserName:         dbConfig.UserName,
		Password:         dbConfig.Password,
		Schema:           order.Schema,
		DBType:           string(dbConfig.DbType),
		SQLType:          string(task.SQLType),
		SQL:              task.SQL,
		OrderID:          order.OrderID.String(),
		TaskID:           task.TaskID.String(),
		ExportFileFormat: string(order.ExportFileFormat),
	}

	// 记录执行开始日志（用于调试）
	global.Logger.Info("Starting task execution",
		zap.String("order_id", order.OrderID.String()),
		zap.String("task_id", task.TaskID.String()),
		zap.String("sql_type", string(task.SQLType)),
	)

	// 创建执行器
	exec, err := executor.NewExecuteSQL(execConfig)
	if err != nil {
		_ = service.InsightServiceApp.UpdateTaskProgress(c.Request.Context(), req.TaskID, insight.TaskProgressFailed, nil)
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	// 执行SQL
	result, err := exec.Run()

	// 保存执行结果
	resultJSON, _ := json.Marshal(result)
	if err != nil {
		_ = service.InsightServiceApp.UpdateTaskProgress(c.Request.Context(), req.TaskID, insight.TaskProgressFailed, resultJSON)
		_ = service.InsightServiceApp.CreateOpLog(c.Request.Context(), &insight.OrderOpLog{
			Username: username,
			OrderID:  order.OrderID,
			Msg:      "任务执行失败: " + err.Error(),
		})
		api.HandleError(c, http.StatusInternalServerError, err, result.ExecuteLog)
		return
	}

	// 更新任务状态为完成
	_ = service.InsightServiceApp.UpdateTaskProgress(c.Request.Context(), req.TaskID, insight.TaskProgressCompleted, resultJSON)
	_ = service.InsightServiceApp.CreateOpLog(c.Request.Context(), &insight.OrderOpLog{
		Username: username,
		OrderID:  order.OrderID,
		Msg:      "任务执行成功，影响行数: " + strconv.FormatInt(result.AffectedRows, 10),
	})

	// 更新工单状态为已完成（如果所有任务都完成）
	h.updateOrderStatusToFinish(c.Request.Context(), order.OrderID.String())

	api.HandleSuccess(c, result)
}

// checkOrderStatus 检查工单状态和执行权限
func (h *OrderHandler) checkOrderStatus(ctx context.Context, orderID string, username string, userID uint) error {
	order, err := service.InsightServiceApp.GetOrderByID(ctx, orderID)
	if err != nil {
		return err
	}

	// 检查用户是否是 admin（用户ID=1 或拥有 admin 角色）
	isAdmin := false
	if userID == 1 {
		// 用户ID=1，是超级管理员
		isAdmin = true
	} else {
		// 检查用户是否有 admin 角色
		roles, err := global.Enforcer.GetRolesForUser(string(rune(userID)))
		if err == nil {
			for _, role := range roles {
				if role == model.AdminRole {
					isAdmin = true
					break
				}
			}
		}
	}

	// 如果不是 admin，检查执行权限
	if !isAdmin {
		var executorList []string
		if order.Executor != nil {
			if err := json.Unmarshal(order.Executor, &executorList); err != nil {
				return err
			}
		}
		// 如果 executor 列表不为空且当前用户不在列表中，则拒绝
		if len(executorList) > 0 && !utils.IsContain(executorList, username) {
			return api.ErrForbidden
		}
	}

	// 检查工单状态
	if order.Progress != insight.ProgressApproved && order.Progress != insight.ProgressExecuting {
		return api.ErrBadRequest
	}

	return nil
}

// updateOrderStatusToFinish 检查所有任务是否完成，如果完成则更新工单状态
func (h *OrderHandler) updateOrderStatusToFinish(ctx context.Context, orderID string) {
	allCompleted, err := service.InsightServiceApp.CheckAllTasksCompleted(ctx, orderID)
	if err != nil || !allCompleted {
		return
	}

	// 更新工单状态为已完成
	_ = service.InsightServiceApp.UpdateOrderProgress(ctx, orderID, insight.ProgressCompleted)

	// TODO: 发送通知消息给申请人（可选功能）
}

// executeAllTasks 执行工单的所有任务
func (h *OrderHandler) executeAllTasks(c *gin.Context, orderID string, username string, userID uint) {
	// 获取工单信息
	order, err := service.InsightServiceApp.GetOrderByID(c.Request.Context(), orderID)
	if err != nil {
		api.HandleError(c, http.StatusNotFound, err, nil)
		return
	}

	// 检查执行权限和工单状态
	if err := h.checkOrderStatus(c.Request.Context(), orderID, username, userID); err != nil {
		api.HandleError(c, http.StatusForbidden, err, nil)
		return
	}

	// 检查是否有任务正在执行中
	noExecutingTasks, err := service.InsightServiceApp.CheckTasksProgressIsDoing(c.Request.Context(), orderID)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}
	if !noExecutingTasks {
		api.HandleError(c, http.StatusBadRequest, api.ErrBadRequest, "当前有任务正在执行中，请先等待执行完成")
		return
	}

	// 检查是否有已暂停的任务
	noPausedTasks, err := service.InsightServiceApp.CheckTasksProgressIsPause(c.Request.Context(), orderID)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}
	if !noPausedTasks {
		api.HandleError(c, http.StatusBadRequest, api.ErrBadRequest, "当前有任务已暂停，可手动执行单个任务")
		return
	}

	// 获取工单的所有任务
	tasks, err := service.InsightServiceApp.GetOrderTasks(c.Request.Context(), orderID)
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	if len(tasks) == 0 {
		api.HandleSuccess(c, gin.H{
			"type":    "warning",
			"message": "没有需要执行的任务",
		})
		return
	}

	// 更新工单状态为执行中
	_ = service.InsightServiceApp.UpdateOrderProgress(c.Request.Context(), orderID, insight.ProgressExecuting)

	// 记录操作日志
	_ = service.InsightServiceApp.CreateOpLog(c.Request.Context(), &insight.OrderOpLog{
		Username: username,
		OrderID:  order.OrderID,
		Msg:      "开始批量执行任务",
	})

	var executedCount, successCount, failCount int
	var msgResult, typeResult string

	// 获取数据库配置
	dbConfig, err := service.InsightServiceApp.GetDBConfigByInstanceID(c.Request.Context(), order.InstanceID.String())
	if err != nil {
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	// 逐个执行任务
	for _, task := range tasks {
		// 跳过已完成的任务
		if task.Progress == insight.TaskProgressCompleted {
			continue
		}

		executedCount++

		// 更新任务状态为执行中
		_ = service.InsightServiceApp.UpdateTaskProgress(c.Request.Context(), task.TaskID.String(), insight.TaskProgressExecuting, nil)

		// 创建执行器配置
		execConfig := &executor.DBConfig{
			Hostname:         dbConfig.Hostname,
			Port:             dbConfig.Port,
			UserName:         dbConfig.UserName,
			Password:         dbConfig.Password,
			Schema:           order.Schema,
			DBType:           string(dbConfig.DbType),
			SQLType:          string(task.SQLType),
			SQL:              task.SQL,
			OrderID:          order.OrderID.String(),
			TaskID:           task.TaskID.String(),
			ExportFileFormat: string(order.ExportFileFormat),
		}

		// 创建执行器
		exec, err := executor.NewExecuteSQL(execConfig)
		if err != nil {
			failCount++
			resultJSON, _ := json.Marshal(map[string]interface{}{"error": err.Error()})
			_ = service.InsightServiceApp.UpdateTaskProgress(c.Request.Context(), task.TaskID.String(), insight.TaskProgressFailed, resultJSON)
			_ = service.InsightServiceApp.CreateOpLog(c.Request.Context(), &insight.OrderOpLog{
				Username: username,
				OrderID:  order.OrderID,
				Msg:      "任务执行失败: " + err.Error(),
			})
			continue
		}

		// 执行SQL
		result, err := exec.Run()

		// 保存执行结果
		resultJSON, _ := json.Marshal(result)
		if err != nil {
			failCount++
			_ = service.InsightServiceApp.UpdateTaskProgress(c.Request.Context(), task.TaskID.String(), insight.TaskProgressFailed, resultJSON)
			_ = service.InsightServiceApp.CreateOpLog(c.Request.Context(), &insight.OrderOpLog{
				Username: username,
				OrderID:  order.OrderID,
				Msg:      "任务执行失败: " + err.Error(),
			})
		} else {
			successCount++
			_ = service.InsightServiceApp.UpdateTaskProgress(c.Request.Context(), task.TaskID.String(), insight.TaskProgressCompleted, resultJSON)
			_ = service.InsightServiceApp.CreateOpLog(c.Request.Context(), &insight.OrderOpLog{
				Username: username,
				OrderID:  order.OrderID,
				Msg:      "任务执行成功，影响行数: " + strconv.FormatInt(result.AffectedRows, 10),
			})
		}
	}

	// 根据执行结果确定返回消息
	if executedCount == 0 {
		msgResult = "没有需要执行的任务"
		typeResult = "warning"
	} else if successCount == executedCount {
		msgResult = "执行成功"
		typeResult = "success"
	} else if failCount == executedCount {
		msgResult = "执行失败"
		typeResult = "error"
	} else {
		msgResult = "执行有失败，请关注执行结果"
		typeResult = "warning"
	}

	// 更新工单执行结果
	_ = service.InsightServiceApp.UpdateOrderExecuteResult(c.Request.Context(), orderID, typeResult)

	// 检查所有任务是否完成，如果完成则更新工单状态
	h.updateOrderStatusToFinish(c.Request.Context(), orderID)

	api.HandleSuccess(c, gin.H{
		"type":    typeResult,
		"message": msgResult,
	})
}

// ControlGhostRequest gh-ost 控制请求
type ControlGhostRequest struct {
	OrderID string `json:"order_id" binding:"required"` // 工单ID
	Action  string `json:"action" binding:"required"`   // 操作类型：throttle(暂停), unthrottle(恢复), panic(取消), chunk-size(调节速度)
	Value   *int   `json:"value,omitempty"`             // 操作值（仅用于 chunk-size，表示新的 chunk-size 值）
}

// ControlGhost 控制 gh-ost 执行
// @Summary 控制 gh-ost 执行（暂停/取消/速度调节）
// @Tags 工单管理
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body ControlGhostRequest true "控制请求"
// @Success 200 {object} api.Response
// @Router /api/v1/insight/orders/ghost/control [post]
func (h *OrderHandler) ControlGhost(c *gin.Context) {
	var req ControlGhostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.HandleError(c, http.StatusBadRequest, err, nil)
		return
	}

	// 验证操作类型
	validActions := map[string]bool{
		"throttle":   true, // 暂停
		"unthrottle": true, // 恢复
		"panic":      true, // 取消
		"chunk-size": true, // 调节速度
	}
	if !validActions[req.Action] {
		api.HandleError(c, http.StatusBadRequest, api.ErrBadRequest, "不支持的操作类型，支持的操作：throttle(暂停), unthrottle(恢复), panic(取消), chunk-size(调节速度)")
		return
	}

	// chunk-size 操作需要提供 value
	if req.Action == "chunk-size" {
		if req.Value == nil || *req.Value <= 0 {
			api.HandleError(c, http.StatusBadRequest, api.ErrBadRequest, "chunk-size 操作需要提供有效的 value 值（大于 0）")
			return
		}
	}

	// 获取 socket 路径
	// 首先尝试从 Redis 获取
	socketPath, err := utils.GetGhostSocketPathFromOrderID(req.OrderID, "", "")
	if err != nil {
		// Redis 中没有，尝试从工单信息推断
		global.Logger.Warn("Failed to get ghost socket path from Redis, trying to infer from order",
			zap.String("order_id", req.OrderID),
			zap.Error(err),
		)

		// 获取工单信息
		order, err := service.InsightServiceApp.GetOrderByID(c.Request.Context(), req.OrderID)
		if err != nil {
			global.Logger.Error("Failed to get order info",
				zap.String("order_id", req.OrderID),
				zap.Error(err),
			)
			api.HandleError(c, http.StatusOK, fmt.Errorf("任务未找到或者未执行"), nil)
			return
		}

		// 从 SQL 内容中提取表名（如果是 ALTER TABLE 语句）
		var tableName string
		if order.SQLType == insight.SQLTypeDDL && order.Content != "" {
			// 尝试从 SQL 内容中提取表名
			extractedTable, err := parser.GetTableNameFromAlterStatement(order.Content)
			if err == nil {
				// 如果提取成功，处理可能的 schema.table 格式
				if strings.Contains(extractedTable, ".") {
					parts := strings.SplitN(extractedTable, ".", 2)
					tableName = strings.Trim(parts[1], "`")
				} else {
					tableName = strings.Trim(extractedTable, "`")
				}
			}
		}

		// 使用工单的 schema 和提取的表名推断 socket 路径
		if order.Schema != "" && tableName != "" {
			socketPath, err = utils.GetGhostSocketPathFromOrderID(req.OrderID, order.Schema, tableName)
			if err != nil {
				global.Logger.Error("Failed to get ghost socket path (inferred)",
					zap.String("order_id", req.OrderID),
					zap.String("schema", order.Schema),
					zap.String("table", tableName),
					zap.Error(err),
				)
				api.HandleError(c, http.StatusOK, fmt.Errorf("任务未找到或者未执行"), nil)
				return
			}
		} else {
			global.Logger.Error("Cannot infer socket path: missing schema or table",
				zap.String("order_id", req.OrderID),
				zap.String("schema", order.Schema),
				zap.String("table", tableName),
			)
			api.HandleError(c, http.StatusOK, fmt.Errorf("任务未找到或者未执行"), nil)
			return
		}
	}

	// 构建命令
	var command string
	switch req.Action {
	case "throttle":
		command = "throttle"
	case "unthrottle":
		command = "unthrottle"
	case "panic":
		command = "panic"
	case "chunk-size":
		command = fmt.Sprintf("chunk-size=%d", *req.Value)
	default:
		api.HandleError(c, http.StatusBadRequest, api.ErrBadRequest, "不支持的操作类型")
		return
	}

	// 发送命令给 gh-ost
	if err := utils.GhostControl(socketPath, command); err != nil {
		global.Logger.Error("Failed to control gh-ost",
			zap.String("order_id", req.OrderID),
			zap.String("socket_path", socketPath),
			zap.String("command", command),
			zap.Error(err),
		)
		api.HandleError(c, http.StatusInternalServerError, err, nil)
		return
	}

	// 推送消息到 WebSocket
	message := fmt.Sprintf("gh-ost 控制命令已发送：%s", command)
	if req.Action == "chunk-size" {
		message = fmt.Sprintf("gh-ost 速度已调节：chunk-size=%d", *req.Value)
	}
	if err := utils.PublishMessageToChannel(req.OrderID, message, "ghost"); err != nil {
		global.Logger.Warn("Failed to publish ghost control message",
			zap.String("order_id", req.OrderID),
			zap.Error(err),
		)
	}

	api.HandleSuccess(c, gin.H{
		"message": message,
	})
}
