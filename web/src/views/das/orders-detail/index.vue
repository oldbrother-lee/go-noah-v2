<script setup lang="ts">
import { computed, h, onMounted, onUnmounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useDebounceFn } from '@vueuse/core';
import {
  NButton,
  NDataTable,
  NDatePicker,
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
import ReadonlySqlEditor from '@/components/custom/readonly-sql-editor.vue';
import LogViewer from '@/components/custom/log-viewer.vue';

const route = useRoute();
const router = useRouter();
const message = useMessage();
const dialog = useDialog();
const themeStore = useThemeStore();
const authStore = useAuthStore();

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

    // 处理响应数据
    if (resp !== null && resp !== undefined) {
      const result: any = resp.data ?? resp;
      syntaxRows.value = Array.isArray(result?.data) ? result.data : [];
      
      // 优先使用 result.status，如果没有则检查 API 是否成功返回
      if (typeof result?.status === 'number') {
        syntaxStatus.value = result.status;
      } else if (syntaxRows.value.length === 0) {
        // 对于导出工单等，如果没有返回 status 但也没有错误数据，视为通过
        syntaxStatus.value = 0;
      } else {
        // 有错误数据但没有 status，视为失败
        syntaxStatus.value = 1;
      }
      
      if (syntaxStatus.value === 0) {
        message.success('语法检查通过');
      } else {
        message.warning('语法检查未通过');
      }
    } else {
      // 如果 resp 为 null/undefined，说明 API 调用成功但没有返回数据
      // 对于导出工单，这可能是正常的，视为通过
      syntaxStatus.value = 0;
      message.success('语法检查通过');
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

const initWebSocket = () => {
  if (websocket.value) return;

  const orderId = route.params.id as string;

  let wsUrl = '';
  const isDev = import.meta.env.DEV;
  const serviceBaseUrl = import.meta.env.VITE_SERVICE_BASE_URL;

  if (isDev && serviceBaseUrl) {
    // Replace http with ws, https with wss
    wsUrl = `${serviceBaseUrl.replace(/^http/, 'ws')}/ws/${orderId}`;
  } else {
    const protocol = window.location.protocol === 'https:' ? 'wss://' : 'ws://';
    const host = window.location.host;
    wsUrl = `${protocol}${host}/ws/${orderId}`;
  }

  websocket.value = new WebSocket(wsUrl);

  // Debounce task refresh to avoid flooding the server
  const debouncedRefresh = useDebounceFn(() => {
    getTasks(true);
  }, 1000);

  websocket.value.onopen = () => {
    console.log('WebSocket connected');
  };

  websocket.value.onmessage = event => {
    try {
      const result = JSON.parse(event.data);
      if (result.type === 'processlist') {
        // Format processlist data
        let html = '当前SQL SESSION ID的SHOW PROCESSLIST输出:\n';
        for (const key in result.data) {
          html += `${key}: ${result.data[key]}\n`;
        }
        oscContent.value = html;
      } else if (result.type === 'ghost') {
        // Append ghost logs
        oscContent.value += result.data;
      } else {
        // Append content
        oscContent.value += `${result.data}\n`;
      }
      // Auto refresh tasks on any message
      debouncedRefresh();
    } catch (e) {
      console.error('WebSocket message parse error:', e);
      oscContent.value += `${event.data}\n`;
      // Auto refresh tasks even on parse error as it implies activity
      debouncedRefresh();
    }
  };

  websocket.value.onerror = error => {
    console.error('WebSocket error:', error);
    // Reconnect after 3s
    setTimeout(() => {
      websocket.value = null;
      initWebSocket();
    }, 3000);
  };

  websocket.value.onclose = () => {
    console.log('WebSocket closed');
    websocket.value = null;
  };
};

const closeWebSocket = () => {
  if (websocket.value) {
    websocket.value.close();
    websocket.value = null;
  }
};

onUnmounted(() => {
  closeWebSocket();
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
    const { data } = await fetchOrderDetail(orderId);
    if (data) {
      orderDetail.value = data as unknown as OrderDetailVO;
      // 这里的 taskStats 需要从 preview 或 tasks 接口获取，暂时保持 null
      // taskStats.value = null;
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
  getTasks();
  initWebSocket();
});
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

    const msgType = data?.data?.type;
    const msgContent = data?.message || '已触发全部执行';

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
  try {
    await fetchExecuteSingleTask({ task_id: row.id } as any);
    window.$message?.success('已触发执行');
    // 刷新单行状态太麻烦，直接刷新列表
    handleRefresh();
  } catch (e: any) {
    window.$message?.error(e?.message || '执行失败');
  }
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
          <div class="mb-4 flex items-center justify-between">
            <div class="flex items-center gap-2">
              <NButton text class="mr-2" @click="router.back()">
                <template #icon>
                  <div class="i-ic:round-arrow-back text-xl" />
                </template>
              </NButton>
              <h1 class="m-0 text-2xl font-bold">{{ orderDetail?.title?.split('_')[0] || '工单详情' }}</h1>
              <NTag type="primary" size="small" bordered>#{{ orderDetail?.order_id }}</NTag>
            </div>
            <div class="flex gap-2">
              <NButton type="primary" ghost size="small" :loading="refreshLoading" @click="handleRefresh">
                <template #icon>
                  <div class="i-ic:round-refresh" />
                </template>
                刷新
              </NButton>
              <!-- 操作按钮组 -->
              <NButton
                v-if="actionType !== 'none'"
                type="primary"
                size="small"
                :disabled="actionDisabled"
                @click="showActionModal"
              >
                {{ actionTitle }}
              </NButton>

              <NButton v-if="orderDetail?.progress === '已复核'" type="info" size="small" @click="handleHook">
                <template #icon>
                  <div class="i-ant-design:link-outlined" />
                </template>
                Hook
              </NButton>

              <NButton type="error" ghost size="small" :disabled="closeDisabled" @click="showCloseModal">
                <template #icon>
                  <div class="i-ant-design:close-circle-outlined" />
                </template>
                关闭工单
              </NButton>

              <NButton
                v-if="showGenerateBtn"
                type="warning"
                ghost
                size="small"
                :loading="executeLoading"
                @click="handleGenerateTasks"
              >
                <template #icon>
                  <div class="i-ant-design:thunderbolt-outlined" />
                </template>
                生成任务
              </NButton>

              <NButton
                v-if="showExecuteAllBtn"
                type="success"
                ghost
                size="small"
                :loading="executeLoading"
                :disabled="!!orderDetail?.schedule_time"
                @click="handleExecuteAll"
              >
                <template #icon>
                  <div class="i-ant-design:thunderbolt-filled" />
                </template>
                执行全部
              </NButton>
            </div>
          </div>

          <!-- 工单基本信息区域 -->
          <div class="order-info-section">
            <div class="flex gap-6">
              <!-- 左侧信息列表 -->
              <div class="grid grid-cols-3 flex-1 gap-x-8 gap-y-4 text-sm">
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
                class="status-display-section min-w-[160px] flex flex-col items-center justify-center border-l border-gray-100 px-8 dark:border-gray-800"
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
        <div class="middle-content-container">
          <!-- 左侧：进度信息 -->
          <div class="progress-section">
            <!-- 进度信息 -->
            <NCard title="进度信息" size="small" class="mb-16px">
              <div class="progress-info">
                <div class="progress-item">
                  <span class="progress-label">当前状态：</span>
                  <NTag :type="getStatusType">{{ getStatusLabel }}</NTag>
                </div>
                <div v-if="taskStats" class="progress-item">
                  <span class="progress-label">任务进度：</span>
                  <div class="task-stats">
                    <NSpace size="small">
                      <NTag :bordered="false" type="primary" size="small">任务数: {{ taskStats.total }}</NTag>
                      <NTag :bordered="false" type="success" size="small">成功数: {{ taskStats.completed }}</NTag>
                      <NTag :bordered="false" type="default" size="small">未执行: {{ taskStats.unexecuted }}</NTag>
                      <NTag :bordered="false" type="error" size="small">已失败: {{ taskStats.failed }}</NTag>
                      <NTag :bordered="false" type="info" size="small">执行中: {{ taskStats.processing }}</NTag>
                      <NTag :bordered="false" type="warning" size="small">已暂停: {{ taskStats.paused }}</NTag>
                    </NSpace>
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
                  <NSpace v-if="activeTab === 'sql-content'" align="center" :size="12">
                    <NButton size="small" type="primary" secondary @click="handleFormatSQL">格式化</NButton>
                    <NButton size="small" type="primary" secondary :loading="checking" @click="handleSyntaxCheck">
                      sql审核
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
                      height="500px"
                      @page-change="handleSqlPageChange"
                      @page-size-change="handleSqlPageSizeChange"
                    />
                    <NCard v-if="syntaxRows.length" title="语法检查结果" class="mt-4" size="small">
                      <NDataTable
                        :columns="syntaxColumns"
                        :data="syntaxRows"
                        :pagination="{ pageSize: 10 }"
                        size="small"
                        :scroll-x="1200"
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
                      <NSpace>
                        <NStatistic label="总执行数">
                          <span class="text-16px font-bold">{{ taskStats?.total || 0 }}</span>
                        </NStatistic>
                        <NStatistic label="成功数">
                          <span class="text-16px text-green-600 font-bold">{{ taskStats?.completed || 0 }}</span>
                        </NStatistic>
                        <NStatistic label="失败数">
                          <span class="text-16px text-red-600 font-bold">{{ taskStats?.failed || 0 }}</span>
                        </NStatistic>
                        <NStatistic label="警告数">
                          <span class="text-16px text-orange-600 font-bold">{{ taskStats?.unexecuted || 0 }}</span>
                        </NStatistic>
                      </NSpace>
                    </div>
                    <NDataTable
                      :columns="resultColumns"
                      :data="resultData"
                      :pagination="{ pageSize: 10 }"
                      :bordered="false"
                      size="small"
                      :scroll-x="1000"
                    />
                  </div>
                </NTabPane>

                <!-- 执行进度标签页 -->
                <NTabPane name="osc-progress" tab="执行进度">
                  <div class="tab-content">
                    <LogViewer :content="oscContent" height="500px" :theme="theme" />
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

    <NModal v-model:show="hookVisible" preset="dialog" title="HOOK工单" :style="{ width: '65%' }">
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
              <div class="grid grid-cols-12 gap-12px">
                <div class="col-span-4">
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
                <div class="col-span-4">
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
                <div class="col-span-3">
                  <NFormItem label="库名">
                    <NSelect v-model:value="item.schema" :options="schemasOptions[idx] || []" clearable filterable />
                  </NFormItem>
                </div>
                <div class="col-span-1 flex items-center">
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

@media (max-width: 768px) {

  .result-summary {
    padding: 12px;
  }

  .task-stats {
    flex-direction: column;
    gap: 8px;
  }

  .tab-content {
    padding: 12px 0;
  }
}

@media (max-width: 480px) {
  .order-header-container {
    padding: 12px;
  }

  .info-item {
    padding: 8px 0;
  }

  .progress-info {
    gap: 8px;
  }

  .progress-item {
    gap: 6px;
  }
}
</style>
