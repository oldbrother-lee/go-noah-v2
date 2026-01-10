import { request, requestRaw } from '../request';

/** Orders API - 新路由: /api/v1/insight/... */

/**
 * Get environments for orders
 */
export function fetchOrdersEnvironments(params?: Record<string, any>) {
  return request<Api.Orders.Environment[]>({
    url: '/api/v1/insight/environments',
    method: 'get',
    params
  });
}

/**
 * Get instances (dbconfigs) for specified environment
 */
export function fetchOrdersInstances(params?: Record<string, any>) {
  return request<Api.Orders.Instance[]>({
    url: '/api/v1/insight/dbconfigs',
    method: 'get',
    params
  });
}

/**
 * Get schemas for specified instance
 */
export function fetchOrdersSchemas(params?: Record<string, any>) {
  return request<Api.Orders.Schema[]>({
    url: `/api/v1/insight/dbconfigs/${params?.instance_id}/schemas`,
    method: 'get'
  });
}

/**
 * Get users for review/audit/cc
 */
export function fetchOrdersUsers(params?: Record<string, any>) {
  return request<Api.Orders.User[]>({
    url: '/api/v1/admin/users',
    method: 'get',
    params: {
      page: 1,
      pageSize: 1000, // 获取足够多的用户供选择
      ...params
    }
  });
}

/**
 * Syntax check for SQL (SQL审核)
 */
export function fetchSyntaxCheck(data: Api.Orders.SyntaxCheckRequest) {
  return request<Api.Orders.SyntaxCheckResult>({
    url: '/api/v1/insight/inspect/sql',
    method: 'post',
    data,
    timeout: 10 * 60 * 1000
  });
}

/**
 * Create order
 */
export function fetchCreateOrder(data: Api.Orders.CreateOrderRequest) {
  return request<Api.Orders.Order>({
    url: '/api/v1/insight/orders',
    method: 'post',
    data
  });
}

/**
 * Get orders list
 */
export function fetchOrdersList(params?: Record<string, any>) {
  return request<Api.Orders.OrdersList>({
    url: '/api/v1/insight/orders',
    method: 'get',
    params
  });
}

/**
 * Get my orders list
 */
export function fetchMyOrdersList(params?: Record<string, any>) {
  return request<Api.Orders.OrdersList>({
    url: '/api/v1/insight/orders/my',
    method: 'get',
    params
  });
}

/**
 * Get order detail
 */
export function fetchOrderDetail(id: string) {
  return request<Api.Orders.OrderDetail>({
    url: `/api/v1/insight/orders/${id}`,
    method: 'get'
  });
}

/**
 * Get operation logs
 */
export function fetchOpLogs(params?: Record<string, any>) {
  return request<Api.Orders.OpLog[]>({
    url: `/api/v1/insight/orders/${params?.order_id}/logs`,
    method: 'get'
  });
}

/**
 * Update order progress (approve/reject/close)
 */
export function fetchUpdateOrderProgress(data: { order_id: string; progress: string; remark?: string }) {
  return request({
    url: '/api/v1/insight/orders/progress',
    method: 'put',
    data
  });
}

/**
 * Approve order
 */
export function fetchApproveOrder(data: Api.Orders.ApproveOrderRequest) {
  return fetchUpdateOrderProgress({
    order_id: data.order_id,
    progress: '已批准',
    remark: data.remark
  });
}

/**
 * Feedback order (reject)
 */
export function fetchFeedbackOrder(data: Api.Orders.FeedbackOrderRequest) {
  return fetchUpdateOrderProgress({
    order_id: data.order_id,
    progress: '已驳回',
    remark: data.remark
  });
}

/**
 * Review order
 */
export function fetchReviewOrder(data: Api.Orders.ReviewOrderRequest) {
  return fetchUpdateOrderProgress({
    order_id: data.order_id,
    progress: '已复核',
    remark: data.remark
  });
}

/**
 * Close order
 */
export function fetchCloseOrder(data: Api.Orders.CloseOrderRequest) {
  return fetchUpdateOrderProgress({
    order_id: data.order_id,
    progress: '已关闭',
    remark: data.remark
  });
}

/**
 * Update order schedule time
 */
export function fetchUpdateOrderSchedule(data: { order_id: string; schedule_time: string }) {
  return request({
    url: '/api/v1/insight/orders/progress',
    method: 'put',
    data
  });
}

/**
 * Hook order (创建关联工单)
 */
export function fetchHookOrder(data: Api.Orders.HookOrderRequest) {
  return request({
    url: '/api/v1/insight/orders',
    method: 'post',
    data
  });
}

/**
 * Generate tasks (由后端在创建工单时自动生成)
 */
export function fetchGenerateTasks(data: Api.Orders.GenerateTasksRequest) {
  return request<Api.Orders.Task[]>({
    url: '/api/v1/insight/orders',
    method: 'post',
    data
  });
}

/**
 * Get tasks for order
 */
export function fetchTasks(params: { order_id: string }) {
  return request<Api.Orders.Task[]>({
    url: `/api/v1/insight/orders/${params.order_id}/tasks`,
    method: 'get'
  });
}

/**
 * Preview tasks
 */
export function fetchPreviewTasks(params?: Record<string, any>) {
  return request<Api.Orders.TaskPreview>({
    url: `/api/v1/insight/orders/${params?.order_id}/tasks`,
    method: 'get'
  });
}

/**
 * Update task progress
 */
export function fetchUpdateTaskProgress(data: { task_id: string; progress: string }) {
  return request({
    url: '/api/v1/insight/orders/tasks/progress',
    method: 'put',
    data
  });
}

/**
 * Execute single task
 */
export function fetchExecuteSingleTask(data: Api.Orders.ExecuteTaskRequest) {
  return request<Api.Orders.TaskResult>({
    url: '/api/v1/insight/orders/tasks/execute',
    method: 'post',
    data
  });
}

/**
 * Execute all tasks
 */
export function fetchExecuteAllTasks(data: Api.Orders.ExecuteAllTasksRequest) {
  return requestRaw<any>({
    url: '/api/v1/insight/orders/tasks/execute',
    method: 'post',
    data,
    timeout: 24 * 60 * 60 * 1000
  });
}

/**
 * Download export file
 */
export function fetchDownloadExportFile(taskId: string | number) {
  return request<Blob>({
    url: `/api/v1/insight/orders/download/${taskId}`,
    method: 'get',
    responseType: 'blob'
  });
}

/**
 * Control gh-ost execution (throttle/unthrottle/panic/chunk-size)
 */
export function fetchControlGhost(data: {
  order_id: string;
  action: 'throttle' | 'unthrottle' | 'panic' | 'chunk-size';
  value?: number;
}) {
  return request<{ message: string }>({
    url: '/api/v1/insight/orders/ghost/control',
    method: 'post',
    data
  });
}
