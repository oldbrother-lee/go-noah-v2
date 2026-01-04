<script setup lang="tsx">
import { onMounted, onUnmounted, reactive, ref } from 'vue';
import { NCard, NDataTable, NSwitch, NTag } from 'naive-ui';
import { fetchOrdersList } from '@/service/api/orders';
import { useAppStore } from '@/store/modules/app';
import { useTable } from '@/hooks/common/table';
import { useRouterPush } from '@/hooks/common/router';
import { $t } from '@/locales';
import OrderSearch from './modules/order-search.vue';

const appStore = useAppStore();
const { routerPushByKey } = useRouterPush();

/**
 * Order search parameters
 * 工单搜索参数
 */
const searchParams = reactive<Api.Orders.OrderSearchParams>({
  current: 1,
  size: 10,
  environment: null,
  status: null,
  search: null,
  only_my_orders: 0
});

const onlyMyOrders = ref(false);

/**
 * Handle "Only My Orders" switch change
 * 处理“只看我的”开关变化
 * @param val boolean value
 */
function handleMyOrdersChange(val: boolean) {
  searchParams.only_my_orders = val ? 1 : 0;
  getDataByPage();
}

/**
 * Get progress tag color
 * 获取进度标签颜色
 * @param progress Progress status string
 * @returns NaiveUI theme color
 */
function getProgressTagColor(progress: string): NaiveUI.ThemeColor {
  switch (progress) {
    case '待审批':
    case '待执行':
      return 'warning';
    case '已驳回':
    case '已失败':
      return 'error';
    case '执行中':
      return 'info';
    case '已完成':
      return 'success';
    default:
      return 'default';
  }
}

/**
 * Handle row click
 * 处理行点击
 * @param row Order data
 */
function handleRowClick(row: Api.Orders.Order) {
  routerPushByKey('das_orders-detail', { params: { id: row.order_id } });
}

const rowProps = (row: Api.Orders.Order) => {
  return {
    style: 'cursor: pointer;',
    onClick: () => handleRowClick(row)
  };
};

/**
 * Reset search parameters
 * 重置搜索参数
 */
function handleReset() {
  onlyMyOrders.value = false;
  searchParams.only_my_orders = 0;
  getDataByPage();
}

/**
 * Table configuration
 * 表格配置
 */
const { columns, data, getData, getDataByPage, loading, mobilePagination } = useTable({
  apiFn: fetchOrdersList,
  apiParams: searchParams,
  showTotal: true,
  pagination: {
    pageSize: 10,
    pageSizes: [10, 20, 50, 100],
    showQuickJumper: true
  },
  transformer: res => {
    // response from requestRaw contains { code, message, data, total }
    // res.data is the records array
    // res.total is the total count
    const responseData = res as any;
    const records = responseData.data || [];
    const total = responseData.total || 0;
    const current = searchParams.current;
    const size = searchParams.size;

    const recordsWithIndex = records.map((item: any, index: number) => {
      return {
        ...item,
        index: (current - 1) * size + index + 1
      };
    });

    return {
      data: recordsWithIndex,
      pageNum: current,
      pageSize: size,
      total
    };
  },
  columns: () => [
    {
      key: 'index',
      title: $t('common.index'),
      align: 'center',
      width: 64
    },
    {
      key: 'progress',
      title: '进度',
      align: 'center',
      width: 100,
      render: row => {
        const color = getProgressTagColor(row.progress);
        return (
          <NTag type={color} size="small">
            {row.progress}
          </NTag>
        );
      }
    },
    {
      key: 'order_title',
      title: '工单标题',
      align: 'center',
      width: 180,
      ellipsis: {
        tooltip: true
      }
    },
    {
      key: 'execution_mode',
      title: '执行方式',
      align: 'center',
      width: 150,
      render: row => {
        if (row.schedule_time) {
          return (
            <div style="display: flex; flex-direction: column; align-items: center; font-size: 12px;">
              <NTag type="info" size="small" style="margin-bottom: 4px;">
                定时执行
              </NTag>
              <span style="color: #666;">{row.schedule_time}</span>
            </div>
          );
        }
        return (
          <NTag type="default" size="small">
            立即执行
          </NTag>
        );
      }
    },
    {
      key: 'applicant',
      title: '申请人',
      align: 'center',
      width: 100
    },
    {
      key: 'sql_type',
      title: 'SQL类型',
      align: 'center',
      width: 100
    },
    {
      key: 'environment',
      title: '环境',
      align: 'center',
      width: 100,
      render: row => {
        const tagMap: Record<string, NaiveUI.ThemeColor> = {
          test: 'primary',
          prod: 'error',
          dev: 'info'
        };
        const type = tagMap[row.environment] || 'default';
        return (
          <NTag type={type} size="small">
            {row.environment}
          </NTag>
        );
      }
    },
    {
      key: 'instance',
      title: '实例',
      align: 'center',
      width: 150,
      ellipsis: {
        tooltip: true
      }
    },
    {
      key: 'schema',
      title: '库名',
      align: 'center',
      width: 100,
      ellipsis: {
        tooltip: true
      }
    },
    {
      key: 'created_at',
      title: '创建时间',
      align: 'center',
      width: 180
    }
  ]
});

// Auto refresh timer
let refreshTimer: ReturnType<typeof setInterval> | null = null;

onMounted(() => {
  // Auto refresh every 30 seconds
  refreshTimer = setInterval(() => {
    getData();
  }, 30000);
});

onUnmounted(() => {
  if (refreshTimer) {
    clearInterval(refreshTimer);
  }
});
</script>

<template>
  <div class="min-h-500px flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
    <OrderSearch v-model:model="searchParams" @search="getDataByPage" @reset="handleReset" />
    <NCard title="工单列表" :bordered="false" size="small" class="card-wrapper sm:flex-1-hidden">
      <template #header-extra>
        <div class="flex-y-center gap-12px">
          <span class="text-14px">只看我的</span>
          <NSwitch v-model:value="onlyMyOrders" size="small" @update:value="handleMyOrdersChange" />
        </div>
      </template>
      <NDataTable
        :columns="columns"
        :data="data"
        :flex-height="!appStore.isMobile"
        :scroll-x="962"
        :loading="loading"
        remote
        :row-key="row => row.order_id"
        :pagination="mobilePagination"
        :row-props="rowProps"
        class="sm:h-full"
      />
    </NCard>
  </div>
</template>

<style scoped></style>
