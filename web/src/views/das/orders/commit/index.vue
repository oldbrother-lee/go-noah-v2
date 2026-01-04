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

const route = useRoute();
const router = useRouter();
const message = useMessage();

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
  const res = await fetchOrdersUsers({ is_page: false } as any);
  users.value = (res as any)?.data ?? [];
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
  // 观察左侧卡片内容高度变化，右侧编辑器按此高度限制
  if (leftContentRef.value) {
    leftResizeObserver = new ResizeObserver(entries => {
      const entry = entries[0];
      if (entry) {
        leftContentHeight.value = Math.round(entry.contentRect.height);
      }
    });
    leftResizeObserver.observe(leftContentRef.value);
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
        message.success('语法检查通过，您可以提交工单了，O(∩_∩)O');
      }
      // 失败不弹窗，直接展示表格
    } else {
      // 如果 resp 为 null/undefined，说明 API 调用成功但没有返回数据
      // 对于导出工单，这可能是正常的，视为通过
      syntaxStatus.value = 0;
      message.success('语法检查通过，您可以提交工单了，O(∩_∩)O');
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
    console.log('API响应内容:', res);
    // 当API返回成功时，res可能为null（因为data字段为null）
    // 此时我们认为请求成功，因为没有抛出异常
    if (res === null || res?.response.data.code === '0000') {
      message.success('工单提交成功');
      router.push('/das/orders-list');
    } else {
      message.warning(res?.message || '工单提交失败');
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
  <div>
    <NCard :title="pageTitle">
      <NGrid :x-gap="16" :y-gap="16" style="align-items: stretch">
        <!-- 左侧表单 -->
        <NGi span="8">
          <NCard style="height: 100%">
            <div ref="leftContentRef">
              <NForm label-placement="left" :label-width="96">
                <NFormItem label="标题">
                  <NInput v-model:value="formModel.title" placeholder="请输入工单标题" />
                </NFormItem>
                <NFormItem label="备注">
                  <NInput
                    v-model:value="formModel.remark"
                    type="textarea"
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
                    :options="environments.map((e: any) => ({ label: e.name, value: e.id }))"
                    filterable
                    clearable
                    placeholder="请选择工单环境"
                    @update:value="onEnvironmentChange"
                  />
                </NFormItem>
                <NFormItem label="实例">
                  <NSelect
                    v-model:value="formModel.instanceId"
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
                    :options="schemas.map((s: any) => ({ label: s.schema, value: s.schema }))"
                    filterable
                    clearable
                    placeholder="请选择数据库"
                  />
                </NFormItem>
                <NFormItem v-if="isExportOrder" label="文件格式">
                  <NSelect
                    v-model:value="formModel.exportFileFormat"
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
                    clearable
                    value-format="yyyy-MM-dd HH:mm:ss"
                    placeholder="请选择计划执行时间(可选)"
                    style="width: 100%"
                  />
                </NFormItem>
                <NFormItem label="审核人">
                  <NSelect
                    v-model:value="formModel.approver"
                    multiple
                    filterable
                    clearable
                    placeholder="请选择工单审核人"
                    :options="
                      users.map((u: any) => ({ label: `${u.username} ${u.nick_name || ''}`, value: u.username }))
                    "
                  />
                </NFormItem>
                <NFormItem label="执行人">
                  <NSelect
                    v-model:value="formModel.executor"
                    multiple
                    filterable
                    clearable
                    placeholder="请选择工单执行人"
                    :options="
                      users.map((u: any) => ({ label: `${u.username} ${u.nick_name || ''}`, value: u.username }))
                    "
                  />
                </NFormItem>
                <NFormItem label="复核人">
                  <NSelect
                    v-model:value="formModel.reviewer"
                    multiple
                    filterable
                    clearable
                    placeholder="请选择工单复核人"
                    :options="
                      users.map((u: any) => ({ label: `${u.username} ${u.nick_name || ''}`, value: u.username }))
                    "
                  />
                </NFormItem>
                <NFormItem label="抄送人">
                  <NSelect
                    v-model:value="formModel.cc"
                    multiple
                    filterable
                    clearable
                    placeholder="请选择工单抄送人"
                    :options="
                      users.map((u: any) => ({ label: `${u.username} ${u.nick_name || ''}`, value: u.username }))
                    "
                  />
                </NFormItem>
                <NFormItem>
                  <NButton type="primary" :loading="loading" @click="submitOrder">提交</NButton>
                </NFormItem>
              </NForm>
            </div>
          </NCard>
        </NGi>
        <!-- 右侧编辑区域 -->
        <NGi span="16">
          <NCard class="editor-card" style="height: 100%">
            <div class="editor-inner" :style="{ height: leftContentHeight + 'px' }">
              <NAlert type="info" title="说明" closable>支持多条SQL语句，每条SQL须以 ; 结尾</NAlert>
              <div style="margin: 8px 0">
                <NSpace>
                  <NButton tertiary type="default" @click="formatSQL">格式化</NButton>
                  <NButton tertiary type="default" :loading="checking" :disabled="checking" @click="syntaxCheck">
                    {{ checking ? '检查中...' : '语法检查' }}
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
    <NCard v-if="syntaxRows.length" title="语法检查结果" style="margin-top: 12px">
      <NDataTable
        :columns="visibleSyntaxColumns"
        :data="syntaxRows"
        :pagination="pagination"
        size="small"
        single-line
        table-layout="fixed"
      />
    </NCard>
  </div>
</template>

<style scoped>
:deep(.n-card .n-card__content) {
  padding: 12px;
}
/* 参考 SQL 查询页的编辑器样式 */
.editor-card :deep(.n-card__content) {
  /* 右侧卡片内容作为外层容器，不再直接拉伸 */
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
</style>
