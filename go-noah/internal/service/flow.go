package service

import (
	"context"
	"fmt"
	"go-noah/api"
	"go-noah/internal/model"
	"go-noah/internal/repository"
	"go-noah/pkg/global"
	"strings"
	"time"

	"github.com/duke-git/lancet/v2/convertor"
)

// FlowServiceApp 全局 Service 实例
var FlowServiceApp = new(FlowService)

type FlowService struct{}

func (s *FlowService) getFlowRepo() *repository.FlowRepository {
	return repository.NewFlowRepository(repository.NewRepository(global.Logger, global.DB, global.Enforcer))
}

// ============= 流程定义 =============

// GetFlowDefinitionList 获取流程定义列表
func (s *FlowService) GetFlowDefinitionList(ctx context.Context) (*api.FlowDefinitionListData, error) {
	repo := s.getFlowRepo()
	list, err := repo.GetFlowDefinitionList(ctx)
	if err != nil {
		return nil, err
	}

	var items []api.FlowDefinitionItem
	for _, f := range list {
		items = append(items, api.FlowDefinitionItem{
			ID:          f.ID,
			Code:        f.Code,
			Name:        f.Name,
			Type:        f.Type,
			Description: f.Description,
			Version:     f.Version,
			Status:      f.Status,
		})
	}

	return &api.FlowDefinitionListData{
		List: items,
	}, nil
}

// GetFlowDefinition 获取流程定义详情（包含节点）
func (s *FlowService) GetFlowDefinition(ctx context.Context, id uint) (*api.FlowDefinitionDetail, error) {
	repo := s.getFlowRepo()
	flow, err := repo.GetFlowDefinition(ctx, id)
	if err != nil {
		return nil, err
	}

	nodes, err := repo.GetFlowNodes(ctx, id)
	if err != nil {
		return nil, err
	}

	var nodeItems []api.FlowNodeItem
	for _, n := range nodes {
		nodeItems = append(nodeItems, api.FlowNodeItem{
			ID:            n.ID,
			NodeCode:      n.NodeCode,
			NodeName:      n.NodeName,
			NodeType:      n.NodeType,
			Sort:          n.Sort,
			ApproverType:  n.ApproverType,
			ApproverIDs:   n.ApproverIDs,
			MultiMode:     n.MultiMode,
			RejectAction:  n.RejectAction,
			TimeoutHours:  n.TimeoutHours,
			TimeoutAction: n.TimeoutAction,
			NextNodeCode:  n.NextNodeCode,
		})
	}

	return &api.FlowDefinitionDetail{
		ID:          flow.ID,
		Code:        flow.Code,
		Name:        flow.Name,
		Type:        flow.Type,
		Description: flow.Description,
		Version:     flow.Version,
		Status:      flow.Status,
		Nodes:       nodeItems,
	}, nil
}

// CreateFlowDefinition 创建流程定义
func (s *FlowService) CreateFlowDefinition(ctx context.Context, req *api.CreateFlowDefinitionRequest) error {
	repo := s.getFlowRepo()
	// 检查编码是否重复
	existing, _ := repo.GetFlowDefinitionByCode(ctx, req.Code)
	if existing != nil {
		return fmt.Errorf("流程编码 %s 已存在", req.Code)
	}

	flow := &model.FlowDefinition{
		Code:        req.Code,
		Name:        req.Name,
		Type:        req.Type,
		Description: req.Description,
		Version:     1,
		Status:      req.Status,
	}

	return repo.CreateFlowDefinition(ctx, flow)
}

// UpdateFlowDefinition 更新流程定义
func (s *FlowService) UpdateFlowDefinition(ctx context.Context, req *api.UpdateFlowDefinitionRequest) error {
	repo := s.getFlowRepo()
	flow, err := repo.GetFlowDefinition(ctx, req.ID)
	if err != nil {
		return fmt.Errorf("流程定义不存在")
	}

	flow.Name = req.Name
	flow.Description = req.Description
	flow.Status = req.Status

	return repo.UpdateFlowDefinition(ctx, flow)
}

// DeleteFlowDefinition 删除流程定义
func (s *FlowService) DeleteFlowDefinition(ctx context.Context, id uint) error {
	repo := s.getFlowRepo()
	return repo.DeleteFlowDefinition(ctx, id)
}

// SaveFlowNodes 保存流程节点（批量）
func (s *FlowService) SaveFlowNodes(ctx context.Context, req *api.SaveFlowNodesRequest) error {
	repo := s.getFlowRepo()
	// 先删除旧节点
	if err := repo.DeleteFlowNodesByFlowDefID(ctx, req.FlowDefID); err != nil {
		return err
	}

	// 创建新节点
	var nodes []model.FlowNode
	for _, n := range req.Nodes {
		nodes = append(nodes, model.FlowNode{
			FlowDefID:     req.FlowDefID,
			NodeCode:      n.NodeCode,
			NodeName:      n.NodeName,
			NodeType:      n.NodeType,
			Sort:          n.Sort,
			ApproverType:  n.ApproverType,
			ApproverIDs:   n.ApproverIDs,
			MultiMode:     n.MultiMode,
			RejectAction:  n.RejectAction,
			TimeoutHours:  n.TimeoutHours,
			TimeoutAction: n.TimeoutAction,
			NextNodeCode:  n.NextNodeCode,
		})
	}

	if len(nodes) > 0 {
		return repo.BatchCreateFlowNodes(ctx, nodes)
	}
	return nil
}

// ============= 流程实例 =============

// StartFlow 发起流程
func (s *FlowService) StartFlow(ctx context.Context, req *api.StartFlowRequest) (*api.StartFlowResponse, error) {
	repo := s.getFlowRepo()
	// 获取流程定义
	flowDef, err := repo.GetFlowDefinitionByType(ctx, req.BusinessType)
	if err != nil {
		return nil, fmt.Errorf("未找到业务类型 %s 对应的流程定义", req.BusinessType)
	}

	// 获取开始节点
	nodes, err := repo.GetFlowNodes(ctx, flowDef.ID)
	if err != nil || len(nodes) == 0 {
		return nil, fmt.Errorf("流程节点配置错误")
	}

	var startNode *model.FlowNode
	for i := range nodes {
		if nodes[i].NodeType == model.NodeTypeStart {
			startNode = &nodes[i]
			break
		}
	}
	if startNode == nil {
		return nil, fmt.Errorf("未找到开始节点")
	}

	// 创建流程实例
	now := time.Now()
	instance := &model.FlowInstance{
		FlowDefID:       flowDef.ID,
		FlowCode:        flowDef.Code,
		BusinessType:    req.BusinessType,
		BusinessID:      req.BusinessID,
		Title:           req.Title,
		InitiatorID:     req.InitiatorID,
		Initiator:       req.Initiator,
		Status:          model.FlowStatusRunning,
		CurrentNodeCode: startNode.NextNodeCode,
		StartTime:       now,
	}

	if err := repo.CreateFlowInstance(ctx, instance); err != nil {
		return nil, err
	}

	// 记录日志
	repo.CreateFlowLog(ctx, &model.FlowLog{
		FlowInstID: instance.ID,
		NodeCode:   "start",
		NodeName:   "开始",
		OperatorID: req.InitiatorID,
		Operator:   req.Initiator,
		Action:     "发起流程",
	})

	// 创建第一个审批任务
	nextNode, err := repo.GetFlowNodeByCode(ctx, flowDef.ID, startNode.NextNodeCode)
	if err != nil {
		return nil, err
	}

	if err := s.createApprovalTasks(ctx, instance, nextNode); err != nil {
		return nil, err
	}

	return &api.StartFlowResponse{
		FlowInstanceID: instance.ID,
	}, nil
}

// createApprovalTasks 创建审批任务
func (s *FlowService) createApprovalTasks(ctx context.Context, instance *model.FlowInstance, node *model.FlowNode) error {
	repo := s.getFlowRepo()
	// 根据审批人类型确定审批人
	var assignees []struct {
		ID       uint
		Username string
	}

	switch node.ApproverType {
	case model.ApproverTypeRole:
		// 获取角色下的所有用户
		roles := strings.Split(node.ApproverIDs, ",")
		for _, role := range roles {
			users, _ := global.Enforcer.GetUsersForRole(role)
			for _, uid := range users {
				uidInt, _ := convertor.ToInt(uid)
				assignees = append(assignees, struct {
					ID       uint
					Username string
				}{ID: uint(uidInt), Username: uid})
			}
		}
	case model.ApproverTypeUser:
		// 指定用户
		userIDs := strings.Split(node.ApproverIDs, ",")
		for _, uid := range userIDs {
			uidInt, _ := convertor.ToInt(uid)
			assignees = append(assignees, struct {
				ID       uint
				Username string
			}{ID: uint(uidInt), Username: uid})
		}
	case model.ApproverTypeAuto:
		// 自动通过，直接推进到下一节点
		return s.moveToNextNode(ctx, instance, node)
	}

	// 创建任务
	for _, assignee := range assignees {
		task := &model.FlowTask{
			FlowInstID: instance.ID,
			FlowNodeID: node.ID,
			NodeCode:   node.NodeCode,
			NodeName:   node.NodeName,
			AssigneeID: assignee.ID,
			Assignee:   assignee.Username,
			Status:     model.TaskStatusPending,
		}

		if node.TimeoutHours > 0 {
			dueTime := time.Now().Add(time.Duration(node.TimeoutHours) * time.Hour)
			task.DueTime = &dueTime
		}

		if err := repo.CreateFlowTask(ctx, task); err != nil {
			return err
		}
	}

	return nil
}

// ApproveTask 审批通过
func (s *FlowService) ApproveTask(ctx context.Context, req *api.ApproveTaskRequest) error {
	repo := s.getFlowRepo()
	task, err := repo.GetFlowTask(ctx, req.TaskID)
	if err != nil {
		return fmt.Errorf("任务不存在")
	}

	if task.Status != model.TaskStatusPending {
		return fmt.Errorf("任务已处理")
	}

	// 更新任务状态
	now := time.Now()
	task.Status = model.TaskStatusApproved
	task.Action = "approve"
	task.Comment = req.Comment
	task.ActionTime = &now

	if err := repo.UpdateFlowTask(ctx, task); err != nil {
		return err
	}

	// 记录日志
	repo.CreateFlowLog(ctx, &model.FlowLog{
		FlowInstID: task.FlowInstID,
		FlowNodeID: task.FlowNodeID,
		NodeCode:   task.NodeCode,
		NodeName:   task.NodeName,
		OperatorID: req.OperatorID,
		Operator:   req.Operator,
		Action:     "审批通过",
		Comment:    req.Comment,
	})

	// 获取流程实例
	instance, err := repo.GetFlowInstance(ctx, task.FlowInstID)
	if err != nil {
		return err
	}

	// 获取当前节点
	node, err := repo.GetFlowNode(ctx, task.FlowNodeID)
	if err != nil {
		return err
	}

	// 判断是否所有任务都已完成（会签模式）
	if node.MultiMode == model.MultiModeAll {
		pendingTasks, _ := repo.GetPendingTasksByInstance(ctx, instance.ID)
		if len(pendingTasks) > 0 {
			return nil // 还有待处理任务
		}
	}

	// 推进到下一节点
	return s.moveToNextNode(ctx, instance, node)
}

// RejectTask 审批驳回
func (s *FlowService) RejectTask(ctx context.Context, req *api.RejectTaskRequest) error {
	repo := s.getFlowRepo()
	task, err := repo.GetFlowTask(ctx, req.TaskID)
	if err != nil {
		return fmt.Errorf("任务不存在")
	}

	if task.Status != model.TaskStatusPending {
		return fmt.Errorf("任务已处理")
	}

	// 更新任务状态
	now := time.Now()
	task.Status = model.TaskStatusRejected
	task.Action = "reject"
	task.Comment = req.Comment
	task.ActionTime = &now

	if err := repo.UpdateFlowTask(ctx, task); err != nil {
		return err
	}

	// 记录日志
	repo.CreateFlowLog(ctx, &model.FlowLog{
		FlowInstID: task.FlowInstID,
		FlowNodeID: task.FlowNodeID,
		NodeCode:   task.NodeCode,
		NodeName:   task.NodeName,
		OperatorID: req.OperatorID,
		Operator:   req.Operator,
		Action:     "审批驳回",
		Comment:    req.Comment,
	})

	// 更新流程实例状态
	return repo.UpdateFlowInstanceStatus(ctx, task.FlowInstID, model.FlowStatusRejected)
}

// moveToNextNode 推进到下一节点
func (s *FlowService) moveToNextNode(ctx context.Context, instance *model.FlowInstance, currentNode *model.FlowNode) error {
	repo := s.getFlowRepo()
	if currentNode.NextNodeCode == "" || currentNode.NextNodeCode == "end" {
		// 流程结束
		return repo.UpdateFlowInstanceStatus(ctx, instance.ID, model.FlowStatusApproved)
	}

	// 获取下一节点
	flowDef, _ := repo.GetFlowDefinition(ctx, instance.FlowDefID)
	nextNode, err := repo.GetFlowNodeByCode(ctx, flowDef.ID, currentNode.NextNodeCode)
	if err != nil {
		return err
	}

	if nextNode.NodeType == model.NodeTypeEnd {
		// 流程结束
		return repo.UpdateFlowInstanceStatus(ctx, instance.ID, model.FlowStatusApproved)
	}

	// 更新当前节点
	instance.CurrentNodeCode = nextNode.NodeCode
	instance.CurrentNodeID = nextNode.ID
	if err := repo.UpdateFlowInstance(ctx, instance); err != nil {
		return err
	}

	// 创建新的审批任务
	return s.createApprovalTasks(ctx, instance, nextNode)
}

// GetMyPendingTasks 获取我的待办任务
func (s *FlowService) GetMyPendingTasks(ctx context.Context, userID uint, page, pageSize int) (*api.FlowTaskListData, error) {
	repo := s.getFlowRepo()
	list, total, err := repo.GetMyPendingTasks(ctx, userID, page, pageSize)
	if err != nil {
		return nil, err
	}

	var items []api.FlowTaskItem
	for _, t := range list {
		items = append(items, api.FlowTaskItem{
			ID:         t.ID,
			FlowInstID: t.FlowInstID,
			NodeCode:   t.NodeCode,
			NodeName:   t.NodeName,
			Status:     t.Status,
			CreatedAt:  t.CreatedAt,
		})
	}

	return &api.FlowTaskListData{
		List:  items,
		Total: total,
	}, nil
}

// GetFlowInstanceDetail 获取流程实例详情
func (s *FlowService) GetFlowInstanceDetail(ctx context.Context, id uint) (*api.FlowInstanceDetail, error) {
	repo := s.getFlowRepo()
	instance, err := repo.GetFlowInstance(ctx, id)
	if err != nil {
		return nil, err
	}

	// 获取任务列表
	tasks, _ := repo.GetFlowTasksByInstance(ctx, id)
	var taskItems []api.FlowTaskItem
	for _, t := range tasks {
		taskItems = append(taskItems, api.FlowTaskItem{
			ID:         t.ID,
			FlowInstID: t.FlowInstID,
			NodeCode:   t.NodeCode,
			NodeName:   t.NodeName,
			Assignee:   t.Assignee,
			Status:     t.Status,
			Action:     t.Action,
			Comment:    t.Comment,
			ActionTime: t.ActionTime,
			CreatedAt:  t.CreatedAt,
		})
	}

	// 获取日志列表
	logs, _ := repo.GetFlowLogs(ctx, id)
	var logItems []api.FlowLogItem
	for _, l := range logs {
		logItems = append(logItems, api.FlowLogItem{
			ID:        l.ID,
			NodeCode:  l.NodeCode,
			NodeName:  l.NodeName,
			Operator:  l.Operator,
			Action:    l.Action,
			Comment:   l.Comment,
			CreatedAt: l.CreatedAt,
		})
	}

	return &api.FlowInstanceDetail{
		ID:              instance.ID,
		FlowCode:        instance.FlowCode,
		BusinessType:    instance.BusinessType,
		BusinessID:      instance.BusinessID,
		Title:           instance.Title,
		Initiator:       instance.Initiator,
		Status:          instance.Status,
		CurrentNodeCode: instance.CurrentNodeCode,
		StartTime:       instance.StartTime,
		EndTime:         instance.EndTime,
		Tasks:           taskItems,
		Logs:            logItems,
	}, nil
}

