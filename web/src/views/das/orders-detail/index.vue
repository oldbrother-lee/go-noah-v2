<script setup lang="ts">
import { computed, h, onMounted, onUnmounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useDebounceFn, useThrottleFn } from '@vueuse/core';
import {
  NButton,
  NDataTable,
  NDatePicker,
  NCard,
  NDivider,
  NInputNumber,
  NProgress,
  NSpace,
  NStep,
  NSteps,
  NTag,
  NTooltip,
  useDialog,
  useMessage
} from 'naive-ui';
import type { TagProps } from 'naive-ui';
import { format } from 'sql-formatter';
import {
  fetchApproveOrder,
  fetchCloseOrder,
  fetchControlGhost,
  fetchExecuteAllTasks,
  fetchExecuteSingleTask,
  fetchFeedbackOrder,
  fetchGenerateTasks,
  fetchHookOrder,
  fetchOpLogs,
  fetchOrderDetail,
  fetchOrdersEnvironments,
  fetchOrdersInstances,
  fetchOrdersSchemas,
  fetchPreviewTasks,
  fetchReviewOrder,
  fetchSyntaxCheck,
  fetchTasks,
  fetchUpdateOrderSchedule
} from '@/service/api/orders';
import { useThemeStore } from '@/store/modules/theme';
import { useAuthStore } from '@/store/modules/auth';
import { useAppStore } from '@/store/modules/app';
import ReadonlySqlEditor from '@/components/custom/readonly-sql-editor.vue';
import LogViewer from '@/components/custom/log-viewer.vue';

const route = useRoute();
const router = useRouter();
const message = useMessage();
const dialog = useDialog();
const themeStore = useThemeStore();
const authStore = useAuthStore();
const appStore = useAppStore();

const showProgress = ref(false); // 默认收起工单进度

const theme = computed(() => (themeStore.darkMode ? 'dark' : 'light'));

// 新的详情数据结构
interface OrderDetailVO {
  id: number;
  created_at: string;
  updated_at: string;
  title: string;
  order_id: string;
  hook_order_id: string;
  remark: string;
  is_restrict_access: boolean;
  db_type: string;
  sql_type: string;
  applicant: string;
  organization: string;
  approver: { user: string; status: string }[];
  executor: string[];
  reviewer: { user: string; status: string }[];
  cc: string[];
  instance_id: string;
  schema: string;
  // 追加可选字段：与模板绑定保持一致
  is_backup?: boolean;
  scheduled?: boolean;
  progress: string;
  execute_result?: string;
  fix_version: string;
  content: string;
  export_file_format: string;
  environment: string;
  instance: string;
  schedule_time?: string;
}

// 工单详情数据
const orderDetail = ref<OrderDetailVO | null>(null);
const loading = ref(false);

// 其他状态
const opLogs = ref<any[]>([]);
const executeLoading = ref(false);
const refreshLoading = ref(false);
const actionVisible = ref(false);
const closeVisible = ref(false);
const confirmMsg = ref('');
const hookVisible = ref(false);
const resetToPending = ref(true);

// 任务列表数据
const tasksList = ref<any[]>([]);

// 当前激活的标签页
const activeTab = ref('sql-content');

// 语法检查相关状态
const checking = ref(false);
const syntaxRows = ref<any[]>([]);
const syntaxStatus = ref<number | null>(null);
const localSqlContent = ref('');

// 初始化 localSqlContent
watch(
  () => orderDetail.value,
  val => {
    if (val) {
      localSqlContent.value = val.content;
      syntaxStatus.value = null;
      syntaxRows.value = [];
    }
  }
);

const syntaxColumns = [
  {
    title: '检测结果',
    key: 'result',
    width: 100,
    fixed: 'left' as const,
    render: (row: any) =>
      h(
        NTag,
        { type: row?.level === 'INFO' && (!row?.summary || row.summary.length === 0) ? 'success' : 'error' },
        { default: () => (row?.level === 'INFO' && (!row?.summary || row.summary.length === 0) ? '通过' : '失败') }
      )
  },
  { title: '错误级别', key: 'level', width: 80 },
  { title: '影响行数', key: 'affected_rows', width: 90 },
  { title: '类型', key: 'type', width: 90 },
  { title: '指纹', key: 'finger_id', width: 120 },
  {
    title: '信息提示',
    key: 'summary',
    width: 300,
    ellipsis: { tooltip: { style: { maxWidth: '600px' } } as any },
    render: (row: any) => (row.summary && row.summary.length ? row.summary.join('；') : '—')
  },
  { title: 'SQL', key: 'query', width: 500, ellipsis: { tooltip: { style: { maxWidth: '800px' } } as any } }
];

const handleFormatSQL = () => {
  try {
    localSqlContent.value = format(localSqlContent.value || '', { language: 'mysql' });
    message.success('格式化完成');
  } catch (e) {
    message.error('格式化失败');
  }
};

const handleSyntaxCheck = async () => {
  if (!orderDetail.value) return;
  checking.value = true;
  syntaxStatus.value = null;
  syntaxRows.value = [];
  try {
    const data = {
      db_type: orderDetail.value.db_type,
      sql_type: orderDetail.value.sql_type,
      instance_id: orderDetail.value.instance_id,
      schema: orderDetail.value.schema,
      content: localSqlContent.value
    };
    const resp: any = await fetchSyntaxCheck(data as any);
    console.log('语法检查响应:', resp);

    // 检查是否有错误
    if (resp.error || resp.data === null || resp.data === undefined) {
      syntaxStatus.value = 1;
      const errorData = resp.error?.response?.data?.data ?? resp.response?.data?.data ?? [];
      syntaxRows.value = Array.isArray(errorData) ? errorData : [];
      return;
    }

    // 请求成功，data 格式：{status: 0/1, data: [...]}（与老服务一致）
    const resultData = resp.data?.data ?? [];
    syntaxRows.value = Array.isArray(resultData) ? resultData : [];
    
    // 检查 status 字段（与老服务一致）
    // status: 0表示语法检查通过，1表示语法检查不通过
    const status = resp.data?.status ?? 1; // 默认不通过
    syntaxStatus.value = status;
    
    if (status === 0) {
      message.success('语法检查通过');
    } else {
      message.warning('语法检查未通过，请修复问题后重新检查');
    }
  } catch (e: any) {
    console.error('语法检查失败:', e);
    message.error(e?.message || '检查失败');
    syntaxStatus.value = null;
  } finally {
    checking.value = false;
  }
};

// 处理标签页切换
const handleTabChange = (value: string) => {
  if (value === 'results') {
    getTasks();
  }
};

// 显示的SQL内容
const displaySqlContent = computed(() => {
  return localSqlContent.value || orderDetail.value?.content || '';
});

// 步骤条当前步骤
const currentStep = computed(() => {
  const status = orderDetail.value?.progress;
  switch (status) {
    case '待审核':
      return 2;
    case '已驳回':
      return 3;
    case '已批准':
      return 3;
    case '执行中':
      return 4;
    case '已完成':
      return 5;
    case '已失败':
      return 5;
    case '已复核':
      return 6;
    case '已关闭':
      return 6;
    default:
      return 1;
  }
});

const currentStepStatus = computed(() => {
  const p = orderDetail.value?.progress;
  if (!p) return 'process';
  if (p === '已驳回' || p === '已失败') return 'error';
  if (['已完成', '已复核'].includes(p)) return 'finish';
  return 'process';
});

// OSC Progress
const oscContent = ref('');
const websocket = ref<WebSocket | null>(null);
const reconnectAttempts = ref(0);
const maxReconnectAttempts = 10; // 最大重连次数
const reconnectDelay = 3000; // 重连延迟（毫秒）
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;

// 性能优化：日志内容限制
const MAX_LOG_LENGTH = 10 * 1024 * 1024; // 最大日志长度：10MB
const MAX_LOG_LINES = 50000; // 最大日志行数：50000行
const TRIM_LOG_LENGTH = 5 * 1024 * 1024; // 当超过最大长度时，保留前5MB

// 日志缓冲区（用于批量更新，减少DOM操作）
const logBuffer = ref<string[]>([]);
const isUpdatingLog = ref(false);

// gh-ost 进度信息
interface GhostProgress {
  percent?: number;
  current?: number;
  total?: number;
  eta?: string;
  operation?: string;
}

const ghostProgress = ref<GhostProgress | null>(null);
const ghostThrottled = ref(false); // 是否已暂停
const ghostControlLoading = ref(false); // 控制按钮加载状态
const ghostChunkSize = ref<number | null>(800); // 默认 chunk-size 值

// 判断是否是 gh-ost 执行的工单（DDL 类型且在执行中）
const isGhostOrder = computed(() => {
  if (!orderDetail.value || !isOrderExecuting.value) return false;
  // DDL 类型会使用 gh-ost 执行
  return orderDetail.value.sql_type === 'DDL';
});

// 显示用的 gh-ost 进度信息（如果没有收到进度信息，显示 0）
const displayGhostProgress = computed((): GhostProgress | null => {
  if (ghostProgress.value && ghostProgress.value.percent !== undefined) {
    return ghostProgress.value;
  }
  // 如果是 gh-ost 工单但没有进度信息，显示 0
  if (isGhostOrder.value) {
    return { percent: 0 };
  }
  return null;
});

// 检查工单是否还在执行中
const isOrderExecuting = computed(() => {
  const status = orderDetail.value?.progress;
  return status && ['执行中', '已批准'].includes(status);
});

const initWebSocket = (force = false) => {
  // 如果已经连接，不重复连接
  if (websocket.value && websocket.value.readyState === WebSocket.OPEN) {
    return;
  }

  // 如果强制连接，跳过状态检查
  if (!force) {
    // 如果工单不在执行中，且不是重连尝试，不建立连接
    if (!isOrderExecuting.value && reconnectAttempts.value === 0) {
      console.log('WebSocket 连接被跳过：工单不在执行中', { 
        orderId: route.params.id, 
        status: orderDetail.value?.progress,
        isOrderExecuting: isOrderExecuting.value 
      });
      return;
    }
    
    // 如果是重连尝试，但工单状态已改变，停止重连
    if (reconnectAttempts.value > 0 && !isOrderExecuting.value) {
      reconnectAttempts.value = 0;
      return;
    }
  }

  const orderId = route.params.id as string;
  if (!orderId) return;

  let wsUrl = '';
  const isDev = import.meta.env.DEV;
  const serviceBaseUrl = import.meta.env.VITE_SERVICE_BASE_URL;

  if (isDev && serviceBaseUrl) {
    // 开发环境：使用配置的 baseURL，确保 WebSocket 也走代理
    // 如果 baseURL 是 http://localhost:8000，则转换为 ws://localhost:8000
    // 如果 baseURL 是 https://xxx.com，则转换为 wss://xxx.com
    const wsBaseUrl = serviceBaseUrl.replace(/^http/, 'ws');
    wsUrl = `${wsBaseUrl}/ws/${orderId}`;
  } else {
    // 生产环境：使用当前页面的协议和主机
    // 注意：如果使用了反向代理（如 Nginx），可能需要使用完整的服务地址
    const protocol = window.location.protocol === 'https:' ? 'wss://' : 'ws://';
    const host = window.location.host;
    
    // 如果配置了 serviceBaseUrl，优先使用它（生产环境也可能配置）
    if (serviceBaseUrl && !serviceBaseUrl.includes('localhost')) {
      const wsBaseUrl = serviceBaseUrl.replace(/^http/, 'ws');
      wsUrl = `${wsBaseUrl}/ws/${orderId}`;
    } else {
      wsUrl = `${protocol}${host}/ws/${orderId}`;
    }
  }
  
  try {
    console.log('正在创建 WebSocket 连接', { orderId, wsUrl });
    websocket.value = new WebSocket(wsUrl);
  } catch (error) {
    console.error('WebSocket 连接创建失败', { orderId, error, wsUrl });
    message.error('WebSocket 连接创建失败，请检查网络连接');
    return;
  }

  // Debounce task refresh to avoid flooding the server
  const debouncedRefresh = useDebounceFn(() => {
    getTasks(true);
  }, 1000);

  websocket.value.onopen = () => {
    reconnectAttempts.value = 0; // 重置重连次数
    console.log('WebSocket 连接成功', { orderId, wsUrl });
    
    // 连接成功后，先恢复历史日志（如果为空）
    // 但不要覆盖已有的实时日志
    if (!oscContent.value) {
      restoreHistoryLogs();
    }
  };

  // 节流更新日志内容（每200ms最多更新一次，减少DOM操作）
  const throttledUpdateLog = useThrottleFn(() => {
    if (logBuffer.value.length === 0) return;
    
    isUpdatingLog.value = true;
    try {
      // 批量追加日志
      const newLogs = logBuffer.value.join('');
      const logCount = logBuffer.value.length;
      logBuffer.value = []; // 清空缓冲区
      
      // 追加新日志
      oscContent.value += newLogs;
      
      // 检查并限制日志长度
      trimLogIfNeeded();
    } finally {
      isUpdatingLog.value = false;
    }
  }, 200);

  websocket.value.onmessage = event => {
    console.log('WebSocket 收到消息', { orderId, rawData: event.data });
    try {
      const result = JSON.parse(event.data);
      console.log('WebSocket 消息解析成功', { orderId, type: result.type, data: result.data });
      let logText = '';
      
      if (result.type === 'processlist') {
        // Format processlist data
        logText = '当前SQL SESSION ID的SHOW PROCESSLIST输出:\n';
        for (const key in result.data) {
          logText += `${key}: ${result.data[key]}\n`;
        }
        // processlist 类型替换整个内容
        oscContent.value = logText;
        logBuffer.value = []; // 清空缓冲区
      } else if (result.type === 'ghost-progress') {
        // gh-ost 进度信息
        const progressData = result.data as GhostProgress;
        if (progressData && typeof progressData.percent === 'number') {
          ghostProgress.value = {
            percent: progressData.percent,
            current: progressData.current,
            total: progressData.total,
            eta: progressData.eta,
            operation: progressData.operation
          };
          console.log('gh-ost 进度更新', ghostProgress.value);
        }
      } else if (result.type === 'ghost') {
        // Append ghost logs
        logText = result.data;
        logBuffer.value.push(logText);
        throttledUpdateLog();
        // 检查是否包含 throttle/unthrottle 相关消息
        if (typeof logText === 'string') {
          const lowerMsg = logText.toLowerCase();
          if (lowerMsg.includes('throttle')) {
            // 根据消息判断是暂停还是恢复
            if (lowerMsg.includes('unthrottle') || lowerMsg.includes('resume')) {
              ghostThrottled.value = false;
            } else if (lowerMsg.includes('throttle') && !lowerMsg.includes('unthrottle')) {
              ghostThrottled.value = true;
            }
          }
        }
      } else {
        // Append content (默认类型)
        logText = `${result.data}\n`;
        logBuffer.value.push(logText);
        throttledUpdateLog();
      }
      
      // Auto refresh tasks on any message (防抖处理)
      debouncedRefresh();
    } catch (e) {
      console.error('WebSocket 消息解析失败', { orderId, error: e, rawData: event.data });
      // 解析失败时，直接追加原始数据
      const logText = `${event.data}\n`;
      logBuffer.value.push(logText);
      throttledUpdateLog();
      // Auto refresh tasks even on parse error as it implies activity
      debouncedRefresh();
    }
  };

  websocket.value.onerror = error => {
    console.error('WebSocket 连接错误', { orderId, error, wsUrl });
    // 错误时会在 onclose 中处理重连
  };

  websocket.value.onclose = event => {
    console.log('WebSocket 连接关闭', { orderId, code: event.code, reason: event.reason, wasClean: event.wasClean });
    websocket.value = null;

    // 检查工单状态（重新获取最新状态，避免状态不同步）
    const currentStatus = orderDetail.value?.progress;
    const shouldReconnect = currentStatus && ['执行中', '已批准'].includes(currentStatus);
    
    if (shouldReconnect && reconnectAttempts.value < maxReconnectAttempts) {
      reconnectAttempts.value++;
      console.log('准备重连 WebSocket', { orderId, attempt: reconnectAttempts.value, maxAttempts: maxReconnectAttempts });
      
      reconnectTimer = setTimeout(() => {
        // 重连前再次检查工单状态
        if (orderDetail.value?.progress && ['执行中', '已批准'].includes(orderDetail.value.progress)) {
          initWebSocket();
        } else {
          reconnectAttempts.value = 0; // 重置重连次数
        }
      }, reconnectDelay);
    } else if (reconnectAttempts.value >= maxReconnectAttempts) {
      console.warn('WebSocket 重连次数已达上限', { orderId, attempts: reconnectAttempts.value });
      message.warning('WebSocket 连接失败，已停止自动重连。请刷新页面重试。');
    } else if (!shouldReconnect) {
      console.log('工单状态已改变，不再重连', { orderId, currentStatus });
      reconnectAttempts.value = 0; // 重置重连次数
    }
  };
};

// 从任务列表中恢复历史日志
const restoreHistoryLogs = async () => {
  if (oscContent.value) return; // 如果已有日志，不恢复

  const orderId = route.params.id as string;
  if (!orderId) return;

  try {
    const { data } = await fetchTasks({ order_id: orderId });
    if (data && data.length > 0) {
      let historyLogs = '';
      data.forEach((task: any) => {
        if (task.result) {
          try {
            // result 可能是字符串也可能是对象
            const resultObj = typeof task.result === 'string' ? JSON.parse(task.result) : task.result;
            if (resultObj && resultObj.execute_log) {
              // 添加任务标识，以便区分不同任务的日志
              if (data.length > 1) {
                historyLogs += `\n--- Task ${task.task_id} ---\n`;
              }
              historyLogs += `${resultObj.execute_log}\n`;
            }
          } catch (e) {
            // 解析失败，跳过该任务
          }
        }
      });
      if (historyLogs) {
        oscContent.value = historyLogs;
      }
    }
  } catch (error) {
    // 恢复历史日志失败，忽略错误
  }
};

// 限制日志长度，防止内存泄漏
const trimLogIfNeeded = () => {
  const content = oscContent.value;
  
  // 检查长度限制
  if (content.length > MAX_LOG_LENGTH) {
    // 计算需要保留的行数（大约保留前5MB）
    const lines = content.split('\n');
    if (lines.length > MAX_LOG_LINES) {
      // 保留最新的日志（后50%）
      const keepLines = Math.floor(MAX_LOG_LINES / 2);
      const trimmedLines = lines.slice(-keepLines);
      oscContent.value = `[日志已自动清理，保留最新 ${keepLines} 行]\n${trimmedLines.join('\n')}`;
    } else {
      // 按字节数截断，保留最新的内容
      const trimmed = content.slice(-TRIM_LOG_LENGTH);
      oscContent.value = `[日志已自动清理，保留最新内容]\n${trimmed}`;
    }
  }
};

const closeWebSocket = () => {
  // 清除重连定时器
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  
  if (websocket.value) {
    websocket.value.close();
    websocket.value = null;
  }
  
  reconnectAttempts.value = 0; // 重置重连次数
  logBuffer.value = []; // 清空日志缓冲区
};

onUnmounted(() => {
  closeWebSocket();
  // 清理日志缓冲区
  logBuffer.value = [];
  // 清理日志内容（可选，如果不想保留）
  // oscContent.value = '';
});

// 任务进度统计
const taskStats = computed(() => {
  const tasks = tasksList.value || [];
  const total = tasks.length;
  const completed = tasks.filter(t => t.progress === '已完成').length;
  const processing = tasks.filter(t => t.progress === '执行中').length;
  const failed = tasks.filter(t => t.progress === '已失败').length;
  const unexecuted = tasks.filter(t => t.progress === '未执行').length;
  const paused = tasks.filter(t => t.progress === '已暂停').length;

  return {
    total,
    completed,
    processing,
    failed,
    unexecuted,
    paused
  };
});

// 结果表格数据
const resultColumns = computed<any[]>(() => [
  {
    title: 'TaskID',
    key: 'task_id',
    width: 260
  },
  {
    title: '执行状态',
    key: 'progress', // 修改为 progress
    width: 100,
    render: (row: any) => {
      const statusMap: Record<string, { label: string; type: TagProps['type'] }> = {
        未执行: { label: '未执行', type: 'default' },
        执行中: { label: '执行中', type: 'info' },
        已完成: { label: '成功', type: 'success' },
        已失败: { label: '失败', type: 'error' },
        已跳过: { label: '跳过', type: 'warning' }
      };
      const status = statusMap[row.progress] || { label: row.progress || '未知', type: 'default' };
      return h(NTag, { type: status.type, bordered: false }, { default: () => status.label });
    }
  },
  {
    title: 'SQL语句',
    key: 'sql', // 修改为 sql
    width: 300,
    ellipsis: {
      tooltip: {
        style: { maxWidth: '600px' }
      } as any
    },
    render: (row: any) => {
      return h('span', { class: 'font-mono' }, row.sql);
    }
  },
  {
    title: '影响行数',
    key: 'result.affected_rows', // 修改为 result.affected_rows
    width: 100,
    render: (row: any) => row.result?.affected_rows || 0
  },
  {
    title: '执行时间',
    key: 'result.execute_cost_time', // 修改为 result.execute_cost_time
    width: 100,
    render: (row: any) => {
      return row.result?.execute_cost_time || '-';
    }
  },
  {
    title: '错误信息',
    key: 'result.error', // 修改为 result.error
    width: 200,
    ellipsis: {
      tooltip: {
        style: { maxWidth: '600px' }
      } as any
    },
    render: (row: any) => {
      const error = row.result?.error;
      return error ? h('span', { class: 'text-red-600' }, error) : '-';
    }
  },
  {
    title: '操作',
    key: 'action',
    width: 80,
    fixed: 'right',
    render: (row: any) => {
      // 检查工单状态是否允许执行
      // 增加 '已完成', '已失败' 状态，允许在这些状态下重试失败的任务
      const isOrderExecutable = ['已批准', '执行中', '已复核', '已完成', '已失败'].includes(
        orderDetail.value?.progress || ''
      );
      // 检查任务状态
      const isTaskCompleted = row.progress === '已完成';
      const isTaskRunning = row.progress === '执行中';

      // 始终显示执行按钮，根据状态禁用
      return h(
        NButton,
        {
          size: 'small',
          type: 'primary',
          secondary: true,
          // 禁用条件：工单不可执行 或 任务已完成 或 任务正在执行 或 是定时工单
          disabled: !isOrderExecutable || isTaskCompleted || isTaskRunning || Boolean(orderDetail.value?.schedule_time),
          onClick: () => handleExecuteSingle(row)
        },
        { default: () => '执行' }
      );
    }
  }
]);

const resultData = computed(() => tasksList.value);

// 状态映射
const statusMap = {
  pending: { label: '待审核', type: 'warning' as TagProps['type'] },
  approved: { label: '已批准', type: 'success' as TagProps['type'] },
  rejected: { label: '已驳回', type: 'error' as TagProps['type'] },
  executing: { label: '执行中', type: 'info' as TagProps['type'] },
  completed: { label: '已完成', type: 'success' as TagProps['type'] },
  closed: { label: '已关闭', type: 'default' as TagProps['type'] }
};

// 状态类型映射（基于 progress 中文状态）
const progressTypeMap: Record<string, TagProps['type']> = {
  待审核: 'warning',
  已批准: 'success',
  已驳回: 'error',
  执行中: 'info',
  已完成: 'success',
  已关闭: 'default'
};

// 获取状态标签类型
const getStatusType = computed((): TagProps['type'] => {
  if (!orderDetail.value) return 'default';

  // 优先显示执行结果状态
  if (orderDetail.value.execute_result) {
    const type = orderDetail.value.execute_result;
    if (type === 'success') return 'success';
    if (type === 'error') return 'error';
    if (type === 'warning') return 'warning';
  }

  return progressTypeMap[orderDetail.value.progress] || 'default';
});

// 获取状态标签文本
const getStatusLabel = computed(() => {
  if (!orderDetail.value) return '未知状态';

  // 优先显示执行结果文本
  if (orderDetail.value.execute_result) {
    const type = orderDetail.value.execute_result;
    if (type === 'success') return '全部成功';
    if (type === 'error') return '全部失败';
    if (type === 'warning') return '部分失败';
  }

  return orderDetail.value.progress || '未知状态';
});

// 获取状态颜色 Class
const statusColorClass = computed(() => {
  const type = getStatusType.value;
  const map: Record<string, string> = {
    warning: 'text-orange-500',
    success: 'text-green-500',
    error: 'text-red-500',
    info: 'text-blue-500',
    default: 'text-gray-500'
  };
  return map[type || 'default'] || 'text-gray-500';
});

const actionDisabled = computed(() => {
  const p = orderDetail.value?.progress || '';
  if (p === '待审核') {
    // 必须先通过 SQL 语法检查 (syntaxStatus === 0) 才能点击审核
    return syntaxStatus.value !== 0;
  }
  return Boolean(['已复核', '已驳回', '已关闭'].includes(p));
});

const showGenerateBtn = computed(() => {
  // 生成任务已合并到执行全部中，不再单独显示
  return false;
});

const showExecuteAllBtn = computed(() => {
  if (!orderDetail.value) return false;
  // 如果任务列表不为空，且状态允许，显示执行全部
  const p = orderDetail.value.progress;
  const validStatus = ['已批准', '执行中'].includes(p);
  return validStatus;
});

const closeDisabled = computed(() => {
  const p = orderDetail.value?.progress || '';
  return p === '已完成' ? true : ['已复核', '已驳回', '已关闭'].includes(p);
});

const actionTitle = computed(() => {
  const p = orderDetail.value?.progress || '';
  if (p === '待审核') return '审核';
  if (['已批准', '执行中'].includes(p)) return '更新状态';
  if (p === '已完成') return '复核';
  return '完成';
});

const confirmOkText = computed(() => {
  const p = orderDetail.value?.progress || '';
  if (p === '待审核') return '同意';
  if (['已批准', '执行中'].includes(p)) return '执行完成';
  if (p === '已完成') return '确定';
  return '确定';
});

const confirmCancelText = computed(() => {
  const p = orderDetail.value?.progress || '';
  if (p === '待审核') return '驳回';
  if (['已批准', '执行中'].includes(p)) return '执行中';
  if (p === '已完成') return '取消';
  return '取消';
});

const actionType = computed(() => {
  const p = orderDetail.value?.progress || '';
  if (p === '待审核') return 'approve';
  if (['已批准', '执行中'].includes(p)) return 'feedback';
  if (p === '已完成') return 'review';
  return 'none';
});

// 获取工单详情
const getOrderDetail = async () => {
  const orderId = route.params.id as string;
  if (!orderId) return;

  loading.value = true;
  try {
    const res: any = await fetchOrderDetail(orderId);
    // 后端返回格式: { order, tasks, logs }
    // createFlatRequest 返回: { data: { order, tasks, logs }, error: null, response: {...} }
    if (res.error) {
      window.$message?.error('获取工单详情失败');
      return;
    }
    
    // 后端返回的数据在 res.data 中，格式为 { order, tasks, logs }
    const responseData = res.data || {};
    const orderData = responseData.order || responseData;
    
    if (orderData) {
      // 时间格式化函数
      const formatDateTime = (dateStr: string | null | undefined): string => {
        if (!dateStr) return '';
        try {
          const date = new Date(dateStr);
          return date.toLocaleString('zh-CN', {
            year: 'numeric',
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit'
          });
        } catch (e) {
          return dateStr;
        }
      };
      
      // 解析 JSON 字段辅助函数
      const parseJSONField = (field: any, defaultStatus: string = 'pending'): any[] => {
        if (!field) return [];
        // 如果是字符串，尝试解析为 JSON
        if (typeof field === 'string') {
          try {
            const parsed = JSON.parse(field);
            // 如果是字符串数组，转换为对象数组
            if (Array.isArray(parsed) && parsed.length > 0 && typeof parsed[0] === 'string') {
              return parsed.map((user: string) => ({ user, status: defaultStatus }));
            }
            // 如果已经是对象数组，直接返回
            if (Array.isArray(parsed)) {
              return parsed;
            }
            return [];
          } catch (e) {
            return [];
          }
        }
        // 如果是数组
        if (Array.isArray(field)) {
          // 如果是字符串数组，转换为对象数组
          if (field.length > 0 && typeof field[0] === 'string') {
            return field.map((user: string) => ({ user, status: defaultStatus }));
          }
          // 如果已经是对象数组，直接返回
          return field;
        }
        return [];
      };

      // 字段映射：后端字段 -> 前端字段
      orderDetail.value = {
        ...orderData,
        // 工单标题：后端是 title，前端期望 title（已一致）
        title: orderData.title || '',
        // 环境：后端返回 environment_name，前端使用环境名称
        environment: orderData.environment_name || orderData.environment || '',
        // 实例：后端返回 instance_name (hostname:port 格式)
        instance: orderData.instance_name || orderData.instance_id || '',
        // 创建时间：后端是 CreatedAt，前端期望 created_at（格式化）
        created_at: formatDateTime(orderData.CreatedAt || orderData.created_at),
        // 更新时间：后端是 UpdatedAt，前端期望 updated_at（格式化）
        updated_at: formatDateTime(orderData.UpdatedAt || orderData.updated_at),
        // 计划执行时间：后端是 ScheduleTime，前端期望 schedule_time（格式化）
        schedule_time: formatDateTime(orderData.ScheduleTime || orderData.schedule_time),
        // 审核人：解析 JSON 字段，将字符串数组转换为对象数组
        approver: parseJSONField(orderData.approver, 'pending'),
        // 复核人：解析 JSON 字段，将字符串数组转换为对象数组
        reviewer: parseJSONField(orderData.reviewer, 'pending'),
        // 抄送人：解析 JSON 字段，保持字符串数组格式
        cc: parseJSONField(orderData.cc, 'pending').map((item: any) => 
          typeof item === 'object' ? item.user : item
        ),
        // 执行人：解析 JSON 字段，保持字符串数组格式
        executor: parseJSONField(orderData.executor, 'pending').map((item: any) => 
          typeof item === 'object' ? item.user : item
        )
      } as unknown as OrderDetailVO;
      
      // 获取任务列表和操作日志（如果后端返回了）
      if (responseData.tasks) {
        tasksList.value = responseData.tasks;
      }
      if (responseData.logs) {
        opLogs.value = responseData.logs;
      }
    }
  } catch (error) {
    console.error('获取工单详情失败:', error);
    window.$message?.error('获取工单详情失败');
  } finally {
    loading.value = false;
  }
};

const isExecutor = computed(() => {
  if (!orderDetail.value || !authStore.userInfo.userName) return false;
  return orderDetail.value.executor?.includes(authStore.userInfo.userName);
});

const canEditSchedule = computed(() => {
  const p = orderDetail.value?.progress;
  if (!p) return false;
  return isExecutor.value && !['已完成', '已复核', '已关闭', '已失败'].includes(p);
});

const isEditingSchedule = ref(false);
const newScheduleTime = ref<number | null>(null);

const handleStartEditSchedule = () => {
  if (orderDetail.value?.schedule_time) {
    newScheduleTime.value = new Date(orderDetail.value.schedule_time).getTime();
  }
  isEditingSchedule.value = true;
};

const handleSaveSchedule = async () => {
  if (!orderDetail.value || !newScheduleTime.value) return;
  loading.value = true;
  try {
    const d = new Date(newScheduleTime.value);
    const pad = (n: number) => n.toString().padStart(2, '0');
    const formatted = `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;

    await fetchUpdateOrderSchedule({
      order_id: orderDetail.value.order_id,
      schedule_time: formatted
    });
    window.$message?.success('更新计划时间成功');
    isEditingSchedule.value = false;
    handleRefresh();
  } catch (e: any) {
    window.$message?.error(e?.message || '更新失败');
  } finally {
    loading.value = false;
  }
};

const handleCancelEditSchedule = () => {
  isEditingSchedule.value = false;
  newScheduleTime.value = null;
};

// 获取任务列表
const getTasks = async (force = false) => {
  const orderId = route.params.id as string;
  if (!orderId) return;

  if (!force) {
    const p = orderDetail.value?.progress;
    // 仅在执行阶段（执行中、已完成、已失败、已复核）获取任务列表
    if (!p || !['执行中', '已完成', '已失败', '已复核'].includes(p)) {
      return;
    }
  }

  try {
    const { data } = await fetchTasks({ order_id: orderId });
    if (data) {
      tasksList.value = data;

      // 如果 oscContent 为空且有任务数据，尝试从任务结果中恢复日志
      // 注意：这个逻辑已经移到 restoreHistoryLogs 函数中，但保留作为备用
      if (!oscContent.value && data.length > 0) {
        let historyLogs = '';
        data.forEach((task: any) => {
          if (task.result) {
            try {
              // result 可能是字符串也可能是对象
              const resultObj = typeof task.result === 'string' ? JSON.parse(task.result) : task.result;
              if (resultObj && resultObj.execute_log) {
                // 添加任务标识，以便区分不同任务的日志
                if (data.length > 1) {
                  historyLogs += `\n--- Task ${task.task_id} ---\n`;
                }
                historyLogs += `${resultObj.execute_log}\n`;
              }
            } catch (e) {
              console.error('解析任务结果失败:', e);
            }
          }
        });
        if (historyLogs) {
          oscContent.value = historyLogs;
        }
      }
    }
  } catch (error) {
    console.error('获取任务列表失败:', error);
  }
};

// 获取操作日志
const getOpLogs = async () => {
  const orderId = route.params.id as string;
  if (!orderId) return;
  try {
    const { data } = await fetchOpLogs({ order_id: orderId });
    if (data) {
      opLogs.value = data as any[];
    }
  } catch (error) {
    console.error('获取操作日志失败:', error);
    window.$message?.error('获取操作日志失败');
  }
};

/* const getTaskPreview = async () => {
  const orderId = route.params.id as string;
  if (!orderId) return;
  try {
    const { data } = await fetchPreviewTasks({ order_id: orderId });
    if (data) {
      taskStats.value = data as any;
    }
  } catch {}
}; */

// 操作函数（临时占位）
const handleRetweet = () => {
  console.log('转推操作');
};

const handleHook = () => {
  console.log('Hook操作');
};

const handleClose = () => {
  console.log('关闭工单');
};

const handleExecute = () => {
  console.log('执行工单');
};

const handleRefresh = async () => {
  refreshLoading.value = true;
  await Promise.all([
    getOrderDetail(),
    getOpLogs(),
    // getTaskPreview(),
    getTasks()
  ]);
  refreshLoading.value = false;
};

// 组件挂载时获取数据
onMounted(async () => {
  await getOrderDetail();
  getOpLogs();
  // getTaskPreview();
  await getTasks();
  // 如果工单在执行中，建立 WebSocket 连接
  // 注意：initWebSocket 内部会检查工单状态
  initWebSocket();
});

// 监听工单状态变化，如果从非执行状态变为执行状态，建立连接
watch(
  () => orderDetail.value?.progress,
  (newStatus, oldStatus) => {
    // 如果工单状态变为执行中，且之前没有连接，则建立连接
    if (newStatus === '执行中' && oldStatus !== '执行中' && !websocket.value) {
      initWebSocket();
    }
    // 如果工单状态变为非执行状态，关闭连接并清空进度信息
    if (newStatus && !['执行中', '已批准'].includes(newStatus) && websocket.value) {
      closeWebSocket();
      // 清空 gh-ost 进度信息
      ghostProgress.value = null;
    }
  }
);
// SQL编辑器事件处理
const handleSqlPageChange = (page: number) => {
  console.log('SQL页码变化:', page);
};

const handleSqlPageSizeChange = (pageSize: number) => {
  console.log('SQL页大小变化:', pageSize);
};

const showActionModal = () => {
  actionVisible.value = true;
};

const handleActionCancel = async () => {
  if (actionType.value === 'approve' && confirmCancelText.value === '驳回') {
    if (confirmMsg.value.length > 256) {
      window.$message?.error('消息长度不能超过256个字符');
      return;
    }
    loading.value = true;
    try {
      await fetchApproveOrder({ status: 'reject', msg: confirmMsg.value, order_id: orderDetail.value?.order_id } as any);
      handleRefresh();
      actionVisible.value = false;
    } catch (e: any) {
      window.$message?.error(e?.message || '操作失败');
    } finally {
      loading.value = false;
      confirmMsg.value = '';
    }
    return;
  }

  if (actionType.value === 'feedback' && confirmCancelText.value === '执行中') {
    loading.value = true;
    try {
      await fetchFeedbackOrder({ progress: '执行中', msg: confirmMsg.value, order_id: orderDetail.value?.order_id } as any);
      handleRefresh();
      actionVisible.value = false;
    } catch (e: any) {
      window.$message?.error(e?.message || '操作失败');
    } finally {
      loading.value = false;
      confirmMsg.value = '';
    }
    return;
  }

  actionVisible.value = false;
};

const handleActionOk = async () => {
  if (actionType.value === 'approve' && confirmOkText.value === '同意') {
    if (syntaxStatus.value === 1) {
      window.$message?.error('语法检查未通过，不允许审核通过');
      return;
    }
  }
  actionVisible.value = false;
  const orderId = orderDetail.value?.order_id || '';
  if (confirmMsg.value.length > 256) {
    window.$message?.error('消息长度不能超过256个字符');
    return;
  }
  loading.value = true;
  try {
    if (actionType.value === 'approve') {
      const status = confirmOkText.value === '同意' ? 'pass' : 'reject';
      await fetchApproveOrder({ status, msg: confirmMsg.value, order_id: orderId } as any);
    } else if (actionType.value === 'feedback') {
      const progress = confirmOkText.value === '执行完成' ? '已完成' : '执行中';
      await fetchFeedbackOrder({ progress, msg: confirmMsg.value, order_id: orderId } as any);
    } else if (actionType.value === 'review') {
      await fetchReviewOrder({ msg: confirmMsg.value, order_id: orderId } as any);
    }
    handleRefresh();
  } catch (e: any) {
    window.$message?.error(e?.message || '操作失败');
  } finally {
    loading.value = false;
    confirmMsg.value = '';
  }
};

const showCloseModal = () => {
  closeVisible.value = true;
};

const handleCloseCancel = () => {
  closeVisible.value = false;
};

const handleCloseOk = async () => {
  const orderId = orderDetail.value?.order_id || '';
  loading.value = true;
  try {
    await fetchCloseOrder({ msg: confirmMsg.value, order_id: orderId } as any);
    handleRefresh();
  } catch (e: any) {
    window.$message?.error(e?.message || '关闭失败');
  } finally {
    loading.value = false;
    closeVisible.value = false;
    confirmMsg.value = '';
  }
};

const handleGenerateTasks = async () => {
  if (!orderDetail.value) return;
  executeLoading.value = true;
  try {
    await fetchGenerateTasks({ order_id: orderDetail.value.order_id } as any);
    window.$message?.success('已生成任务');
    await getTasks(true); // 生成任务后刷新任务列表
    // getTaskPreview(); // 刷新进度
  } catch (e: any) {
    window.$message?.error(e?.message || '生成任务失败');
  } finally {
    executeLoading.value = false;
  }
};

const handleExecuteAll = async () => {
  if (!orderDetail.value) return;
  activeTab.value = 'osc-progress';
  // 在执行任务之前，先建立 WebSocket 连接以接收实时日志
  initWebSocket(true);
  executeLoading.value = true;
  try {
    // 如果没有任务，先自动生成任务
    // if (tasksList.value.length === 0) {
    //   const { error: genError } = await fetchGenerateTasks({ order_id: orderDetail.value.order_id } as any);
    //   if (genError) return;
    //   // 刷新任务列表
    //   await getTasks(true);
    // }

    const { data, error } = await fetchExecuteAllTasks({ order_id: orderDetail.value.order_id } as any);
    if (error) return;

    // 从 data.data.message 获取消息
    const msgType = data?.data?.type;
    const msgContent = data?.data?.message || data?.message || '已触发全部执行';

    if (msgType === 'error') {
      window.$message?.error(msgContent);
    } else if (msgType === 'warning') {
      window.$message?.warning(msgContent);
    } else {
      window.$message?.success(msgContent);
    }

    handleRefresh();
  } catch (e: any) {
    window.$message?.error(e?.message || '执行失败');
  } finally {
    executeLoading.value = false;
  }
};

const handleExecuteSingle = async (row: any) => {
  activeTab.value = 'osc-progress';
  // 在执行任务之前，先建立 WebSocket 连接以接收实时日志
  initWebSocket(true);
  try {
    const res: any = await fetchExecuteSingleTask({ task_id: row.id } as any);
    // request 的 transformBackendResponse 返回 response.data.data
    // 所以 res.data 就是 { message: "执行成功", type: "success" }
    const msgContent = res?.data?.message || '已触发执行';
    const msgType = res?.data?.type || 'success';
    
    if (msgType === 'error') {
      window.$message?.error(msgContent);
    } else if (msgType === 'warning') {
      window.$message?.warning(msgContent);
    } else {
      window.$message?.success(msgContent);
    }
    // 刷新单行状态太麻烦，直接刷新列表
    handleRefresh();
  } catch (e: any) {
    window.$message?.error(e?.message || '执行失败');
  }
};

// gh-ost 控制处理函数
const handleGhostControl = async (action: 'throttle' | 'unthrottle' | 'chunk-size') => {
  if (!orderDetail.value) return;

  // chunk-size 操作需要先验证值（在设置 loading 之前验证，避免 loading 状态异常）
  if (action === 'chunk-size') {
    if (!ghostChunkSize.value || ghostChunkSize.value <= 0) {
      window.$message?.warning('请输入有效的 chunk-size 值（大于 0）');
      return;
    }
  }

  ghostControlLoading.value = true;
  try {
    const data: any = {
      order_id: orderDetail.value.order_id,
      action
    };

    if (action === 'chunk-size') {
      data.value = ghostChunkSize.value;
    }

    const res: any = await fetchControlGhost(data);
    
    // createFlatRequest 返回格式: { data, error, response }
    // 检查是否有错误（全局拦截器已经处理了 code 检查和错误消息显示）
    if (res.error) {
      // 不手动显示错误消息，全局拦截器已经显示过了
      return;
    }
    
    // 操作成功，根据操作类型更新状态和提示
    if (action === 'throttle') {
      ghostThrottled.value = true;
      window.$message?.success('已暂停执行');
    } else if (action === 'unthrottle') {
      ghostThrottled.value = false;
      window.$message?.success('已恢复执行');
    } else if (action === 'chunk-size') {
      window.$message?.success(`速度已调节为 ${ghostChunkSize.value}`);
    }
  } catch (e: any) {
    // createFlatRequest 不会抛出异常，但如果真的发生异常，全局拦截器也会处理
    // 这里只做兜底处理，不重复显示错误消息
    console.error('gh-ost 控制异常:', e);
  } finally {
    ghostControlLoading.value = false;
  }
};

// gh-ost 取消处理函数
const handleGhostCancel = async () => {
  if (!orderDetail.value) return;

  dialog.warning({
    title: '确认取消',
    content: '确定要取消 gh-ost 执行吗？此操作不可恢复。',
    positiveText: '确认',
    negativeText: '取消',
    onPositiveClick: async () => {
      ghostControlLoading.value = true;
      try {
        await fetchControlGhost({
          order_id: orderDetail.value!.order_id,
          action: 'panic'
        });
        window.$message?.warning('已发送取消命令，gh-ost 将停止执行');
        // 清空进度信息
        ghostProgress.value = null;
        ghostThrottled.value = false;
      } catch (e: any) {
        window.$message?.error(e?.message || '操作失败');
      } finally {
        ghostControlLoading.value = false;
      }
    }
  });
};

const hookForm = ref({ order_id: '', title: '', db_type: '', schema: '' });
const environmentOptions = ref<{ label: string; value: any }[]>([]);
const targetList = ref<{ environment: any; instance_id: any; schema: any }[]>([{} as any]);
const instancesOptions = ref<Record<number, { label: string; value: any }[]>>({});
const schemasOptions = ref<Record<number, { label: string; value: any }[]>>({});

const showHookModal = async () => {
  if (!orderDetail.value) return;
  hookForm.value = {
    order_id: orderDetail.value.order_id,
    title: orderDetail.value.title,
    db_type: orderDetail.value.db_type,
    schema: orderDetail.value.schema
  } as any;
  hookVisible.value = true;
  const envs = await fetchOrdersEnvironments({ is_page: false });
  environmentOptions.value = (envs.data || []).map((e: any) => ({ label: e.name, value: e.id }));
};

const hideHookModal = () => {
  hookVisible.value = false;
};

const addTarget = () => {
  targetList.value.push({} as any);
};

const removeTarget = (idx: number) => {
  targetList.value.splice(idx, 1);
};

const changeEnv = async (idx: number, val: any) => {
  const params = { id: val, db_type: hookForm.value.db_type, is_page: false } as any;
  const res = await fetchOrdersInstances(params);
  instancesOptions.value[idx] = (res.data || []).map((i: any) => ({ label: i.remark, value: i.instance_id }));
  schemasOptions.value[idx] = [];
  targetList.value[idx].instance_id = null;
  targetList.value[idx].schema = null;
};

const changeInstance = async (idx: number, val: any) => {
  const params = { instance_id: val, is_page: false } as any;
  const res = await fetchOrdersSchemas(params);
  schemasOptions.value[idx] = (res.data || []).map((s: any) => ({ label: s.schema, value: s.schema }));
};

const submitHook = async () => {
  loading.value = true;
  try {
    const target = targetList.value.map(i => ({
      environment: i.environment,
      instance_id: i.instance_id,
      schema: i.schema
    }));
    const progress = resetToPending.value ? '待审核' : '已批准';
    await fetchHookOrder({ ...hookForm.value, target, progress } as any);
    window.$message?.success('Hook成功');
    hideHookModal();
  } catch (e: any) {
    window.$message?.error(e?.message || 'Hook失败');
  } finally {
    loading.value = false;
  }
};
</script>

<template>
  <div class="order-detail-page min-h-500px flex-col-stretch gap-16px">
    <NSpin :show="loading">
      <div v-if="orderDetail">
        <!-- 新的顶部布局 -->
        <div class="order-header-container">
          <!-- 顶部标题栏 -->
          <div class="mb-4 flex flex-col gap-3" :class="appStore.isMobile ? '' : 'items-center justify-between flex-row'">
            <div class="flex items-center gap-2">
              <NButton text class="mr-2" @click="router.back()">
                <template #icon>
                  <div class="i-ic:round-arrow-back text-xl" />
                </template>
              </NButton>
              <h1 class="m-0 text-2xl font-bold" :class="appStore.isMobile ? 'text-lg' : ''">{{ orderDetail?.title || '工单详情' }}</h1>
              <NTag type="primary" size="small" bordered>#{{ orderDetail?.order_id }}</NTag>
            </div>
            <div class="flex gap-2 flex-wrap">
              <NButton type="primary" ghost :size="appStore.isMobile ? 'tiny' : 'small'" :loading="refreshLoading" @click="handleRefresh">
                <template #icon>
                  <div class="i-ic:round-refresh" />
                </template>
                <span v-if="!appStore.isMobile">刷新</span>
              </NButton>
              <!-- 操作按钮组 -->
              <NButton
                v-if="actionType !== 'none'"
                type="primary"
                :size="appStore.isMobile ? 'tiny' : 'small'"
                :disabled="actionDisabled"
                @click="showActionModal"
              >
                {{ actionTitle }}
              </NButton>

              <NButton v-if="orderDetail?.progress === '已复核'" type="info" :size="appStore.isMobile ? 'tiny' : 'small'" @click="handleHook">
                <template #icon>
                  <div class="i-ant-design:link-outlined" />
                </template>
                <span v-if="!appStore.isMobile">Hook</span>
              </NButton>

              <NButton type="error" ghost :size="appStore.isMobile ? 'tiny' : 'small'" :disabled="closeDisabled" @click="showCloseModal">
                <template #icon>
                  <div class="i-ant-design:close-circle-outlined" />
                </template>
                <span v-if="!appStore.isMobile">关闭工单</span>
              </NButton>

              <NButton
                v-if="showGenerateBtn"
                type="warning"
                ghost
                :size="appStore.isMobile ? 'tiny' : 'small'"
                :loading="executeLoading"
                @click="handleGenerateTasks"
              >
                <template #icon>
                  <div class="i-ant-design:thunderbolt-outlined" />
                </template>
                <span v-if="!appStore.isMobile">生成任务</span>
              </NButton>

              <NButton
                v-if="showExecuteAllBtn"
                type="success"
                ghost
                :size="appStore.isMobile ? 'tiny' : 'small'"
                :loading="executeLoading"
                :disabled="!!orderDetail?.schedule_time"
                @click="handleExecuteAll"
              >
                <template #icon>
                  <div class="i-ant-design:thunderbolt-filled" />
                </template>
                <span v-if="!appStore.isMobile">执行全部</span>
              </NButton>
            </div>
          </div>

          <!-- 工单基本信息区域 -->
          <div class="order-info-section">
            <div class="flex flex-col gap-4" :class="appStore.isMobile ? '' : 'flex-row gap-6'">
              <!-- 左侧信息列表 -->
              <div class="grid flex-1 gap-x-8 gap-y-4 text-sm" :class="appStore.isMobile ? 'grid-cols-1' : 'grid-cols-3'">
                <div class="info-item">
                  <span class="label">申请人：</span>
                  <span class="value font-medium">{{ orderDetail?.applicant }}</span>
                </div>
                <div class="info-item">
                  <span class="label">工单环境：</span>
                  <NTag type="error" size="small" :bordered="false">{{ orderDetail?.environment }}</NTag>
                </div>
                <div class="info-item">
                  <span class="label">DB类型：</span>
                  <span class="value">{{ orderDetail?.db_type }}</span>
                </div>
                <div class="info-item">
                  <span class="label">工单类型：</span>
                  <span class="value">{{ orderDetail?.sql_type }}</span>
                </div>
                <div class="info-item min-w-0">
                  <span class="label flex-shrink-0 whitespace-nowrap">DB实例：</span>
                  <NTooltip trigger="hover">
                    <template #trigger>
                      <span class="value min-w-0 truncate">{{ orderDetail?.instance }}</span>
                    </template>
                    {{ orderDetail?.instance }}
                  </NTooltip>
                </div>
                <div class="info-item">
                  <span class="label">库名：</span>
                  <span class="value text-blue-600">{{ orderDetail?.schema }}</span>
                </div>
                <div v-if="orderDetail?.sql_type === 'EXPORT'" class="info-item">
                  <span class="label">文件格式：</span>
                  <span class="value">{{ orderDetail?.export_file_format }}</span>
                </div>
                <div class="info-item">
                  <span class="label">创建时间：</span>
                  <span class="value">{{ orderDetail?.created_at }}</span>
                </div>
                <div class="info-item">
                  <span class="label">更新时间：</span>
                  <span class="value">{{ orderDetail?.updated_at }}</span>
                </div>
                <div v-if="orderDetail?.schedule_time" class="info-item">
                  <span class="label">计划时间：</span>
                  <div v-if="isEditingSchedule" class="flex items-center gap-2">
                    <NDatePicker v-model:value="newScheduleTime" type="datetime" size="small" clearable />
                    <NButton size="tiny" type="primary" @click="handleSaveSchedule">保存</NButton>
                    <NButton size="tiny" @click="handleCancelEditSchedule">取消</NButton>
                  </div>
                  <div v-else class="flex items-center gap-2">
                    <span class="value">{{ orderDetail?.schedule_time }}</span>
                    <NButton v-if="canEditSchedule" size="tiny" type="primary" text @click="handleStartEditSchedule">
                      <template #icon>
                        <div class="i-ant-design:edit-outlined" />
                      </template>
                    </NButton>
                  </div>
                </div>
              </div>

              <!-- 右侧状态展示 -->
              <div
                class="status-display-section flex flex-col items-center justify-center px-8 dark:border-gray-800"
                :class="appStore.isMobile ? 'border-t border-gray-100 pt-4 mt-4 min-w-full' : 'min-w-[160px] border-l border-gray-100'"
              >
                <div class="sci-fi-status-container" :class="getStatusType">
                  <div class="status-label-mini">CURRENT STATUS</div>
                  <div class="status-content">
                    <div class="status-indicator">
                      <div class="status-dot"></div>
                      <div class="status-ping"></div>
                    </div>
                    <span class="status-text">{{ getStatusLabel }}</span>
                  </div>
                  <div class="corner-accents">
                    <div class="corner top-left"></div>
                    <div class="corner top-right"></div>
                    <div class="corner bottom-left"></div>
                    <div class="corner bottom-right"></div>
                  </div>
                  <div class="scan-line"></div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 步骤条区域 -->
        <NCard size="small" class="mb-16px" title="审批流程">
          <template #header-extra>
            <NButton text size="small" @click="showProgress = !showProgress">
              {{ showProgress ? '收起' : '展开' }}
              <template #icon>
                <div :class="showProgress ? 'i-ic:round-keyboard-arrow-up' : 'i-ic:round-keyboard-arrow-down'" />
              </template>
            </NButton>
          </template>

          <NCollapseTransition :show="showProgress">
            <NSteps :current="currentStep" :status="currentStepStatus" class="mb-6 pt-4">
              <NStep title="创建工单" description="提交申请" />
              <NStep title="待审核" description="等待审批" />
              <NStep title="审核结果" description="批准或驳回" />
              <NStep title="执行中" description="任务运行中" />
              <NStep title="执行结果" description="完成或失败" />
              <NStep title="已复核" description="最终确认" />
            </NSteps>
          </NCollapseTransition>

          <div
            v-if="orderDetail?.approver?.length || orderDetail?.reviewer?.length || orderDetail?.cc?.length"
            class="mt-4 border-t border-gray-100 pt-4 dark:border-gray-800"
          >
            <div class="flex flex-wrap gap-8">
              <div v-if="orderDetail?.approver?.length" class="flex items-center gap-2">
                <span class="text-gray-500">审核人:</span>
                <NSpace size="small">
                  <NTag
                    v-for="item in orderDetail.approver"
                    :key="item.user"
                    size="small"
                    :type="item.status === 'pass' ? 'success' : item.status === 'reject' ? 'error' : 'default'"
                    :bordered="false"
                  >
                    {{ item.user }}
                  </NTag>
                </NSpace>
              </div>

              <div v-if="orderDetail?.reviewer?.length" class="flex items-center gap-2">
                <span class="text-gray-500">复核人:</span>
                <NSpace size="small">
                  <NTag
                    v-for="item in orderDetail.reviewer"
                    :key="item.user"
                    size="small"
                    :type="item.status === 'pass' ? 'success' : 'default'"
                    :bordered="false"
                  >
                    {{ item.user }}
                  </NTag>
                </NSpace>
              </div>

              <div v-if="orderDetail?.cc?.length" class="flex items-center gap-2">
                <span class="text-gray-500">抄送人:</span>
                <NSpace size="small">
                  <NTag v-for="user in orderDetail.cc" :key="user" size="small" :bordered="false">
                    {{ user }}
                  </NTag>
                </NSpace>
              </div>
            </div>
          </div>
        </NCard>

        <!-- 中间区域：两栏布局 -->
        <div class="middle-content-container" :class="appStore.isMobile ? 'mobile-layout' : ''">
          <!-- 左侧：进度信息 -->
          <div class="progress-section">
            <!-- 进度信息 -->
            <NCard title="进度信息" size="small" class="mb-16px">
              <div class="progress-info">
                <div v-if="taskStats" class="progress-item">
                  <div class="task-stats">
                    <NSpace :size="appStore.isMobile ? 'small' : 'small'" :wrap="appStore.isMobile">
                      <NTag :bordered="false" type="primary" :size="appStore.isMobile ? 'tiny' : 'small'">任务数: {{ taskStats.total }}</NTag>
                      <NTag :bordered="false" type="success" :size="appStore.isMobile ? 'tiny' : 'small'">成功数: {{ taskStats.completed }}</NTag>
                      <NTag :bordered="false" type="default" :size="appStore.isMobile ? 'tiny' : 'small'">未执行: {{ taskStats.unexecuted }}</NTag>
                      <NTag :bordered="false" type="error" :size="appStore.isMobile ? 'tiny' : 'small'">已失败: {{ taskStats.failed }}</NTag>
                      <NTag :bordered="false" type="info" :size="appStore.isMobile ? 'tiny' : 'small'">执行中: {{ taskStats.processing }}</NTag>
                      <NTag :bordered="false" type="warning" :size="appStore.isMobile ? 'tiny' : 'small'">已暂停: {{ taskStats.paused }}</NTag>
                    </NSpace>
                  </div>
                </div>
                <!-- gh-ost 执行进度和控制 -->
                <NDivider v-if="isGhostOrder" style="margin: 12px 0;" />
                <div v-if="isGhostOrder" class="progress-item">
                  <div class="ghost-control-header">
                    <span class="progress-label">gh-ost 执行控制：</span>
                    <NSpace :size="8" :wrap="appStore.isMobile" style="flex-wrap: wrap;">
                      <NButton
                        v-if="!ghostThrottled"
                        type="warning"
                        size="small"
                        :loading="ghostControlLoading"
                        :disabled="ghostControlLoading"
                        @click="handleGhostControl('throttle')"
                      >
                        暂停
                      </NButton>
                      <NButton
                        v-else
                        type="primary"
                        size="small"
                        :loading="ghostControlLoading"
                        :disabled="ghostControlLoading"
                        @click="handleGhostControl('unthrottle')"
                      >
                        恢复
                      </NButton>
                      <NButton
                        type="error"
                        size="small"
                        :loading="ghostControlLoading"
                        :disabled="ghostControlLoading"
                        @click="handleGhostCancel"
                      >
                        取消
                      </NButton>
                      <div style="display: flex; align-items: center; gap: 4px;">
                        <NInputNumber
                          v-model:value="ghostChunkSize"
                          size="small"
                          :min="100"
                          :max="10000"
                          :step="100"
                          :disabled="ghostControlLoading"
                          style="width: 100px"
                          placeholder="速度"
                        />
                        <NButton
                          type="info"
                          size="small"
                          :loading="ghostControlLoading"
                          :disabled="ghostControlLoading || !ghostChunkSize"
                          @click="handleGhostControl('chunk-size')"
                        >
                          调节速度
                        </NButton>
                      </div>
                    </NSpace>
                  </div>
                  <!-- 进度信息显示 -->
                  <div v-if="displayGhostProgress && displayGhostProgress.percent !== undefined" class="ghost-progress-container">
                    <div style="margin-bottom: 4px;">
                      <span style="font-size: 12px; color: #666;">
                        {{ displayGhostProgress.percent.toFixed(2) }}%
                        <span v-if="displayGhostProgress.current !== undefined && displayGhostProgress.total !== undefined">
                          ({{ displayGhostProgress.current }}/{{ displayGhostProgress.total }})
                        </span>
                        <span v-if="displayGhostProgress.eta" style="margin-left: 8px; color: #999;">
                          ETA: {{ displayGhostProgress.eta }}
                        </span>
                      </span>
                    </div>
                    <NProgress
                      type="line"
                      :percentage="displayGhostProgress.percent"
                      :show-indicator="false"
                      :border-radius="4"
                      :fill-border-radius="4"
                      :status="displayGhostProgress.percent === 100 ? 'success' : 'default'"
                      :height="appStore.isMobile ? 16 : 20"
                    />
                  </div>
                </div>
              </div>
            </NCard>

            <!-- 操作日志 -->
            <NCard v-if="opLogs.length" title="操作日志" size="small">
              <NTimeline>
                <NTimelineItem
                  v-for="(log, idx) in opLogs"
                  :key="log.id ?? log.order_id ?? idx"
                  :title="log.msg ?? log.action ?? '日志'"
                  :content="log.msg ?? log.comment ?? ''"
                  :time="log.updated_at ?? log.operateTime ?? ''"
                />
              </NTimeline>
            </NCard>
          </div>

          <!-- 右侧：主要内容 -->
          <div class="main-content-section">
            <NCard size="small" class="mb-16px">
              <NTabs v-model:value="activeTab" type="line" animated @update:value="handleTabChange">
                <template #suffix>
                  <NSpace v-if="activeTab === 'sql-content'" align="center" :size="appStore.isMobile ? 6 : 12" :wrap="appStore.isMobile">
                    <NButton :size="appStore.isMobile ? 'tiny' : 'small'" type="primary" secondary @click="handleFormatSQL">
                      <template v-if="appStore.isMobile" #icon>
                        <div class="i-ant-design:format-painter-outlined" />
                      </template>
                      <span v-if="!appStore.isMobile">格式化</span>
                    </NButton>
                    <NButton :size="appStore.isMobile ? 'tiny' : 'small'" type="primary" secondary :loading="checking" @click="handleSyntaxCheck">
                      <template v-if="appStore.isMobile" #icon>
                        <div class="i-ant-design:check-circle-outlined" />
                      </template>
                      <span v-if="!appStore.isMobile">sql审核</span>
                    </NButton>
                  </NSpace>
                </template>
                <!-- SQL内容标签页 -->
                <NTabPane name="sql-content" tab="SQL内容">
                  <div class="tab-content">
                    <ReadonlySqlEditor
                      :sql-content="displaySqlContent"
                      :show-pagination="true"
                      :page-size="10"
                      :theme="theme"
                      :height="appStore.isMobile ? '300px' : '500px'"
                      @page-change="handleSqlPageChange"
                      @page-size-change="handleSqlPageSizeChange"
                    />
                    <NCard v-if="syntaxRows.length" title="语法检查结果" class="mt-4" size="small">
                      <NDataTable
                        :columns="syntaxColumns"
                        :data="syntaxRows"
                        :pagination="{ pageSize: 10 }"
                        size="small"
                        :scroll-x="appStore.isMobile ? 800 : 1200"
                      />
                    </NCard>
                  </div>
                </NTabPane>

                <!-- 评论标签页 -->
                <!--
 <NTabPane name="comments" tab="评论">
                  <div class="tab-content">
                    <NEmpty description="暂无评论" />
                  </div>
                </NTabPane> 
-->

                <!-- 结果标签页 -->
                <NTabPane name="results" tab="执行结果">
                  <div class="tab-content">
                    <div class="result-summary mb-16px">
                      <NSpace :size="appStore.isMobile ? 'small' : 'medium'" :wrap="appStore.isMobile">
                        <NStatistic label="总执行数">
                          <span :class="appStore.isMobile ? 'text-14px' : 'text-16px'" class="font-bold">{{ taskStats?.total || 0 }}</span>
                        </NStatistic>
                        <NStatistic label="成功数">
                          <span :class="appStore.isMobile ? 'text-14px' : 'text-16px'" class="text-green-600 font-bold">{{ taskStats?.completed || 0 }}</span>
                        </NStatistic>
                        <NStatistic label="失败数">
                          <span :class="appStore.isMobile ? 'text-14px' : 'text-16px'" class="text-red-600 font-bold">{{ taskStats?.failed || 0 }}</span>
                        </NStatistic>
                        <NStatistic label="警告数">
                          <span :class="appStore.isMobile ? 'text-14px' : 'text-16px'" class="text-orange-600 font-bold">{{ taskStats?.unexecuted || 0 }}</span>
                        </NStatistic>
                      </NSpace>
                    </div>
                    <NDataTable
                      :columns="resultColumns"
                      :data="resultData"
                      :pagination="{ pageSize: 10 }"
                      :bordered="false"
                      size="small"
                      :scroll-x="appStore.isMobile ? 800 : 1000"
                    />
                  </div>
                </NTabPane>

                <!-- 执行进度标签页 -->
                <NTabPane name="osc-progress" tab="执行进度">
                  <div class="tab-content">
                    <LogViewer :content="oscContent" :height="appStore.isMobile ? '300px' : '500px'" :theme="theme" />
                  </div>
                </NTabPane>
              </NTabs>
            </NCard>

            <!-- 操作按钮 - 临时隐藏 -->
            <NCard title="操作" size="small" class="mb-16px" style="display: none">
              <NSpace>
                <NButton type="primary" @click="handleRetweet">转推</NButton>
                <NButton @click="handleHook">Hook</NButton>
                <NButton type="error" @click="handleClose">关闭工单</NButton>
                <NButton type="success" @click="handleExecute">执行工单</NButton>
              </NSpace>
            </NCard>
          </div>
        </div>

        <!-- 底部：标签页区域 - 已移除，整合到上方 -->
      </div>

      <div v-else class="min-h-200px flex-center">
        <NEmpty description="工单不存在或已被删除" />
      </div>
    </NSpin>

    <NModal v-model:show="actionVisible" preset="dialog" title="请输入附加信息">
      <NInput v-model:value="confirmMsg" type="textarea" :autosize="{ minRows: 3, maxRows: 8 }" />
      <template #action>
        <NSpace>
          <NButton @click="handleActionCancel">{{ confirmCancelText }}</NButton>
          <NButton type="primary" :loading="loading" @click="handleActionOk">{{ confirmOkText }}</NButton>
        </NSpace>
      </template>
    </NModal>

    <NModal v-model:show="closeVisible" preset="dialog" title="请输入附加信息">
      <NInput v-model:value="confirmMsg" type="textarea" :autosize="{ minRows: 3, maxRows: 5 }" />
      <template #action>
        <NSpace>
          <NButton @click="handleCloseCancel">取消</NButton>
          <NButton type="primary" :loading="loading" @click="handleCloseOk">确定</NButton>
        </NSpace>
      </template>
    </NModal>

    <NModal v-model:show="hookVisible" preset="dialog" title="HOOK工单" :style="{ width: appStore.isMobile ? '95%' : '65%' }">
      <NForm :model="hookForm">
        <NFormItem label="工单ID"><NInput v-model:value="hookForm.order_id" disabled /></NFormItem>
        <NFormItem label="当前工单"><NInput v-model:value="hookForm.title" disabled /></NFormItem>
        <NFormItem label="DB类型"><NInput v-model:value="hookForm.db_type" disabled /></NFormItem>
        <NFormItem label="当前库"><NInput v-model:value="hookForm.schema" disabled /></NFormItem>
        <NFormItem label="审核状态">
          <NSwitch v-model:value="resetToPending" :round="true" />
        </NFormItem>
        <NFormItem label="目标库">
          <NCard>
            <div v-for="(item, idx) in targetList" :key="idx" class="mb-8px">
              <div class="grid gap-12px" :class="appStore.isMobile ? 'grid-cols-1' : 'grid-cols-12'">
                <div :class="appStore.isMobile ? 'col-span-1' : 'col-span-4'">
                  <NFormItem label="环境">
                    <NSelect
                      v-model:value="item.environment"
                      :options="environmentOptions"
                      clearable
                      filterable
                      @update:value="val => changeEnv(idx, val)"
                    />
                  </NFormItem>
                </div>
                <div :class="appStore.isMobile ? 'col-span-1' : 'col-span-4'">
                  <NFormItem label="实例">
                    <NSelect
                      v-model:value="item.instance_id"
                      :options="instancesOptions[idx] || []"
                      clearable
                      filterable
                      @update:value="val => changeInstance(idx, val)"
                    />
                  </NFormItem>
                </div>
                <div :class="appStore.isMobile ? 'col-span-1' : 'col-span-3'">
                  <NFormItem label="库名">
                    <NSelect v-model:value="item.schema" :options="schemasOptions[idx] || []" clearable filterable />
                  </NFormItem>
                </div>
                <div :class="appStore.isMobile ? 'col-span-1 flex items-center' : 'col-span-1 flex items-center'">
                  <NButton v-if="targetList.length > 1" tertiary @click="removeTarget(idx)">删除</NButton>
                </div>
              </div>
            </div>
            <NButton tertiary @click="addTarget">新增一行</NButton>
          </NCard>
        </NFormItem>
      </NForm>
      <template #action>
        <NSpace>
          <NButton @click="hideHookModal">取消</NButton>
          <NButton type="primary" :loading="loading" @click="submitHook">确定</NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
/* 页面容器样式 */
.order-detail-page {
  padding: 16px;
  overflow-y: auto;
  max-height: calc(100vh - 120px); /* 减去导航栏等高度 */
}

/* 基础布局样式 */
.order-header-container {
  background: white;
  border-radius: 8px;
  padding: 24px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.dark .order-header-container {
  background: #1f1f1f;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
}

/* 基本信息样式 */
.info-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.info-label {
  color: #666;
  font-size: 14px;
  white-space: nowrap;
}

.dark .info-label {
  color: #999;
}

.info-value {
  color: #333;
  font-size: 14px;
  font-weight: 500;
}

.dark .info-value {
  color: #e5e5e5;
}

/* 中间区域两栏布局样式 */
.middle-content-container {
  display: grid;
  grid-template-columns: 1fr 3fr;
  gap: 24px;
  margin-bottom: 24px;
}

/* 移动端单列布局 */
.middle-content-container.mobile-layout {
  grid-template-columns: 1fr;
  gap: 16px;
}

.progress-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* Sci-Fi Status Styles */
.sci-fi-status-container {
  position: relative;
  padding: 8px 16px;
  background: transparent;
  /* border: 1px solid rgba(0, 0, 0, 0.05); */
  border-radius: 4px;
  overflow: hidden;
  transition: all 0.3s ease;
  /* Default color var */
  --status-color: #888;
  --status-bg: transparent; /* Make background transparent */
  min-width: 120px;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.dark .sci-fi-status-container {
  background: transparent;
  /* border-color: rgba(255, 255, 255, 0.08); */
}

.sci-fi-status-container.warning {
  --status-color: #f97316;
}
.sci-fi-status-container.success {
  --status-color: #22c55e;
}
.sci-fi-status-container.error {
  --status-color: #ef4444;
}
.sci-fi-status-container.info {
  --status-color: #3b82f6;
}
.sci-fi-status-container.default {
  --status-color: #9ca3af;
}

.sci-fi-status-container {
  /* border-color: var(--status-color); */
  background-color: transparent;
  /* box-shadow: 0 0 0 1px var(--status-bg); */
}

.status-label-mini {
  font-size: 9px;
  color: var(--status-color);
  opacity: 0.8;
  letter-spacing: 1px;
  margin-bottom: 4px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  text-transform: uppercase;
  font-weight: 600;
}

.status-content {
  display: flex;
  align-items: center;
  gap: 8px;
  z-index: 1;
}

.status-text {
  font-size: 16px;
  font-weight: 700;
  color: var(--status-color);
  letter-spacing: 0.5px;
  line-height: 1.2;
}

.status-indicator {
  position: relative;
  width: 8px;
  height: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.status-dot {
  width: 4px;
  height: 4px;
  background-color: var(--status-color);
  border-radius: 50%;
  box-shadow: 0 0 8px var(--status-color); /* Add glow */
  animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite; /* Add pulse */
}

.status-ping {
  position: absolute;
  width: 100%;
  height: 100%;
  border-radius: 50%;
  background-color: var(--status-color); /* Fill instead of border for better visibility */
  opacity: 0;
  animation: ping 1.5s cubic-bezier(0, 0, 0.2, 1) infinite;
}

@keyframes ping {
  75%,
  100% {
    transform: scale(3);
    opacity: 0;
  }
  0% {
    transform: scale(1);
    opacity: 0.6;
  }
}

@keyframes pulse {
  0%,
  100% {
    opacity: 1;
    transform: scale(1);
  }
  50% {
    opacity: 0.7;
    transform: scale(1.2);
  }
}

/* Corner accents removed */
.corner-accents {
  display: none;
}

/* Scan line effect removed for cleaner look */
.scan-line {
  display: none;
}

.main-content-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0; /* 防止内容撑开容器 */
}

.progress-info {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.progress-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.progress-label {
  font-weight: 500;
  color: var(--text-color-2);
  white-space: nowrap;
}

.ghost-control-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  gap: 12px;
  flex-wrap: wrap;
}

@media (max-width: 768px) {
  .ghost-control-header {
    flex-direction: column;
    align-items: flex-start;
  }
}

.ghost-progress-container {
  width: 100%;
  padding: 4px 0;
}

.task-stats {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.stat-item {
  padding: 4px 8px;
  background: var(--color-fill-2);
  border-radius: 4px;
  font-size: 12px;
  color: var(--text-color-1);
}

.dark .stat-item {
  background: rgba(255, 255, 255, 0.1);
  color: rgba(255, 255, 255, 0.85);
}

/* 标签页样式 */
.tab-content {
  padding: 4px 0 16px 0;
}

.sql-actions {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--border-color);
}

.result-summary {
  background: var(--color-fill-1);
  padding: 16px;
  border-radius: 8px;
  margin-bottom: 16px;
}

.result-table {
  border-radius: 8px;
  overflow: hidden;
}

/* 移动端适配 */
@media (max-width: 640px) {
  .order-detail-page {
    padding: 8px;
    gap: 12px !important;
  }

  .order-header-container {
    padding: 12px;
  }

  .info-item {
    padding: 8px 0;
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
  }

  .info-item .label {
    font-size: 12px;
    color: #999;
  }

  .info-item .value {
    font-size: 13px;
  }

  .status-display-section {
    min-width: 100% !important;
    padding: 12px 8px !important;
  }

  .sci-fi-status-container {
    min-width: 100%;
    padding: 12px;
  }

  .status-text {
    font-size: 14px;
  }

  .middle-content-container {
    grid-template-columns: 1fr !important;
    gap: 12px !important;
  }

  .progress-info {
    gap: 8px;
  }

  .progress-item {
    gap: 6px;
  }

  .task-stats {
    flex-direction: column;
    gap: 8px;
  }

  .tab-content {
    padding: 8px 0;
  }

  .result-summary {
    padding: 12px;
  }

  .result-summary .n-statistic {
    min-width: 80px;
  }

  /* 表格优化 */
  .n-data-table {
    font-size: 12px;
  }

  /* 按钮组优化 */
  .n-button {
    font-size: 12px;
  }

  /* 步骤条优化 */
  .n-steps {
    font-size: 12px;
  }

  .n-step {
    padding: 8px 0;
  }

  /* 标签页优化 */
  .n-tabs {
    font-size: 13px;
  }

  /* 卡片优化 */
  .n-card {
    padding: 12px;
  }

  /* 操作日志优化 */
  .n-timeline {
    font-size: 12px;
  }
}
</style>
