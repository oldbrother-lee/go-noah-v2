package service

import (
	"context"
	"go-noah/internal/inspect/parser"
	"go-noah/internal/model/insight"
	"go-noah/internal/repository"
	insightRepo "go-noah/internal/repository/insight"
	"go-noah/pkg/global"
)

// InsightServiceApp 全局 Service 实例
var InsightServiceApp = new(InsightService)

// InsightService goInsight 功能的业务逻辑层
type InsightService struct{}

func (s *InsightService) getRepo() *insightRepo.InsightRepository {
	return insightRepo.NewInsightRepository(
		repository.NewRepository(global.Logger, global.DB, global.Enforcer),
		global.Logger,
		global.Enforcer,
	)
}

// ============ 环境管理 ============

func (s *InsightService) GetEnvironments(ctx context.Context) ([]insight.DBEnvironment, error) {
	return s.getRepo().GetEnvironments(ctx)
}

func (s *InsightService) CreateEnvironment(ctx context.Context, env *insight.DBEnvironment) error {
	return s.getRepo().CreateEnvironment(ctx, env)
}

func (s *InsightService) UpdateEnvironment(ctx context.Context, id uint, name string) error {
	env := &insight.DBEnvironment{}
	env.ID = id
	env.Name = name
	return s.getRepo().UpdateEnvironment(ctx, env)
}

func (s *InsightService) DeleteEnvironment(ctx context.Context, id uint) error {
	return s.getRepo().DeleteEnvironment(ctx, id)
}

// ============ 数据库配置管理 ============

func (s *InsightService) GetDBConfigs(ctx context.Context, useType insight.UseType, environment int) ([]insight.DBConfig, error) {
	return s.getRepo().GetDBConfigs(ctx, useType, environment)
}

func (s *InsightService) GetDBConfigByInstanceID(ctx context.Context, instanceID string) (*insight.DBConfig, error) {
	return s.getRepo().GetDBConfigByInstanceID(ctx, instanceID)
}

func (s *InsightService) CreateDBConfig(ctx context.Context, config *insight.DBConfig) error {
	// TODO: 密码加密存储
	return s.getRepo().CreateDBConfig(ctx, config)
}

func (s *InsightService) UpdateDBConfig(ctx context.Context, config *insight.DBConfig) error {
	// TODO: 密码加密存储
	return s.getRepo().UpdateDBConfig(ctx, config)
}

func (s *InsightService) DeleteDBConfig(ctx context.Context, id uint) error {
	return s.getRepo().DeleteDBConfig(ctx, id)
}

// ============ Schema 管理 ============

func (s *InsightService) GetSchemasByInstanceID(ctx context.Context, instanceID string) ([]insight.DBSchema, error) {
	return s.getRepo().GetSchemasByInstanceID(ctx, instanceID)
}

// ============ 组织管理 ============

func (s *InsightService) GetOrganizations(ctx context.Context) ([]insight.Organization, error) {
	return s.getRepo().GetOrganizations(ctx)
}

func (s *InsightService) CreateOrganization(ctx context.Context, org *insight.Organization) error {
	return s.getRepo().CreateOrganization(ctx, org)
}

func (s *InsightService) UpdateOrganization(ctx context.Context, org *insight.Organization) error {
	return s.getRepo().UpdateOrganization(ctx, org)
}

func (s *InsightService) DeleteOrganization(ctx context.Context, id uint64) error {
	return s.getRepo().DeleteOrganization(ctx, id)
}

func (s *InsightService) GetOrganizationByID(ctx context.Context, id uint64) (*insight.Organization, error) {
	return s.getRepo().GetOrganizationByID(ctx, id)
}

func (s *InsightService) GetOrganizationUsers(ctx context.Context, orgKey string) ([]insight.OrganizationUser, error) {
	return s.getRepo().GetOrganizationUsers(ctx, orgKey)
}

func (s *InsightService) BindOrganizationUser(ctx context.Context, ou *insight.OrganizationUser) error {
	return s.getRepo().BindOrganizationUser(ctx, ou)
}

func (s *InsightService) UnbindOrganizationUser(ctx context.Context, uid uint64) error {
	return s.getRepo().UnbindOrganizationUser(ctx, uid)
}

// ============ DAS 权限管理 ============

// GetUserAuthorizedSchemas 获取用户授权的所有 schemas
func (s *InsightService) GetUserAuthorizedSchemas(ctx context.Context, username string) ([]insight.UserAuthorizedSchema, error) {
	return s.getRepo().GetUserAuthorizedSchemas(ctx, username)
}

func (s *InsightService) GetUserSchemaPermissions(ctx context.Context, username string) ([]insight.DASUserSchemaPermission, error) {
	return s.getRepo().GetUserSchemaPermissions(ctx, username)
}

func (s *InsightService) CreateSchemaPermission(ctx context.Context, perm *insight.DASUserSchemaPermission) error {
	return s.getRepo().CreateSchemaPermission(ctx, perm)
}

func (s *InsightService) DeleteSchemaPermission(ctx context.Context, id uint) error {
	return s.getRepo().DeleteSchemaPermission(ctx, id)
}

func (s *InsightService) GetUserTablePermissions(ctx context.Context, username string) ([]insight.DASUserTablePermission, error) {
	return s.getRepo().GetUserTablePermissions(ctx, username)
}

// ============ 权限模板管理 ============

func (s *InsightService) GetPermissionTemplates(ctx context.Context) ([]insight.DASPermissionTemplate, error) {
	return s.getRepo().GetPermissionTemplates(ctx)
}

func (s *InsightService) GetPermissionTemplate(ctx context.Context, id uint) (*insight.DASPermissionTemplate, error) {
	return s.getRepo().GetPermissionTemplate(ctx, id)
}

func (s *InsightService) CreatePermissionTemplate(ctx context.Context, template *insight.DASPermissionTemplate) error {
	return s.getRepo().CreatePermissionTemplate(ctx, template)
}

func (s *InsightService) UpdatePermissionTemplate(ctx context.Context, template *insight.DASPermissionTemplate) error {
	return s.getRepo().UpdatePermissionTemplate(ctx, template)
}

func (s *InsightService) DeletePermissionTemplate(ctx context.Context, id uint) error {
	return s.getRepo().DeletePermissionTemplate(ctx, id)
}

// ============ 角色权限管理 ============

func (s *InsightService) GetRolePermissions(ctx context.Context, role string) ([]insight.DASRolePermission, error) {
	return s.getRepo().GetRolePermissions(ctx, role)
}

func (s *InsightService) CreateRolePermission(ctx context.Context, perm *insight.DASRolePermission) error {
	return s.getRepo().CreateRolePermission(ctx, perm)
}

func (s *InsightService) DeleteRolePermission(ctx context.Context, id uint) error {
	return s.getRepo().DeleteRolePermission(ctx, id)
}

func (s *InsightService) BatchCreateRolePermissions(ctx context.Context, perms []insight.DASRolePermission) error {
	return s.getRepo().BatchCreateRolePermissions(ctx, perms)
}

// ============ 权限查询 ============

func (s *InsightService) GetUserEffectivePermissions(ctx context.Context, username string) ([]insight.PermissionObject, error) {
	return s.getRepo().GetUserEffectivePermissions(ctx, username)
}

func (s *InsightService) ExpandRolePermissions(ctx context.Context, role string) ([]insight.PermissionObject, error) {
	return s.getRepo().ExpandRolePermissions(ctx, role)
}

func (s *InsightService) CreateTablePermission(ctx context.Context, perm *insight.DASUserTablePermission) error {
	return s.getRepo().CreateTablePermission(ctx, perm)
}

func (s *InsightService) DeleteTablePermission(ctx context.Context, id uint) error {
	return s.getRepo().DeleteTablePermission(ctx, id)
}

// ============ DAS 执行记录 ============

func (s *InsightService) CreateDASRecord(ctx context.Context, record *insight.DASRecord) error {
	return s.getRepo().CreateDASRecord(ctx, record)
}

func (s *InsightService) GetDASRecords(ctx context.Context, username string, page, pageSize int) ([]insight.DASRecord, int64, error) {
	return s.getRepo().GetDASRecords(ctx, username, page, pageSize)
}

// ============ DAS 收藏夹 ============

func (s *InsightService) GetFavorites(ctx context.Context, username string) ([]insight.DASFavorite, error) {
	return s.getRepo().GetFavorites(ctx, username)
}

func (s *InsightService) CreateFavorite(ctx context.Context, fav *insight.DASFavorite) error {
	return s.getRepo().CreateFavorite(ctx, fav)
}

func (s *InsightService) UpdateFavorite(ctx context.Context, fav *insight.DASFavorite) error {
	return s.getRepo().UpdateFavorite(ctx, fav)
}

func (s *InsightService) DeleteFavorite(ctx context.Context, id uint, username string) error {
	return s.getRepo().DeleteFavorite(ctx, id, username)
}

// ============ 工单管理 ============

func (s *InsightService) GetOrders(ctx context.Context, params *insightRepo.OrderQueryParams) ([]insightRepo.OrderWithInstance, int64, error) {
	return s.getRepo().GetOrders(ctx, params)
}

func (s *InsightService) GetOrderByID(ctx context.Context, orderID string) (*insightRepo.OrderWithInstance, error) {
	return s.getRepo().GetOrderByID(ctx, orderID)
}

func (s *InsightService) CreateOrder(ctx context.Context, order *insight.OrderRecord) error {
	return s.getRepo().CreateOrder(ctx, order)
}

func (s *InsightService) UpdateOrder(ctx context.Context, order *insight.OrderRecord) error {
	return s.getRepo().UpdateOrder(ctx, order)
}

func (s *InsightService) UpdateOrderProgress(ctx context.Context, orderID string, progress insight.Progress) error {
	// 先更新进度
	if err := s.getRepo().UpdateOrderProgress(ctx, orderID, progress); err != nil {
		return err
	}

	// 如果进度变为"已批准"，自动生成任务
	if progress == insight.ProgressApproved {
		// 检查任务是否已存在
		tasks, err := s.getRepo().GetOrderTasks(ctx, orderID)
		if err == nil && len(tasks) == 0 {
			// 任务不存在，自动生成
			_ = s.GenerateOrderTasks(ctx, orderID)
		}
	}

	return nil
}

// GenerateOrderTasks 为工单生成任务
func (s *InsightService) GenerateOrderTasks(ctx context.Context, orderID string) error {
	// 获取工单信息
	order, err := s.getRepo().GetOrderByID(ctx, orderID)
	if err != nil {
		return err
	}

	// 检查任务是否已存在
	existingTasks, err := s.getRepo().GetOrderTasks(ctx, orderID)
	if err == nil && len(existingTasks) > 0 {
		// 任务已存在，跳过
		return nil
	}

	// 拆分 SQL
	sqls, err := s.splitSQLText(order.Content)
	if err != nil {
		return err
	}

	// 创建任务
	var tasks []insight.OrderTask
	for _, sql := range sqls {
		task := insight.OrderTask{
			OrderID:  order.OrderID,
			DBType:   order.DBType,
			SQLType:  order.SQLType,
			SQL:      sql,
			Progress: insight.TaskProgressPending,
		}
		tasks = append(tasks, task)
	}

	// 批量创建任务
	if len(tasks) > 0 {
		return s.getRepo().CreateOrderTasks(ctx, tasks)
	}

	return nil
}

// splitSQLText 拆分SQL文本（内部辅助方法）
func (s *InsightService) splitSQLText(sqltext string) ([]string, error) {
	// 使用 inspect parser 拆分 SQL
	audit, warns, err := parser.ParseSQL(sqltext)
	if err != nil {
		return nil, err
	}
	if len(warns) > 0 {
		// 有警告但不影响拆分
	}

	var sqls []string
	for _, stmt := range audit.TiStmt {
		sqls = append(sqls, stmt.Text())
	}
	return sqls, nil
}

// ============ 工单任务管理 ============

func (s *InsightService) GetOrderTasks(ctx context.Context, orderID string) ([]insight.OrderTask, error) {
	return s.getRepo().GetOrderTasks(ctx, orderID)
}

func (s *InsightService) GetTaskByID(ctx context.Context, taskID string) (*insight.OrderTask, error) {
	return s.getRepo().GetTaskByID(ctx, taskID)
}

func (s *InsightService) CreateOrderTasks(ctx context.Context, tasks []insight.OrderTask) error {
	return s.getRepo().CreateOrderTasks(ctx, tasks)
}

func (s *InsightService) UpdateTaskProgress(ctx context.Context, taskID string, progress insight.TaskProgress, result []byte) error {
	return s.getRepo().UpdateTaskProgress(ctx, taskID, progress, result)
}

// CheckTasksProgressIsDoing 检查工单是否有任务正在执行中
func (s *InsightService) CheckTasksProgressIsDoing(ctx context.Context, orderID string) (bool, error) {
	return s.getRepo().CheckTasksProgressIsDoing(ctx, orderID)
}

// CheckTasksProgressIsPause 检查工单是否有已暂停的任务
func (s *InsightService) CheckTasksProgressIsPause(ctx context.Context, orderID string) (bool, error) {
	return s.getRepo().CheckTasksProgressIsPause(ctx, orderID)
}

// UpdateOrderExecuteResult 更新工单执行结果
func (s *InsightService) UpdateOrderExecuteResult(ctx context.Context, orderID string, result string) error {
	return s.getRepo().UpdateOrderExecuteResult(ctx, orderID, result)
}

// CheckAllTasksCompleted 检查所有任务是否都已完成
func (s *InsightService) CheckAllTasksCompleted(ctx context.Context, orderID string) (bool, error) {
	return s.getRepo().CheckAllTasksCompleted(ctx, orderID)
}

// UpdateTaskAndOrderProgress 使用事务同时更新任务和工单状态
func (s *InsightService) UpdateTaskAndOrderProgress(ctx context.Context, taskID string, orderID string, taskProgress insight.TaskProgress, orderProgress insight.Progress) error {
	return s.getRepo().UpdateTaskAndOrderProgress(ctx, taskID, orderID, taskProgress, orderProgress)
}

// ============ 操作日志管理 ============

func (s *InsightService) CreateOpLog(ctx context.Context, log *insight.OrderOpLog) error {
	return s.getRepo().CreateOpLog(ctx, log)
}

func (s *InsightService) GetOpLogs(ctx context.Context, orderID string) ([]insight.OrderOpLog, error) {
	return s.getRepo().GetOpLogs(ctx, orderID)
}
