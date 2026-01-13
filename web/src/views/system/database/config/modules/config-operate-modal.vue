<script setup lang="ts">
import { computed, ref, watch, onMounted } from 'vue';
import {
  NModal,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NSelect,
  NButton,
  NSpace,
  NGrid,
  NFormItemGi
} from 'naive-ui';
import { jsonClone } from '@sa/utils';
import { fetchCreateDBConfig, fetchUpdateDBConfig } from '@/service/api/admin';
import { fetchGetEnvironments } from '@/service/api/admin';
import { useFormRules, useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';

defineOptions({
  name: 'ConfigOperateModal'
});

interface Props {
  operateType: NaiveUI.TableOperateType;
  rowData?: Api.Admin.DBConfig | null;
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
  const titles: Record<NaiveUI.TableOperateType, string> = {
    add: $t('page.manage.database.config.addConfig'),
    edit: $t('page.manage.database.config.editConfig')
  };
  return titles[props.operateType];
});

type Model = Api.Admin.DBConfigCreateRequest;

const model = ref<Model>(createDefaultModel());

function createDefaultModel(): Model {
  return {
    hostname: '',
    port: 3306,
    user_name: '',
    password: '',
    use_type: '工单',
    db_type: 'MySQL',
    environment: undefined,
    organization_key: '',
    remark: ''
  };
}

type RuleKey = Extract<keyof Model, 'hostname' | 'port' | 'user_name' | 'password' | 'use_type' | 'db_type'>;

const rules: Record<RuleKey, App.Global.FormRule> = {
  hostname: defaultRequiredRule,
  port: defaultRequiredRule,
  user_name: defaultRequiredRule,
  password: defaultRequiredRule,
  use_type: defaultRequiredRule,
  db_type: defaultRequiredRule
};

const useTypeOptions = [
  { label: '查询', value: '查询' },
  { label: '工单', value: '工单' }
];

const dbTypeOptions = [
  { label: 'MySQL', value: 'MySQL' },
  { label: 'TiDB', value: 'TiDB' },
  { label: 'ClickHouse', value: 'ClickHouse' }
];

const environmentOptions = ref<{ label: string; value: number }[]>([]);

async function loadEnvironments() {
  try {
    const res = await fetchGetEnvironments();
    const responseData = (res as any)?.data || res;
    const environments: Api.Admin.Environment[] = Array.isArray(responseData) ? responseData : [];
    environmentOptions.value = environments.map(env => ({
      label: env.name,
      value: env.id
    }));
  } catch (error) {
    console.error('Failed to load environments:', error);
  }
}

function handleInitModel() {
  model.value = createDefaultModel();

  if (props.operateType === 'edit' && props.rowData) {
    Object.assign(model.value, jsonClone(props.rowData));
    // 密码字段在编辑时应该保持为空，如果后端返回的是隐藏后的密码（******），则不设置
    if (props.rowData.password === '******') {
      model.value.password = '';
    }
  }
}

function closeModal() {
  visible.value = false;
}

async function handleSubmit() {
  await validate();

  try {
    if (props.operateType === 'edit' && props.rowData) {
      // 获取 ID，兼容后端返回的 ID（大写）和 id（小写）
      const configId = (props.rowData as any).ID || props.rowData.id;
      if (!configId) {
        window.$message?.error('无法获取配置ID');
        return;
      }
      await fetchUpdateDBConfig(configId, model.value);
      window.$message?.success($t('common.updateSuccess'));
    } else {
      await fetchCreateDBConfig(model.value);
      window.$message?.success($t('common.addSuccess'));
    }
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
    loadEnvironments();
  }
});

onMounted(() => {
  loadEnvironments();
});
</script>

<template>
  <NModal v-model:show="visible" :title="title" preset="card" :style="{ width: '800px' }" :mask-closable="false">
    <NForm ref="formRef" :model="model" :rules="rules" label-placement="left" :label-width="120">
      <NGrid responsive="screen" item-responsive>
        <NFormItemGi span="24 m:12" :label="$t('page.manage.database.config.hostname')" path="hostname">
          <NInput
            v-model:value="model.hostname"
            :placeholder="$t('page.manage.database.config.form.hostname')"
            clearable
          />
        </NFormItemGi>
        <NFormItemGi span="24 m:12" :label="$t('page.manage.database.config.port')" path="port">
          <NInputNumber
            v-model:value="model.port"
            :placeholder="$t('page.manage.database.config.form.port')"
            :min="1"
            :max="65535"
            class="w-full"
            clearable
          />
        </NFormItemGi>
        <NFormItemGi span="24 m:12" :label="$t('page.manage.database.config.userName')" path="user_name">
          <NInput
            v-model:value="model.user_name"
            :placeholder="$t('page.manage.database.config.form.userName')"
            clearable
          />
        </NFormItemGi>
        <NFormItemGi span="24 m:12" :label="$t('page.manage.database.config.password')" path="password">
          <NInput
            v-model:value="model.password"
            type="password"
            show-password-on="click"
            :placeholder="$t('page.manage.database.config.form.password')"
            clearable
          />
        </NFormItemGi>
        <NFormItemGi span="24 m:12" :label="$t('page.manage.database.config.dbType')" path="db_type">
          <NSelect
            v-model:value="model.db_type"
            :options="dbTypeOptions"
            :placeholder="$t('page.manage.database.config.form.dbType')"
          />
        </NFormItemGi>
        <NFormItemGi span="24 m:12" :label="$t('page.manage.database.config.useType')" path="use_type">
          <NSelect
            v-model:value="model.use_type"
            :options="useTypeOptions"
            :placeholder="$t('page.manage.database.config.form.useType')"
          />
        </NFormItemGi>
        <NFormItemGi span="24 m:12" :label="$t('page.manage.database.config.environment')" path="environment">
          <NSelect
            v-model:value="model.environment"
            :options="environmentOptions"
            :placeholder="$t('page.manage.database.config.form.environment')"
            clearable
          />
        </NFormItemGi>
        <NFormItemGi span="24 m:12" :label="$t('page.manage.database.config.organizationKey')" path="organization_key">
          <NInput
            v-model:value="model.organization_key"
            :placeholder="$t('page.manage.database.config.form.organizationKey')"
            clearable
          />
        </NFormItemGi>
        <NFormItemGi span="24" :label="$t('page.manage.database.config.remark')" path="remark">
          <NInput
            v-model:value="model.remark"
            type="textarea"
            :placeholder="$t('page.manage.database.config.form.remark')"
            :rows="3"
            clearable
          />
        </NFormItemGi>
      </NGrid>
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

