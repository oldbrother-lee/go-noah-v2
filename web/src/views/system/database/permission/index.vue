<script setup lang="tsx">
import { ref, onMounted } from 'vue';
import { NButton, NPopconfirm, NCard, NDataTable, NTag, NSpace, NSelect, NInput } from 'naive-ui';
import { fetchGetUserPermissions, fetchRevokeSchemaPermission } from '@/service/api/das';
import { fetchGetAdminUsers } from '@/service/api/admin';
import { useAppStore } from '@/store/modules/app';
import { useTable } from '@/hooks/common/table';
import { $t } from '@/locales';
import PermissionOperateModal from './modules/permission-operate-modal.vue';
import TableHeaderOperation from '@/components/advanced/table-header-operation.vue';

defineOptions({
  name: 'SystemDatabasePermission'
});

const appStore = useAppStore();

const searchUsername = ref<string>('');
const selectedUsername = ref<string>('');

const { columns, columnChecks, data, loading, pagination, getData, getDataByPage } = useTable({
  apiFn: async () => {
    if (!selectedUsername.value) {
      return { data: [], pageNum: 1, pageSize: 10, total: 0 };
    }
    const res = await fetchGetUserPermissions(selectedUsername.value);
    const responseData = (res as any)?.data || res;
    const schemaPerms = responseData?.schema_permissions || [];
    
    return {
      data: schemaPerms.map((item: any, index: number) => ({
        ...item,
        index: index + 1
      })),
      pageNum: 1,
      pageSize: schemaPerms.length,
      total: schemaPerms.length
    };
  },
  columns: () => [
    { type: 'selection', align: 'center', width: 48 },
    { key: 'index', title: $t('common.index'), align: 'center', width: 80, render: (_: any, index: number) => index + 1 },
    { key: 'username', title: $t('page.manage.database.permission.username'), align: 'center', minWidth: 120 },
    { key: 'instance_id', title: $t('page.manage.database.permission.instanceId'), align: 'center', minWidth: 200 },
    { key: 'schema', title: $t('page.manage.database.permission.schema'), align: 'center', minWidth: 150 },
    { key: 'created_at', title: $t('page.manage.database.permission.createdAt'), align: 'center', width: 180 },
    { key: 'updated_at', title: $t('page.manage.database.permission.updatedAt'), align: 'center', width: 180 },
    {
      key: 'operate',
      title: $t('common.operate'),
      align: 'center',
      width: 130,
      render: (row: Api.Das.SchemaPermission) => (
        <div class="flex-center gap-8px">
          <NPopconfirm onPositiveClick={() => handleDelete(row.id)}>
            {{
              default: () => $t('common.confirmDelete'),
              trigger: () => (
                <NButton type="error" ghost size="small">
                  {$t('common.delete')}
                </NButton>
              )
            }}
          </NPopconfirm>
        </div>
      )
    }
  ],
  pagination: { pageSize: 10, pageSizes: [10, 20, 50, 100], showQuickJumper: true }
});

const checkedRowKeys = ref<(string | number)[]>([]);
const modalVisible = ref(false);

const userOptions = ref<{ label: string; value: string }[]>([]);

async function getUsers() {
  try {
    const res = await fetchGetAdminUsers({ page: 1, pageSize: 1000 });
    const responseData = (res as any)?.data || res;
    const users = responseData?.list || [];
    userOptions.value = users.map((user: any) => ({
      label: `${user.username}${user.nickname ? ` (${user.nickname})` : ''}`,
      value: user.username
    }));
  } catch (error) {
    console.error('Failed to load users:', error);
  }
}

function handleAdd() {
  if (!selectedUsername.value) {
    window.$message?.warning($t('page.manage.database.permission.selectUserFirst') || '请先选择用户');
    return;
  }
  modalVisible.value = true;
}

async function handleDelete(id: number) {
  try {
    await fetchRevokeSchemaPermission(id);
    window.$message?.success($t('common.deleteSuccess'));
    await getData();
  } catch (error) {
    window.$message?.error($t('common.deleteFailed') || '删除失败');
  }
}

async function handleBatchDelete() {
  console.log(checkedRowKeys.value);
  window.$message?.info('批量删除功能待实现');
}

function handleSubmitted() {
  modalVisible.value = false;
  getData();
}

function handleSearch() {
  if (!searchUsername.value) {
    window.$message?.warning($t('page.manage.database.permission.pleaseSelectUser') || '请选择用户');
    return;
  }
  selectedUsername.value = searchUsername.value;
  getData();
}

function handleReset() {
  searchUsername.value = '';
  selectedUsername.value = '';
  getData();
}

onMounted(() => {
  getUsers();
});
</script>

<template>
  <div class="min-h-500px flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
    <NCard
      :title="$t('page.manage.database.permission.title')"
      :bordered="false"
      size="small"
      class="card-wrapper sm:flex-1-hidden"
    >
      <template #header-extra>
        <NSpace>
          <NSelect
            v-model:value="searchUsername"
            :options="userOptions"
            filterable
            clearable
            :placeholder="$t('page.manage.database.permission.selectUser')"
            class="w-200px"
          />
          <NButton type="primary" @click="handleSearch">
            {{ $t('common.search') }}
          </NButton>
          <NButton @click="handleReset">
            {{ $t('common.reset') }}
          </NButton>
          <TableHeaderOperation
            v-model:columns="columnChecks"
            :disabled-delete="checkedRowKeys.length === 0"
            :loading="loading"
            @add="handleAdd"
            @delete="handleBatchDelete"
            @refresh="getData"
          />
        </NSpace>
      </template>
      <NDataTable
        v-model:checked-row-keys="checkedRowKeys"
        :columns="columns"
        :data="data"
        size="small"
        :flex-height="!appStore.isMobile"
        :scroll-x="1200"
        :loading="loading"
        remote
        :row-key="row => row.id"
        :pagination="pagination"
        class="sm:h-full"
      />
      <PermissionOperateModal
        v-model:visible="modalVisible"
        :username="selectedUsername"
        @submitted="handleSubmitted"
      />
    </NCard>
  </div>
</template>

<style scoped></style>

