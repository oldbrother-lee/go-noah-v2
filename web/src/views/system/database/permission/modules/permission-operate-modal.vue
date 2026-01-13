<script setup lang="ts">
import { computed, ref, watch, onMounted } from 'vue';
import { NModal, NForm, NFormItem, NInput, NButton, NSpace, NSelect } from 'naive-ui';
import { jsonClone } from '@sa/utils';
import { fetchGrantSchemaPermission } from '@/service/api/das';
import { fetchGetDBConfigs } from '@/service/api/admin';
import { fetchOrdersSchemas } from '@/service/api/orders';
import { useFormRules, useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';

defineOptions({
  name: 'PermissionOperateModal'
});

interface Props {
  username: string;
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
  return $t('page.manage.database.permission.addPermission');
});

type Model = Api.Das.GrantSchemaPermissionRequest;

const model = ref<Model>(createDefaultModel());

function createDefaultModel(): Model {
  return {
    username: '',
    instance_id: '',
    schema: ''
  };
}

type RuleKey = Extract<keyof Model, 'username' | 'instance_id' | 'schema'>;

const rules: Record<RuleKey, App.Global.FormRule> = {
  username: defaultRequiredRule,
  instance_id: defaultRequiredRule,
  schema: defaultRequiredRule
};

const dbConfigOptions = ref<{ label: string; value: string }[]>([]);
const schemaOptions = ref<{ label: string; value: string }[]>([]);

async function getDBConfigs() {
  try {
    const res = await fetchGetDBConfigs({ useType: '查询' });
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

async function getSchemas(instanceId: string) {
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
  model.value.username = props.username;
}

function closeModal() {
  visible.value = false;
}

async function handleSubmit() {
  await validate();

  try {
    await fetchGrantSchemaPermission(model.value);
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
    getDBConfigs();
  }
});

watch(
  () => model.value.instance_id,
  (newVal) => {
    if (newVal) {
      getSchemas(newVal);
    } else {
      schemaOptions.value = [];
    }
  }
);

onMounted(() => {
  getDBConfigs();
});
</script>

<template>
  <NModal v-model:show="visible" :title="title" preset="card" :style="{ width: '600px' }" :mask-closable="false">
    <NForm ref="formRef" :model="model" :rules="rules" label-placement="left" :label-width="120">
      <NFormItem :label="$t('page.manage.database.permission.username')" path="username">
        <NInput v-model:value="model.username" :disabled="true" clearable />
      </NFormItem>
      <NFormItem :label="$t('page.manage.database.permission.instanceId')" path="instance_id">
        <NSelect
          v-model:value="model.instance_id"
          :options="dbConfigOptions"
          :placeholder="$t('page.manage.database.permission.form.instanceId')"
          clearable
        />
      </NFormItem>
      <NFormItem :label="$t('page.manage.database.permission.schema')" path="schema">
        <NSelect
          v-model:value="model.schema"
          :options="schemaOptions"
          :placeholder="$t('page.manage.database.permission.form.schema')"
          :disabled="!model.instance_id"
          clearable
        />
      </NFormItem>
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

