<script setup lang="ts">
import { computed, ref, watch, onMounted } from 'vue';
import {
  NModal,
  NForm,
  NFormItem,
  NInput,
  NButton,
  NSpace,
  NSelect,
  NInputNumber
} from 'naive-ui';
import { fetchCreateRolePermission } from '@/service/api/das';
import { fetchGetPermissionTemplates } from '@/service/api/das';
import { fetchGetDBConfigs } from '@/service/api/admin';
import { fetchOrdersSchemas } from '@/service/api/orders';
import { useFormRules, useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';

defineOptions({
  name: 'RolePermissionOperateModal'
});

interface Props {
  role: string;
}

const props = defineProps<Props>();

interface Emits {
  (e: 'submitted'): void;
}

const emit = defineEmits<Emits>();

const visible = defineModel<boolean>('visible', {
  default: false
});

const { formRef, validate, restoreValidation } = useNaiveForm();
const { defaultRequiredRule } = useFormRules();

const title = computed(() => {
  return '新增角色权限';
});

type Model = Api.Das.RolePermissionCreateRequest;

const model = ref<Model>(createDefaultModel());

function createDefaultModel(): Model {
  return {
    role: '',
    permission_type: 'object',
    permission_id: 0,
    instance_id: '',
    schema: '',
    table: ''
  };
}

type RuleKey = Extract<keyof Model, 'role' | 'permission_type' | 'permission_id'>;

const rules: Record<RuleKey, App.Global.FormRule> = {
  role: defaultRequiredRule,
  permission_type: defaultRequiredRule,
  permission_id: defaultRequiredRule
};

const permissionTypeOptions = [
  { label: '直接权限', value: 'object' },
  { label: '权限模板', value: 'template' }
];

// 权限模板选项
const templateOptions = ref<{ label: string; value: number }[]>([]);

// 直接权限相关
const dbConfigOptions = ref<{ label: string; value: string }[]>([]);
const schemaOptions = ref<{ label: string; value: string }[]>([]);

async function loadTemplates() {
  try {
    const res = await fetchGetPermissionTemplates();
    const responseData = (res as any)?.data || res;
    const templates = Array.isArray(responseData) ? responseData : [];
    templateOptions.value = templates.map((t: any) => ({
      label: `${t.name} (${t.permissions?.length || 0}项)`,
      value: t.id
    }));
  } catch (error) {
    console.error('Failed to load templates:', error);
  }
}


async function loadDBConfigs() {
  try {
    const res = await fetchGetDBConfigs({ useType: '查询' } as any);
    const responseData = (res as any)?.data || res;
    const configs = Array.isArray(responseData) ? responseData : [];
    dbConfigOptions.value = configs.map((config: any) => ({
      label: `${config.hostname}:${config.port}${config.instance_id ? ` (${config.instance_id})` : ''}`,
      value: config.instance_id
    }));
  } catch (error) {
    console.error('Failed to load DB configs:', error);
  }
}

async function loadSchemas(instanceId: string) {
  if (!instanceId) {
    schemaOptions.value = [];
    return;
  }
  try {
    const res = await fetchOrdersSchemas({ instance_id: instanceId });
    const responseData = (res as any)?.data || res;
    const schemas = Array.isArray(responseData) ? responseData : [];
    schemaOptions.value = schemas.map((schema: any) => ({
      label: schema.schema || schema.name,
      value: schema.schema || schema.name
    }));
  } catch (error) {
    console.error('Failed to load schemas:', error);
    schemaOptions.value = [];
  }
}

function handleInitModel() {
  model.value = createDefaultModel();
  model.value.role = props.role;
}

function closeModal() {
  visible.value = false;
}

async function handleSubmit() {
  await validate();

  try {
    await fetchCreateRolePermission(model.value);
    window.$message?.success($t('common.addSuccess'));
    closeModal();
    emit('submitted');
  } catch (error) {
    window.$message?.error($t('common.operationFailed') || '操作失败');
  }
}

watch(visible, () => {
  if (visible.value) {
    handleInitModel();
    restoreValidation();
    loadTemplates();
    loadDBConfigs();
  }
});

watch(
  () => model.value.permission_type,
  newVal => {
    // 切换权限类型时，清空相关字段
    model.value.permission_id = 0;
    model.value.instance_id = '';
    model.value.schema = '';
    model.value.table = '';
    schemaOptions.value = [];
  }
);

watch(
  () => model.value.instance_id,
  newVal => {
    if (newVal) {
      loadSchemas(newVal);
    } else {
      schemaOptions.value = [];
      model.value.schema = '';
    }
  }
);

onMounted(() => {
  loadTemplates();
  loadDBConfigs();
});
</script>

<template>
  <NModal
    v-model:show="visible"
    :title="title"
    preset="card"
    :style="{ width: '700px' }"
    :mask-closable="false"
  >
    <NForm ref="formRef" :model="model" :rules="rules" label-placement="left" :label-width="120">
      <NFormItem label="角色" path="role">
        <NInput v-model:value="model.role" :disabled="true" clearable />
      </NFormItem>
      <NFormItem label="权限类型" path="permission_type">
        <NSelect
          v-model:value="model.permission_type"
          :options="permissionTypeOptions"
          placeholder="请选择权限类型"
        />
      </NFormItem>
      <NFormItem
        v-if="model.permission_type === 'template'"
        label="权限模板"
        path="permission_id"
      >
        <NSelect
          v-model:value="model.permission_id"
          :options="templateOptions"
          placeholder="请选择权限模板"
        />
      </NFormItem>
      <template v-if="model.permission_type === 'object'">
        <NFormItem label="实例ID" path="instance_id">
          <NSelect
            v-model:value="model.instance_id"
            :options="dbConfigOptions"
            placeholder="请选择实例"
            clearable
          />
        </NFormItem>
        <NFormItem label="库名" path="schema">
          <NSelect
            v-model:value="model.schema"
            :options="schemaOptions"
            placeholder="请选择库名"
            :disabled="!model.instance_id"
            clearable
          />
        </NFormItem>
        <NFormItem label="表名" path="table">
          <NInput
            v-model:value="model.table"
            placeholder="表名（可选，留空表示整个库）"
            clearable
          />
        </NFormItem>
      </template>
    </NForm>
    <template #footer>
      <NSpace :size="16">
        <NButton @click="closeModal">{{ $t('common.cancel') }}</NButton>
        <NButton type="primary" @click="handleSubmit">{{ $t('common.confirm') }}</NButton>
      </NSpace>
    </template>
  </NModal>
</template>

<style scoped></style>

