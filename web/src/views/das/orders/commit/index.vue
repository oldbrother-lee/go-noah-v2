<script setup lang="ts">
import { computed, h, onMounted, onUnmounted, reactive, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import {
  NAlert,
  NButton,
  NCard,
  NDataTable,
  NDatePicker,
  NForm,
  NFormItem,
  NGi,
  NGrid,
  NInput,
  NSelect,
  NSpace,
  NSwitch,
  NTag,
  NText,
  useMessage
} from 'naive-ui';
import type { PaginationProps } from 'naive-ui';
import { format } from 'sql-formatter';
// CodeMirror 6 imports (align with SQL 查询页)
import { EditorState } from '@codemirror/state';
import { EditorView, keymap, lineNumbers } from '@codemirror/view';
import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands';
import { sql } from '@codemirror/lang-sql';
import { defaultHighlightStyle, foldGutter, foldKeymap, syntaxHighlighting } from '@codemirror/language';
import { autocompletion, completionKeymap } from '@codemirror/autocomplete';
import {
  fetchCreateOrder,
  fetchOrdersEnvironments,
  fetchOrdersInstances,
  fetchOrdersSchemas,
  fetchOrdersUsers,
  fetchSyntaxCheck
} from '@/service/api/orders';
import { useAppStore } from '@/store/modules/app';

const route = useRoute();
const router = useRouter();
const message = useMessage();
const appStore = useAppStore();

// 页面标题与工单类型
const sqlType = ref<string>('DDL');
const pageTitle = computed(() => `提交${sqlType.value}工单`);
const isExportOrder = computed(() => sqlType.value.toLowerCase() === 'export');

// 表单模型
interface FormModel {
  title: string;
  remark?: string;
  isRestrictAccess: boolean;
  dbType: 'MySQL' | 'TiDB';
  environment?: number | null;
  instanceId?: number | null;
  schema?: string | null;
  exportFileFormat?: 'XLSX' | 'CSV';
  approver: string[]; // username list
  executor: string[]; // username list
  reviewer: string[]; // username list
  cc: string[]; // username list
  content: string;
  scheduleTime?: string | null;
}

const formModel = reactive<FormModel>({
  title: '',
  remark: '',
  isRestrictAccess: true,
  dbType: 'MySQL',
  environment: null,
  instanceId: null,
  schema: null,
  exportFileFormat: 'XLSX',
  approver: [],
  executor: [],
  reviewer: [],
  cc: [],
  content: '',
  scheduleTime: null
});

// 下拉数据源
const environments = ref<any[]>([]);
const instances = ref<any[]>([]);
const schemas = ref<any[]>([]);
const users = ref<any[]>([]);

// 加载器状态
const loading = ref(false);
const checking = ref(false);

function inferSqlTypeFromPath() {
  const seg = route.path.split('/').pop()?.toUpperCase();
  if (seg && ['DDL', 'DML', 'EXPORT'].includes(seg)) {
    sqlType.value = seg as any;
  } else {
    sqlType.value = 'DDL';
  }
}

async function loadEnvironments() {
  const res = await fetchOrdersEnvironments({ is_page: false } as any);
  environments.value = (res as any)?.data ?? [];
}

async function loadInstances(envId: number | null) {
  if (!envId) {
    instances.value = [];
    return;
  }
  const res = await fetchOrdersInstances({ id: envId, db_type: formModel.dbType, is_page: false } as any);
  instances.value = (res as any)?.data ?? [];
}

async function loadSchemas(instanceId: number | null) {
  if (!instanceId) {
    schemas.value = [];
    return;
  }
  const res = await fetchOrdersSchemas({ instance_id: instanceId, is_page: false } as any);
  schemas.value = (res as any)?.data ?? [];
}

async function loadUsers() {
  const res = await fetchOrdersUsers();
  // admin/users 返回分页格式 { list: [...], total: ... }
  users.value = (res as any)?.data?.list ?? (res as any)?.data ?? [];
}

// 编辑器设置：右侧改为与 SQL 查询页一致的 CodeMirror
const editorRoot = ref<HTMLElement | null>(null);
const editorView = ref<EditorView | null>(null);

function initEditor() {
  if (editorView.value || !editorRoot.value) return;
  const state = EditorState.create({
    doc: formModel.content || '',
    extensions: [
      lineNumbers(),
      foldGutter(),
      sql({ upperCaseKeywords: true }),
      syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
      history(),
      keymap.of([...defaultKeymap, ...historyKeymap, ...foldKeymap, ...completionKeymap, indentWithTab]),
      autocompletion({ activateOnTyping: true }),
      EditorView.updateListener.of(v => {
        if (v.docChanged) {
          formModel.content = v.state.doc.toString();
        }
      })
    ]
  });
  editorView.value = new EditorView({ state, parent: editorRoot.value });
}

// 外部更新（如格式化）时同步到编辑器
watch(
  () => formModel.content,
  val => {
    syntaxStatus.value = null;
    syntaxRows.value = [];

    const view = editorView.value;
    if (!view) return;
    const cur = view.state.doc.toString();
    if (cur !== (val || '')) {
      view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: val || '' } });
    }
  }
);

const leftContentRef = ref<HTMLElement | null>(null);
const leftContentHeight = ref<number>(0);
let leftResizeObserver: ResizeObserver | null = null;

onMounted(async () => {
  inferSqlTypeFromPath();
  await loadEnvironments();
  await loadUsers();
  initEditor();
  // 观察左侧卡片内容高度变化，右侧编辑器按此高度限制（仅桌面端）
  if (leftContentRef.value && !appStore.isMobile) {
    leftResizeObserver = new ResizeObserver(entries => {
      const entry = entries[0];
      if (entry) {
        leftContentHeight.value = Math.round(entry.contentRect.height);
      }
    });
    leftResizeObserver.observe(leftContentRef.value);
  } else if (appStore.isMobile) {
    // 移动端设置固定高度
    leftContentHeight.value = 400;
  }
});

onUnmounted(() => {
  if (editorView.value) {
    editorView.value.destroy();
    editorView.value = null;
  }
  if (leftResizeObserver) {
    leftResizeObserver.disconnect();
    leftResizeObserver = null;
  }
});

function onDBTypeChange() {
  formModel.environment = null;
  formModel.instanceId = null;
  formModel.schema = null;
  instances.value = [];
  schemas.value = [];
}

async function onEnvironmentChange(val: number) {
  formModel.instanceId = null;
  formModel.schema = null;
  await loadInstances(val ?? null);
}

async function onInstanceChange(val: number) {
  formModel.schema = null;
  await loadSchemas(val ?? null);
}

function formatSQL() {
  try {
    formModel.content = format(formModel.content || '', { language: 'mysql' });
    message.success('格式化完成');
  } catch (e) {
    message.error('格式化失败');
  }
}

const syntaxRows = ref<any[]>([]);
const syntaxStatus = ref<number | null>(null);
const showFingerprint = ref(false);
const visibleSyntaxColumns = computed(() =>
  syntaxColumns.filter((col: any) => col.key !== 'finger_id' || showFingerprint.value)
);
function isPass(row: any) {
  return row?.level === 'INFO' && (!row?.summary || row.summary.length === 0);
}
const pagination = reactive<PaginationProps>({
  page: 1,
  pageSize: 10,
  showSizePicker: true,
  itemCount: 0,
  pageSizes: [10, 20, 50, 100],
  onUpdatePage: (page: number) => {
    pagination.page = page;
  },
  onUpdatePageSize: (size: number) => {
    pagination.pageSize = size;
    pagination.page = 1;
  }
});
watch(syntaxRows, rows => {
  pagination.itemCount = rows.length;
  pagination.page = 1;
});
const syntaxColumns = [
  { title: '错误级别', key: 'level', width: 80 },
  { title: '影响行数', key: 'affected_rows', width: 90 },
  { title: '类型', key: 'type', width: 90 },
  { title: '指纹', key: 'finger_id', width: 120 },
  {
    title: '信息提示',
    key: 'summary',
    width: 300,
    ellipsis: { tooltip: true },
    render: (row: any) => (row.summary && row.summary.length ? row.summary.join('；') : '—')
  },
  { title: 'SQL', key: 'query', width: 500, ellipsis: { tooltip: true } },
  {
    title: '检测结果',
    key: 'result',
    width: 100,
    render: (row: any) =>
      h(NTag, { type: isPass(row) ? 'success' : 'error' }, { default: () => (isPass(row) ? '通过' : '失败') })
  }
];
async function syntaxCheck() {
  if (!formModel.content) {
    message.warning('输入内容不能为空');
    return;
  }
  if (!formModel.environment) {
    message.warning('请选择环境');
    return;
  }
  if (!formModel.instanceId) {
    message.warning('请选择实例');
    return;
  }
  if (!formModel.schema) {
    message.warning('请选择库名');
    return;
  }
  checking.value = true;
  syntaxStatus.value = null;
  syntaxRows.value = [];
  try {
    const data = {
      db_type: formModel.dbType,
      sql_type: sqlType.value,
      instance_id: formModel.instanceId,
      schema: formModel.schema,
      content: formModel.content
    };
    const resp: any = await fetchSyntaxCheck(data as any);
    console.log('语法检查响应:', resp);

    // createFlatRequest 返回 { data, error, response } 格式
    // 错误时: data=null, error=AxiosError
    // 成功时: data=响应数据, error=null
    
    // 检查是否有错误（error 不为 null 或 data 为 null）
    if (resp.error || resp.data === null || resp.data === undefined) {
      // 请求失败
      syntaxStatus.value = 1;
      // 尝试从错误响应中获取详细数据用于展示
      const errorData = resp.error?.response?.data?.data ?? resp.response?.data?.data ?? [];
      syntaxRows.value = Array.isArray(errorData) ? errorData : [];
      // 错误消息已由全局拦截器显示，不再重复
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
      message.success('语法检查通过，您可以提交工单了，O(∩_∩)O');
    } else {
      message.warning('语法检查未通过，请修复问题后重新检查');
    }
  } catch (e: any) {
    console.error('语法检查失败:', e);
    message.error(e?.message || '语法检查失败');
    syntaxStatus.value = null;
  } finally {
    checking.value = false;
  }
}

function validateApprover(list: string[]) {
  if (!list || list.length < 1) {
    message.error('请至少选择1位工单审核人');
    return false;
  }
  if (list.length >= 3) {
    message.error('最多不允许超过3位工单审核人');
    return false;
  }
  return true;
}

async function submitOrder() {
  loading.value = true;
  try {
    if (!formModel.title || formModel.title.length < 5) {
      message.error('请填写标题(不少于5个字符)');
      return;
    }
    if (!formModel.environment || !formModel.instanceId || !formModel.schema) {
      message.error('请完善环境/实例/库名');
      return;
    }
    if (!validateApprover(formModel.approver)) return;
    if (!formModel.content) {
      message.error('提交的SQL内容不能为空');
      return;
    }

    if (syntaxStatus.value !== 0) {
      message.error('语法检测未通过不允许提交工单');
      return;
    }

    const payload = {
      title: formModel.title,
      remark: formModel.remark,
      is_restrict_access: formModel.isRestrictAccess,
      db_type: formModel.dbType,
      environment: formModel.environment,
      instance_id: formModel.instanceId,
      schema: formModel.schema,
      export_file_format: formModel.exportFileFormat,
      approver: formModel.approver,
      executor: formModel.executor,
      reviewer: formModel.reviewer,
      cc: formModel.cc,
      sql_type: sqlType.value,
      content: formModel.content,
      schedule_time: formModel.scheduleTime
    };

    const res: any = await fetchCreateOrder(payload as any);
    
    // createFlatRequest 返回 { data, error, response } 格式
    // 错误时: data=null, error=AxiosError
    // 成功时: data=后端响应的data字段（工单对象）, error=null, response=完整响应
    // 后端成功响应格式: { code: 0, message: "ok", data: {...} }
    // 注意：res.data 是后端响应的 data 字段，不是整个响应对象
    // 响应码在 res.response.data.code 中
    
    // 检查是否有错误
    if (res.error) {
      // 请求失败，错误消息已由全局拦截器显示
      message.warning('工单提交失败');
      return;
    }
    
    // 检查响应码（成功时为 0）
    // 响应码在 res.response.data.code 中，而不是 res.data.code
    const successCode = import.meta.env.VITE_SERVICE_SUCCESS_CODE || '0';
    const responseCode = String(res.response?.data?.code ?? '');
    
    if (responseCode === successCode) {
      message.success('工单提交成功');
      router.push('/das/orders-list');
    } else {
      // 如果响应码不匹配，显示后端返回的消息
      const errorMessage = res.response?.data?.message || '工单提交失败';
      message.warning(errorMessage);
    }
  } catch (e: any) {
    message.error(e?.message || '工单提交失败');
  } finally {
    loading.value = false;
  }
}

watch(
  () => route.path,
  () => inferSqlTypeFromPath()
);
</script>

<template>
  <div class="order-commit-page">
    <NCard :title="pageTitle" :content-style="{ padding: appStore.isMobile ? '8px' : '16px' }">
      <NGrid :x-gap="appStore.isMobile ? 8 : 16" :y-gap="appStore.isMobile ? 8 : 16" responsive="screen" style="align-items: stretch">
        <!-- 左侧表单 -->
        <NGi :span="appStore.isMobile ? 24 : 8">
          <NCard style="height: 100%">
            <div ref="leftContentRef">
              <NForm :label-placement="appStore.isMobile ? 'top' : 'left'" :label-width="appStore.isMobile ? 'auto' : 96">
                <NFormItem label="标题">
                  <NInput v-model:value="formModel.title" :size="appStore.isMobile ? 'small' : 'medium'" placeholder="请输入工单标题" />
                </NFormItem>
                <NFormItem label="备注">
                  <NInput
                    v-model:value="formModel.remark"
                    type="textarea"
                    :size="appStore.isMobile ? 'small' : 'medium'"
                    :autosize="{ minRows: 2, maxRows: 6 }"
                    placeholder="请输入工单需求或备注"
                  />
                </NFormItem>
                <NFormItem label="限制访问">
                  <NSwitch v-model:value="formModel.isRestrictAccess" />
                </NFormItem>
                <NFormItem label="DB类型">
                  <NSelect
                    v-model:value="formModel.dbType"
                    :size="appStore.isMobile ? 'small' : 'medium'"
                    :options="[
                      { label: 'MySQL', value: 'MySQL' },
                      { label: 'TiDB', value: 'TiDB' }
                    ]"
                    @update:value="onDBTypeChange"
                  />
                </NFormItem>
                <NFormItem label="环境">
                  <NSelect
                    v-model:value="formModel.environment"
                    :size="appStore.isMobile ? 'small' : 'medium'"
                    :options="environments.map((e: any) => ({ label: e.name, value: e.ID }))"
                    filterable
                    clearable
                    placeholder="请选择工单环境"
                    @update:value="onEnvironmentChange"
                  />
                </NFormItem>
                <NFormItem label="实例">
                  <NSelect
                    v-model:value="formModel.instanceId"
                    :size="appStore.isMobile ? 'small' : 'medium'"
                    :options="instances.map((i: any) => ({ label: i.remark, value: i.instance_id }))"
                    filterable
                    clearable
                    placeholder="请选择数据库实例"
                    @update:value="onInstanceChange"
                  />
                </NFormItem>
                <NFormItem label="库名">
                  <NSelect
                    v-model:value="formModel.schema"
                    :size="appStore.isMobile ? 'small' : 'medium'"
                    :options="schemas.map((s: any) => ({ label: s.schema, value: s.schema }))"
                    filterable
                    clearable
                    placeholder="请选择数据库"
                  />
                </NFormItem>
                <NFormItem v-if="isExportOrder" label="文件格式">
                  <NSelect
                    v-model:value="formModel.exportFileFormat"
                    :size="appStore.isMobile ? 'small' : 'medium'"
                    :options="[
                      { label: 'XLSX', value: 'XLSX' },
                      { label: 'CSV', value: 'CSV' }
                    ]"
                  />
                </NFormItem>
                <NFormItem label="定时执行">
                  <NDatePicker
                    v-model:formatted-value="formModel.scheduleTime"
                    type="datetime"
                    :size="appStore.isMobile ? 'small' : 'medium'"
                    clearable
                    value-format="yyyy-MM-dd HH:mm:ss"
                    placeholder="请选择计划执行时间(可选)"
                    style="width: 100%"
                  />
                </NFormItem>
                <NFormItem label="审核人">
                  <NSelect
                    v-model:value="formModel.approver"
                    :size="appStore.isMobile ? 'small' : 'medium'"
                    multiple
                    filterable
                    clearable
                    placeholder="请选择工单审核人"
                    :options="
                      users.map((u: any) => ({ label: `${u.username} ${u.nickname || ''}`, value: u.username }))
                    "
                  />
                </NFormItem>
                <NFormItem label="执行人">
                  <NSelect
                    v-model:value="formModel.executor"
                    :size="appStore.isMobile ? 'small' : 'medium'"
                    multiple
                    filterable
                    clearable
                    placeholder="请选择工单执行人"
                    :options="
                      users.map((u: any) => ({ label: `${u.username} ${u.nickname || ''}`, value: u.username }))
                    "
                  />
                </NFormItem>
                <NFormItem label="复核人">
                  <NSelect
                    v-model:value="formModel.reviewer"
                    :size="appStore.isMobile ? 'small' : 'medium'"
                    multiple
                    filterable
                    clearable
                    placeholder="请选择工单复核人"
                    :options="
                      users.map((u: any) => ({ label: `${u.username} ${u.nickname || ''}`, value: u.username }))
                    "
                  />
                </NFormItem>
                <NFormItem label="抄送人">
                  <NSelect
                    v-model:value="formModel.cc"
                    :size="appStore.isMobile ? 'small' : 'medium'"
                    multiple
                    filterable
                    clearable
                    placeholder="请选择工单抄送人"
                    :options="
                      users.map((u: any) => ({ label: `${u.username} ${u.nickname || ''}`, value: u.username }))
                    "
                  />
                </NFormItem>
                <NFormItem>
                  <NButton type="primary" :size="appStore.isMobile ? 'small' : 'medium'" :loading="loading" :block="appStore.isMobile" @click="submitOrder">提交</NButton>
                </NFormItem>
              </NForm>
            </div>
          </NCard>
        </NGi>
        <!-- 右侧编辑区域 -->
        <NGi :span="appStore.isMobile ? 24 : 16">
          <NCard class="editor-card" :style="{ height: appStore.isMobile ? 'auto' : '100%' }">
            <div class="editor-inner" :style="{ height: appStore.isMobile ? '400px' : (leftContentHeight > 0 ? leftContentHeight + 'px' : '500px') }">
              <NAlert type="info" title="说明" :closable="!appStore.isMobile" :size="appStore.isMobile ? 'small' : 'medium'">支持多条SQL语句，每条SQL须以 ; 结尾</NAlert>
              <div style="margin: 8px 0">
                <NSpace :size="appStore.isMobile ? 4 : 8" :wrap="appStore.isMobile">
                  <NButton :size="appStore.isMobile ? 'small' : 'medium'" tertiary type="default" @click="formatSQL">
                    <template v-if="appStore.isMobile" #icon>
                      <div class="i-ant-design:format-painter-outlined" />
                    </template>
                    <span v-if="!appStore.isMobile">格式化</span>
                  </NButton>
                  <NButton :size="appStore.isMobile ? 'small' : 'medium'" tertiary type="default" :loading="checking" :disabled="checking" @click="syntaxCheck">
                    {{ checking ? '检查中...' : (appStore.isMobile ? '检查' : '语法检查') }}
                  </NButton>
                </NSpace>
              </div>
              <!-- 替换 textarea 为与 SQL 查询一致的 CodeMirror 编辑器 -->
              <div ref="editorRoot" class="code-editor-container" />
            </div>
          </NCard>
        </NGi>
      </NGrid>
    </NCard>
    <NCard v-if="syntaxRows.length" title="语法检查结果" :style="{ marginTop: appStore.isMobile ? '8px' : '12px' }" :content-style="{ padding: appStore.isMobile ? '8px' : '16px' }">
      <NDataTable
        :columns="visibleSyntaxColumns"
        :data="syntaxRows"
        :pagination="pagination"
        size="small"
        single-line
        table-layout="fixed"
        :scroll-x="appStore.isMobile ? 800 : 1200"
      />
    </NCard>
  </div>
</template>

<style scoped>
.order-commit-page {
  padding: 0;
}

:deep(.n-card .n-card__content) {
  padding: 12px;
}

/* 参考 SQL 查询页的编辑器样式 */
.editor-card :deep(.n-card__content) {
  /* 右侧卡片内容作为外层容器，不再直接拉伸 */
  display: flex;
  flex-direction: column;
}
.editor-inner {
  display: flex;
  flex-direction: column;
  overflow: hidden; /* 限制整体高度，内部滚动 */
}
.code-editor-container {
  border: 1px solid var(--n-border-color);
  border-radius: 8px;
  background-color: var(--n-color);
  margin-bottom: 8px;
  display: flex;
  flex-direction: column;
  flex: 1 1 auto;
  min-height: 0; /* 允许内部滚动 */
}
.code-editor-container :deep(.cm-editor) {
  background-color: transparent;
  font-family: 'JetBrains Mono', 'Fira Code', Menlo, Monaco, 'Courier New', monospace;
  font-size: 13px;
  height: 100%; /* 填满容器以启用滚动 */
}
.code-editor-container :deep(.cm-scroller) {
  height: 100%;
  padding: 4px;
  overflow: auto; /* 内容超出时滚动 */
}
.code-editor-container :deep(.cm-gutters) {
  background-color: var(--n-color);
  border-right: 1px solid var(--n-border-color);
}
.code-editor-container :deep(.cm-activeLine) {
  background-color: rgba(0, 0, 0, 0.03);
}

/* 移动端适配 */
@media (max-width: 640px) {
  .order-commit-page {
    padding: 0;
  }

  :deep(.n-card .n-card__content) {
    padding: 8px;
  }

  .editor-inner {
    min-height: 300px;
  }

  .code-editor-container {
    min-height: 300px;
    font-size: 12px;
  }

  .code-editor-container :deep(.cm-editor) {
    font-size: 12px !important;
  }

  /* 表单优化 */
  :deep(.n-form-item) {
    margin-bottom: 16px;
  }

  :deep(.n-form-item-label) {
    font-size: 13px;
    margin-bottom: 4px;
  }

  /* 按钮优化 */
  :deep(.n-button) {
    font-size: 13px;
  }

  /* 选择框优化 */
  :deep(.n-select) {
    font-size: 13px;
  }

  /* 输入框优化 */
  :deep(.n-input) {
    font-size: 13px;
  }

  /* 表格优化 */
  :deep(.n-data-table) {
    font-size: 12px;
  }

  /* 卡片标题优化 */
  :deep(.n-card-header) {
    padding: 12px;
    font-size: 16px;
  }

  /* 警告框优化 */
  :deep(.n-alert) {
    font-size: 12px;
    padding: 8px;
  }
}
</style>
