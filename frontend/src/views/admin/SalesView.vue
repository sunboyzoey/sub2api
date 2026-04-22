<template>
  <AppLayout>
    <div class="space-y-6 p-4 sm:p-6">
      <div
        v-if="notice && noticeType === 'error'"
        class="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900/40 dark:bg-red-950/20 dark:text-red-300"
      >
        {{ notice }}
      </div>

      <section class="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div class="mb-4">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">渠道商</h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            新增渠道商后，直接在对应渠道下配置套餐价格。
          </p>
        </div>

        <form class="grid gap-3 md:grid-cols-[minmax(0,1fr),auto]" @submit.prevent="submitPartner">
          <input v-model="partnerForm.name" class="input" type="text" placeholder="渠道商名称" />
          <div class="flex gap-2">
            <button class="btn btn-primary" type="submit">{{ partnerForm.id ? '保存渠道商' : '添加渠道商' }}</button>
            <button v-if="partnerForm.id" class="btn btn-secondary" type="button" @click="resetPartnerForm">取消编辑</button>
          </div>
        </form>

        <div class="mt-5 space-y-4">
          <div
            v-for="item in partners"
            :key="item.id"
            class="rounded-2xl border border-gray-200 p-4 dark:border-dark-700"
          >
            <div class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
              <button
                class="group flex min-w-0 flex-1 items-center justify-between gap-3 rounded-2xl px-1 py-1 text-left transition-colors hover:bg-gray-50/80 active:bg-gray-100/70 focus:outline-none dark:hover:bg-dark-900/60 dark:active:bg-dark-900"
                type="button"
                @click="togglePartnerExpanded(item.id)"
              >
                <div class="min-w-0">
                  <div class="flex flex-wrap items-center gap-2">
                    <div class="text-sm font-semibold text-gray-900 dark:text-white">{{ item.name }}</div>
                    <span
                      class="inline-flex rounded-full px-2 py-0.5 text-xs font-medium"
                      :class="item.status === 'active'
                        ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
                        : 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300'"
                    >
                      {{ item.status === 'active' ? '启用' : '停用' }}
                    </span>
                    <span class="inline-flex rounded-full bg-gray-100 px-2.5 py-0.5 text-xs font-medium text-gray-500 dark:bg-dark-700 dark:text-gray-300">
                      {{ getPartnerProductRows(item.id).length }} 个套餐
                    </span>
                  </div>
                  <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    密钥尾号 {{ item.secret_hint || '-' }}
                  </div>
                </div>
                <div
                  class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-white text-gray-400 shadow-sm transition-all group-hover:-translate-y-0.5 group-hover:text-primary-500 dark:bg-dark-800 dark:text-gray-500"
                >
                  <svg
                    class="h-4 w-4 transition-transform duration-200"
                    :class="isPartnerExpanded(item.id) ? 'rotate-180' : ''"
                    viewBox="0 0 20 20"
                    fill="none"
                    xmlns="http://www.w3.org/2000/svg"
                  >
                    <path
                      d="M5 7.5L10 12.5L15 7.5"
                      stroke="currentColor"
                      stroke-width="1.8"
                      stroke-linecap="round"
                      stroke-linejoin="round"
                    />
                  </svg>
                </div>
              </button>
              <div class="flex flex-wrap gap-2">
                <button class="btn btn-secondary btn-sm" type="button" @click="editPartner(item)">编辑</button>
                <button class="btn btn-secondary btn-sm" type="button" @click="copyPartnerSecret(item.id)">复制密钥</button>
                <button class="btn btn-secondary btn-sm" type="button" @click="downloadSalesAPIDocument(item)">
                  下载文档
                </button>
                <button class="btn btn-secondary btn-sm" type="button" @click="downloadPartnerTestScript(item)">
                  下载脚本
                </button>
                <button class="btn btn-secondary btn-sm" type="button" @click="handleRotatePartnerSecret(item.id)">重置密钥</button>
                <button class="btn btn-danger btn-sm" type="button" @click="handleDeletePartner(item.id)">删除</button>
              </div>
            </div>

            <div
              v-if="isPartnerExpanded(item.id)"
              class="mt-4 rounded-2xl bg-gradient-to-br from-gray-50 to-white p-4 shadow-inner dark:from-dark-900 dark:to-dark-800"
            >
              <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
                <div class="text-sm font-medium text-gray-900 dark:text-white">套餐价格</div>
                <button class="btn btn-secondary btn-sm" type="button" @click="startCreateProduct(item.id)">
                  新增套餐价格
                </button>
              </div>

              <form
                v-if="activeProductPartnerId === item.id"
                class="mt-4 grid gap-3 md:grid-cols-[minmax(0,1fr),120px,180px,auto]"
                @submit.prevent="submitProduct"
              >
                <select v-model.number="productForm.group_id" class="input">
                  <option :value="0">选择分组套餐</option>
                  <option v-for="group in subscriptionGroups" :key="group.id" :value="group.id">
                    {{ group.name }} / {{ group.platform }}
                  </option>
                </select>
                <select v-model="productForm.cycle_unit" class="input">
                  <option value="day">1天</option>
                  <option value="month">1月</option>
                </select>
                <input
                  v-model.number="productForm.sale_price"
                  class="input"
                  type="number"
                  min="0"
                  step="0.01"
                  placeholder="价格"
                />
                <div class="flex gap-2">
                  <button class="btn btn-primary" type="submit">{{ productForm.mapping_id ? '保存' : '新增' }}</button>
                  <button class="btn btn-secondary" type="button" @click="cancelProductEdit">取消</button>
                </div>
              </form>

              <div class="mt-4 space-y-2">
                <div
                  v-for="mapping in getPartnerProductRows(item.id)"
                  :key="mapping.mapping.id"
                  class="flex flex-col gap-3 rounded-xl border border-gray-200 bg-white px-4 py-3 dark:border-dark-700 dark:bg-dark-800 lg:flex-row lg:items-center lg:justify-between"
                >
                  <div>
                    <div class="text-sm font-medium text-gray-900 dark:text-white">
                      {{ mapping.groupName }} / {{ mapping.cycleLabel }}
                    </div>
                    <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                      价格 {{ formatPrice(mapping.mapping.sale_price) }}
                    </div>
                  </div>
                  <div class="flex flex-wrap gap-2">
                    <button class="btn btn-secondary btn-sm" type="button" @click="editProduct(mapping.mapping)">编辑</button>
                    <button class="btn btn-danger btn-sm" type="button" @click="handleDeleteProduct(mapping.mapping)">删除</button>
                  </div>
                </div>
                <div
                  v-if="getPartnerProductRows(item.id).length === 0"
                  class="rounded-xl border border-dashed border-gray-300 px-4 py-6 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400"
                >
                  暂无套餐价格
                </div>
              </div>
            </div>
          </div>

          <div
            v-if="partners.length === 0"
            class="rounded-xl border border-dashed border-gray-300 px-4 py-8 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400"
          >
            暂无渠道商
          </div>
        </div>
      </section>

      <section class="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div class="mb-5 flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">订单记录</h2>
          <div class="flex flex-wrap gap-2">
            <button class="btn btn-secondary" type="button" @click="loadOrders">刷新订单</button>
            <button
              class="btn btn-danger"
              type="button"
              :disabled="selectedOrderIds.size === 0"
              @click="handleBatchDeleteOrders"
            >
              批量删除
            </button>
          </div>
        </div>

        <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          <div class="flex items-center gap-2">
            <input
              id="order-select-all"
              type="checkbox"
              class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :checked="isAllOrdersSelected"
              @change="toggleSelectAll(($event.target as HTMLInputElement | null)?.checked)"
            />
            <label class="text-sm text-gray-700 dark:text-gray-300" for="order-select-all">
              当前列表全选
            </label>
          </div>
          <select v-model.number="orderFilters.partner_id" class="input">
            <option :value="0">全部渠道商</option>
            <option v-for="item in partners" :key="item.id" :value="item.id">
              {{ item.name }}
            </option>
          </select>
          <select v-model="orderFilters.status" class="input">
            <option value="">全部状态</option>
            <option value="pending">待处理</option>
            <option value="subscription_applied">已开通订阅</option>
            <option value="fulfilled">已完成</option>
            <option value="failed">失败</option>
          </select>
          <input
            v-model="orderFilters.search"
            class="input md:col-span-2"
            type="text"
            placeholder="搜索订单号"
            @keyup.enter="loadOrders"
          />
          <div class="md:col-span-2 xl:col-span-4 flex gap-2">
            <button class="btn btn-primary" type="button" @click="loadOrders">查询</button>
            <button class="btn btn-secondary" type="button" @click="resetOrderFilters">重置</button>
          </div>
        </div>

        <div class="mt-5 overflow-x-auto">
          <table class="min-w-full text-left text-sm">
            <thead class="text-gray-500 dark:text-gray-400">
              <tr>
                <th class="pb-3 pr-4">
                  <span class="sr-only">选择</span>
                </th>
                <th class="pb-3 pr-4">订单号</th>
                <th class="pb-3 pr-4">渠道商</th>
                <th class="pb-3 pr-4">关联分组</th>
                <th class="pb-3 pr-4">套餐价格</th>
                <th class="pb-3 pr-4">状态</th>
                <th class="pb-3 pr-4">到期时间</th>
                <th class="pb-3 pr-4">创建时间</th>
                <th class="pb-3">操作</th>
              </tr>
            </thead>
            <tbody>
              <template v-for="item in orders" :key="item.id">
                <tr class="border-t border-gray-100 dark:border-dark-700">
                  <td class="py-3 pr-3 align-top">
                    <input
                      type="checkbox"
                      class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                      :checked="selectedOrderIds.has(item.id)"
                      @change="toggleOrderSelection(item.id, ($event.target as HTMLInputElement | null)?.checked)"
                    />
                  </td>
                  <td class="py-3 pr-4 align-top">
                    <div class="font-medium text-gray-900 dark:text-white">{{ item.external_order_id }}</div>
                    <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                      {{ orderTypeLabel(item.order_type) }}
                    </div>
                  </td>
                  <td class="py-3 pr-4 align-top">
                    {{ orderPartnerLabel(item) }}
                  </td>
                  <td class="py-3 pr-4 align-top">
                    <div class="font-medium text-gray-900 dark:text-white">
                      {{ item.package?.group?.name || item.package?.name || item.package_id }}
                    </div>
                    <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                      {{ formatPackageCycle(item.package) }}
                    </div>
                  </td>
                  <td class="py-3 pr-4 align-top">
                    <div class="font-medium text-gray-900 dark:text-white">
                      {{ formatOrderAmount(item) }}
                    </div>
                  </td>
                  <td class="py-3 pr-4 align-top">
                    <span
                      class="inline-flex rounded-full px-2.5 py-1 text-xs font-medium"
                      :class="orderStatusClass(item.status)"
                    >
                      {{ orderStatusLabel(item.status) }}
                    </span>
                  </td>
                  <td class="py-3 pr-4 align-top">{{ formatExpiresAt(item) }}</td>
                  <td class="py-3 pr-4 align-top">{{ formatDate(item.created_at) }}</td>
                  <td class="py-3 align-top">
                    <div class="flex flex-wrap gap-2">
                      <button class="btn btn-secondary btn-sm" type="button" @click="toggleOrderDetail(item.id)">
                        {{ orderDetailLoadingId === item.id ? '加载中...' : isOrderDetailOpen(item.id) ? '收起' : '详情' }}
                      </button>
                      <button class="btn btn-danger btn-sm" type="button" @click="handleDeleteOrder(item)">
                        删除
                      </button>
                    </div>
                  </td>
                </tr>
                <tr
                  v-if="selectedOrderDetail && selectedOrderDetail.order.id === item.id"
                  class="border-t border-gray-100 bg-gray-50/80 dark:border-dark-700 dark:bg-dark-900/60"
                >
                  <td colspan="8" class="p-4">
                    <div class="rounded-2xl border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
                      <div class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                        <div>
                          <h3 class="text-base font-semibold text-gray-900 dark:text-white">订单详情</h3>
                          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                            {{ selectedOrderDetail.order.external_order_id }}
                          </p>
                        </div>
                        <button class="btn btn-secondary btn-sm" type="button" @click="closeOrderDetail">
                          关闭
                        </button>
                      </div>

                      <div class="mt-4 grid gap-4 md:grid-cols-2 xl:grid-cols-4">
                        <div>
                          <div class="text-xs text-gray-500 dark:text-gray-400">渠道商</div>
                          <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                            {{ orderPartnerLabel(selectedOrderDetail.order) }}
                          </div>
                        </div>
                        <div>
                          <div class="text-xs text-gray-500 dark:text-gray-400">分组套餐</div>
                          <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                            {{ selectedOrderDetail.order.package?.group?.name || selectedOrderDetail.order.package?.name || selectedOrderDetail.order.package_id }}
                          </div>
                        </div>
                        <div>
                          <div class="text-xs text-gray-500 dark:text-gray-400">订单类型</div>
                          <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                            {{ orderTypeLabel(selectedOrderDetail.order.order_type) }}
                          </div>
                        </div>
                        <div>
                          <div class="text-xs text-gray-500 dark:text-gray-400">状态</div>
                          <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                            {{ orderStatusLabel(selectedOrderDetail.order.status) }}
                          </div>
                        </div>
                        <div>
                          <div class="text-xs text-gray-500 dark:text-gray-400">套餐价格</div>
                          <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                            {{ formatOrderAmount(selectedOrderDetail.order) }}
                          </div>
                        </div>
                        <div>
                          <div class="text-xs text-gray-500 dark:text-gray-400">到期时间</div>
                          <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                            {{ formatExpiresAt(selectedOrderDetail.order) }}
                          </div>
                        </div>
                        <div>
                          <div class="text-xs text-gray-500 dark:text-gray-400">创建时间</div>
                          <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                            {{ formatDate(selectedOrderDetail.order.created_at) }}
                          </div>
                        </div>
                        <div class="md:col-span-2 xl:col-span-4">
                          <div class="text-xs text-gray-500 dark:text-gray-400">返回 Key</div>
                          <div class="mt-1 break-all text-sm font-medium text-gray-900 dark:text-white">
                            {{ selectedOrderDetail.api_key_value || '-' }}
                          </div>
                        </div>
                      </div>
                    </div>
                  </td>
                </tr>
              </template>
              <tr v-if="orders.length === 0">
                <td colspan="8" class="py-10 text-center text-sm text-gray-500 dark:text-gray-400">
                  暂无订单记录
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import adminAPI from '@/api/admin'
import type {
  SalesMapping,
  SalesOrder,
  SalesPartner,
  SalesPackage,
  SalesProvisionResult
} from '@/api/admin/sales'

type GroupOption = {
  id: number
  name: string
  platform: string
  subscription_type?: string
  default_validity_days?: number
}

type ProductCycle = 'day' | 'month'

const notice = ref('')
const noticeType = ref<'success' | 'error'>('success')
const latestSecret = ref('')
const latestSecretPartnerId = ref(0)
const activeProductPartnerId = ref(0)
const expandedPartnerId = ref(0)

const partners = ref<SalesPartner[]>([])
const mappings = ref<SalesMapping[]>([])
const orders = ref<SalesOrder[]>([])
const groups = ref<GroupOption[]>([])
const selectedOrderDetail = ref<SalesProvisionResult | null>(null)
const orderDetailLoadingId = ref(0)

const partnerForm = reactive({
  id: 0,
  name: ''
})

const productForm = reactive({
  mapping_id: 0,
  package_id: 0,
  partner_id: 0,
  group_id: 0,
  cycle_unit: 'month' as ProductCycle,
  sale_price: 0
})

const orderFilters = reactive({
  partner_id: 0,
  status: '',
  search: ''
})
const selectedOrderIds = ref<Set<number>>(new Set())

const subscriptionGroups = computed(() =>
  groups.value.filter((item) => item.subscription_type === 'subscription')
)

const productRows = computed(() =>
  mappings.value
    .map((mapping) => ({
      mapping,
      partnerName: mapping.partner?.name || `渠道商 ${mapping.partner_id}`,
      groupName: mapping.package?.group?.name || mapping.package?.name || `分组 ${mapping.package_id}`,
      cycleLabel: formatPackageCycle(mapping.package)
    }))
    .sort((a, b) => {
      if (a.mapping.partner_id !== b.mapping.partner_id) {
        return a.mapping.partner_id - b.mapping.partner_id
      }
      if (a.groupName !== b.groupName) {
        return a.groupName.localeCompare(b.groupName, 'zh-CN')
      }
      return productCycleSortValue(a.mapping.package) - productCycleSortValue(b.mapping.package)
    })
)

function setNotice(message: string, type: 'success' | 'error' = 'success') {
  notice.value = message
  noticeType.value = type
}

function clearNotice() {
  notice.value = ''
}

function formatError(error: unknown): string {
  const err = error as {
    response?: { data?: { message?: string; error?: string } }
    message?: string
  }
  return err?.response?.data?.message || err?.response?.data?.error || err?.message || '请求失败'
}

function formatDate(value?: string | null): string {
  if (!value) return '-'
  try {
    return new Date(value).toLocaleString()
  } catch {
    return value
  }
}

function formatPrice(value?: number): string {
  const amount = Number(value || 0)
  return amount.toFixed(2)
}

function formatOrderAmount(order?: SalesOrder | null): string {
  if (!order) return '-'
  return formatPrice(order.amount)
}

function productCycleSortValue(pkg?: SalesPackage | null): number {
  if (!pkg) return 99
  if (pkg.cycle_unit === 'day') return 1
  if (pkg.cycle_unit === 'month') return 2
  return 99
}

function formatPackageCycle(pkg?: SalesPackage | null): string {
  if (!pkg) return '-'
  if (pkg.cycle_unit === 'day') {
    return `${pkg.cycle_count || 1}天`
  }
  if (pkg.cycle_unit === 'month') {
    return `${pkg.cycle_count || 1}月`
  }
  if (pkg.validity_days) {
    return `${pkg.validity_days}天`
  }
  return '-'
}

function getProductCycleConfig(cycleUnit: ProductCycle) {
  if (cycleUnit === 'day') {
    return {
      cycleUnit: 'day' as const,
      cycleCount: 1,
      validityDays: 1,
      suffix: '1天',
      codeSuffix: '1d'
    }
  }
  return {
    cycleUnit: 'month' as const,
    cycleCount: 1,
    validityDays: 30,
    suffix: '1月',
    codeSuffix: '1m'
  }
}

function resolveEditableProductCycle(pkg?: SalesPackage | null): ProductCycle | null {
  if (!pkg) return 'month'
  if (pkg.cycle_unit === 'day' && pkg.cycle_count === 1) {
    return 'day'
  }
  if (pkg.cycle_unit === 'month' && pkg.cycle_count === 1) {
    return 'month'
  }
  return null
}

function getAPIBaseURL(): string {
  return `${window.location.origin}/api/v1`
}

function sanitizeFileName(value: string): string {
  const cleaned = value.trim().replace(/[^\w\u4e00-\u9fa5-]+/g, '-').replace(/-+/g, '-')
  return cleaned || 'partner'
}

function getEmbeddedPartnerSecret(partner?: SalesPartner): string {
  if (!partner) return 'YOUR_SALES_SECRET'
  if (latestSecretPartnerId.value === partner.id && latestSecret.value) {
    return latestSecret.value
  }
  return 'YOUR_SALES_SECRET'
}

function downloadTextFile(filename: string, content: string, type: string) {
  const blob = new Blob([content], { type })
  const url = window.URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  window.URL.revokeObjectURL(url)
}

function buildSalesAPIDocument(partner?: SalesPartner): string {
  const baseURL = getAPIBaseURL()
  const partnerName = partner?.name?.trim() || '渠道商'
  const embeddedSecret = getEmbeddedPartnerSecret(partner)
  const secretSection =
    embeddedSecret === 'YOUR_SALES_SECRET'
      ? '当前文档未带入真实密钥。如需直接交付第三方，请先在该渠道商卡片点击“重置密钥”后重新下载。'
      : `当前渠道密钥：\`${embeddedSecret}\``

  return `# 渠道订单对接 API 文档

版本：v1
基础地址：\`${baseURL}\`
适用渠道：${partnerName}
${secretSection}

## 鉴权方式

所有渠道接口只需要提供密钥，不再需要渠道编号。

请求头：

\`\`\`http
X-Sales-Partner-Secret: ${embeddedSecret}
\`\`\`

也兼容：

\`\`\`http
Authorization: Bearer ${embeddedSecret}
\`\`\`

## 统一返回格式

成功：

\`\`\`json
{
  "code": 0,
  "message": "success",
  "data": {}
}
\`\`\`

失败：

\`\`\`json
{
  "code": 401,
  "message": "invalid partner credentials",
  "reason": "SALES_INVALID_SECRET"
}
\`\`\`

## 1. 获取本渠道套餐价格

\`\`\`http
GET /sales/partner/packages?page=1&page_size=20
\`\`\`

请求示例：

\`\`\`bash
curl -X GET "${baseURL}/sales/partner/packages?page=1&page_size=20" \\
  -H "X-Sales-Partner-Secret: ${embeddedSecret}"
\`\`\`

返回字段重点：

- \`package_id\`：下单时直接使用的套餐 ID
- \`name\`：套餐名称
- \`package_type\`：套餐类型，返回 \`day\` 或 \`month\`
- \`content\`：套餐内容
- \`price\`：套餐价格

## 2. 提交开通/续订订单

\`\`\`http
POST /sales/partner/orders/provision
\`\`\`

首购请求体示例：

\`\`\`json
{
  "external_order_id": "ORDER_20260411_0001",
  "package_id": 4,
  "order_type": "purchase",
  "amount": 99,
  "currency": "CNY",
  "raw_payload": {
    "source": "partner-test"
  }
}
\`\`\`

续订请求体示例：

\`\`\`json
{
  "external_order_id": "ORDER_20260412_0001",
  "api_key": "sk-xxxxxxxx",
  "order_type": "renewal",
  "amount": 99,
  "currency": "CNY"
}
\`\`\`

字段说明：

- \`external_order_id\`：渠道订单号，同一渠道内必须唯一
- \`package_id\`：首购时必填，直接使用套餐列表接口返回的值
- \`api_key\`：续订时必填，系统会按该 Key 在当前渠道最近一笔已完成订单自动反查原套餐
- \`order_type\`：可选 \`purchase\` / \`renewal\` / \`manual\`
- \`external_user_id\` / \`external_email\` / \`external_name\`：可选，仅作为扩展资料；渠道对接时可不传

成功后返回：

- \`order_id\`：订单号
- \`api_key\`：最终返回给用户的密钥
- \`api_base_url\`：用户实际调用的 API 地址
- \`created_at\`：订单创建时间

## 3. 查询订单结果

\`\`\`http
GET /sales/partner/orders/{external_order_id}
\`\`\`

请求示例：

\`\`\`bash
curl -X GET "${baseURL}/sales/partner/orders/ORDER_20260411_0001" \\
  -H "X-Sales-Partner-Secret: ${embeddedSecret}"
\`\`\`

适用场景：

- 下单超时后的补查
- 轮询订单处理结果
- 重新获取 \`api_key\`

## 4. 业务规则

- 同一渠道内，\`external_order_id\` 必须唯一
- 相同 \`external_order_id\` 重试时，系统按幂等处理
- 首购只需要 \`package_id\`
- 续订只需要 \`api_key\`，套餐会按该 Key 的最近成交记录自动反查
- 当前价格固定按人民币处理

## 5. 推荐对接流程

1. 先调用 \`/sales/partner/packages\` 获取套餐 ID
2. 用户首次购买时，使用 \`package_id\` 调用 \`/sales/partner/orders/provision\`
3. 保存返回中的 \`api_key\`
4. 用户后续续订时，只传 \`api_key\` 再次调用 \`/sales/partner/orders/provision\`
5. 如果超时或处理中，再调用 \`/sales/partner/orders/{external_order_id}\` 查询
`
}

function buildPartnerTestScript(partner: SalesPartner): string {
  const baseURL = getAPIBaseURL()
  const embeddedSecret = getEmbeddedPartnerSecret(partner)

  return `#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${baseURL}"
SALES_SECRET="${embeddedSecret}"
PURCHASE_ORDER_ID="\${PURCHASE_ORDER_ID:-TEST-BUY-$(date +%s)}"
RENEW_ORDER_ID="\${RENEW_ORDER_ID:-TEST-RENEW-$(date +%s)}"

if [ "\${SALES_SECRET}" = "YOUR_SALES_SECRET" ]; then
  echo "请先把 SALES_SECRET 改成真实渠道密钥。"
  echo "如果后台已无法回显旧密钥，请先在管理界面重置密钥后重新下载脚本。"
  exit 1
fi

pretty_print() {
  if command -v jq >/dev/null 2>&1; then
    jq .
  else
    cat
  fi
}

echo "===> 1. 获取套餐列表"
PACKAGES_RESPONSE=$(curl -sS "${baseURL}/sales/partner/packages?page=1&page_size=20" \\
  -H "X-Sales-Partner-Secret: \${SALES_SECRET}")
printf '%s\n' "\${PACKAGES_RESPONSE}" | pretty_print

PACKAGE_ID="\${PACKAGE_ID:-$(printf '%s' "\${PACKAGES_RESPONSE}" | python3 -c 'import sys, json; data=json.load(sys.stdin); items=data.get(\"data\", {}).get(\"items\", []); print(items[0].get(\"package_id\", \"\") if items else \"\")')}"

if [ -z "\${PACKAGE_ID}" ]; then
  echo "未获取到 package_id，请先在后台为该渠道配置套餐价格。"
  exit 1
fi

echo "使用套餐ID: \${PACKAGE_ID}"

extract_data_field() {
  local field="$1"
  if command -v jq >/dev/null 2>&1; then
    jq -r ".data.\${field} // empty"
  else
    python3 -c 'import sys, json; field = sys.argv[1]; data = json.load(sys.stdin); print((data.get("data") or {}).get(field, ""))' "$field"
  fi
}

PURCHASE_PAYLOAD=$(cat <<JSON
{
  "external_order_id": "\${PURCHASE_ORDER_ID}",
  "package_id": \${PACKAGE_ID},
  "order_type": "purchase",
  "amount": 1,
  "currency": "CNY",
  "raw_payload": {
    "source": "sales-script",
    "partner_name": "${partner.name}"
  }
}
JSON
)

echo "===> 2. 首购开通"
ORDER_RESPONSE=$(curl -sS -X POST "${baseURL}/sales/partner/orders/provision" \\
  -H "Content-Type: application/json" \\
  -H "X-Sales-Partner-Secret: \${SALES_SECRET}" \\
  -d "\${PURCHASE_PAYLOAD}")
printf '%s\n' "\${ORDER_RESPONSE}" | pretty_print

API_KEY="$(printf '%s' "\${ORDER_RESPONSE}" | extract_data_field api_key)"
if [ -z "\${API_KEY}" ]; then
  echo "未从首购返回中取到 api_key，请检查渠道密钥、套餐映射和订单返回内容。"
  exit 1
fi

echo "首购返回的 API Key: \${API_KEY}"

echo "===> 3. 查询首购订单"
QUERY_RESPONSE=$(curl -sS "${baseURL}/sales/partner/orders/\${PURCHASE_ORDER_ID}" \\
  -H "X-Sales-Partner-Secret: \${SALES_SECRET}")
printf '%s\n' "\${QUERY_RESPONSE}" | pretty_print

RENEW_PAYLOAD=$(cat <<JSON
{
  "external_order_id": "\${RENEW_ORDER_ID}",
  "api_key": "\${API_KEY}",
  "order_type": "renewal",
  "amount": 1,
  "currency": "CNY",
  "raw_payload": {
    "source": "sales-script-renew",
    "partner_name": "${partner.name}"
  }
}
JSON
)

echo "===> 4. 按 API Key 续订"
RENEW_RESPONSE=$(curl -sS -X POST "${baseURL}/sales/partner/orders/provision" \\
  -H "Content-Type: application/json" \\
  -H "X-Sales-Partner-Secret: \${SALES_SECRET}" \\
  -d "\${RENEW_PAYLOAD}")
printf '%s\n' "\${RENEW_RESPONSE}" | pretty_print
`
}

function downloadSalesAPIDocument(partner?: SalesPartner) {
  try {
    const filename = `${sanitizeFileName(partner?.name || 'sales')}-api-doc.md`
    downloadTextFile(filename, buildSalesAPIDocument(partner), 'text/markdown;charset=utf-8')
  } catch {
    setNotice('下载 API 文档失败', 'error')
  }
}

function downloadPartnerTestScript(partner: SalesPartner) {
  try {
    const filename = `${sanitizeFileName(partner.name)}-sales-test.sh`
    downloadTextFile(filename, buildPartnerTestScript(partner), 'text/x-shellscript;charset=utf-8')
  } catch {
    setNotice('下载测试脚本失败', 'error')
  }
}

function orderTypeLabel(value?: string): string {
  switch (value) {
    case 'renewal':
      return '续订'
    case 'manual':
      return '手动补发'
    case 'purchase':
    default:
      return '开通'
  }
}

function orderStatusLabel(value?: string): string {
  switch (value) {
    case 'pending':
      return '待处理'
    case 'subscription_applied':
      return '已开通订阅'
    case 'fulfilled':
      return '已完成'
    case 'failed':
      return '失败'
    default:
      return value || '-'
  }
}

function orderStatusClass(value?: string): string {
  switch (value) {
    case 'fulfilled':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
    case 'subscription_applied':
      return 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
    case 'failed':
      return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
    case 'pending':
    default:
      return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
  }
}

function formatExpiresAt(item: SalesOrder): string {
  const expiresAt = item.result_snapshot?.expires_at
  if (typeof expiresAt === 'string' && expiresAt) {
    return formatDate(expiresAt)
  }
  return '-'
}

function buildExternalPackageCode(groupID: number, cycleUnit: ProductCycle): string {
  const { codeSuffix } = getProductCycleConfig(cycleUnit)
  return `group-${groupID}-${codeSuffix}`
}

function buildInternalProductCode(partnerID: number, groupID: number, cycleUnit: ProductCycle): string {
  const { codeSuffix } = getProductCycleConfig(cycleUnit)
  return `sales-p${partnerID}-g${groupID}-${codeSuffix}-${Date.now()}`
}

function getGroupByID(groupID: number): GroupOption | undefined {
  return groups.value.find((item) => item.id === groupID)
}

function getPartnerProductRows(partnerID: number) {
  return productRows.value.filter((item) => item.mapping.partner_id === partnerID)
}

function isPartnerExpanded(partnerID: number): boolean {
  return expandedPartnerId.value === partnerID
}

function expandPartner(partnerID: number) {
  expandedPartnerId.value = partnerID
}

function togglePartnerExpanded(partnerID: number) {
  if (expandedPartnerId.value === partnerID) {
    expandedPartnerId.value = 0
    if (activeProductPartnerId.value === partnerID) {
      cancelProductEdit()
    }
    return
  }
  expandedPartnerId.value = partnerID
}

async function loadBaseData() {
  const [groupsData, partnersData, mappingsData] = await Promise.all([
    adminAPI.groups.getAll(),
    adminAPI.sales.listPartners(1, 100),
    adminAPI.sales.listMappings(1, 100)
  ])
  groups.value = groupsData as GroupOption[]
  partners.value = partnersData.items
  mappings.value = mappingsData.items
  if (expandedPartnerId.value && !partners.value.some((item) => item.id === expandedPartnerId.value)) {
    expandedPartnerId.value = 0
  }
}

async function loadOrders() {
  try {
    const data = await adminAPI.sales.listOrders(1, 100, {
      partner_id: orderFilters.partner_id || undefined,
      status: orderFilters.status || undefined,
      search: orderFilters.search || undefined
    })
    orders.value = data.items
    selectedOrderIds.value.clear()
  } catch (error) {
    setNotice(formatError(error), 'error')
  }
}

async function loadAll() {
  clearNotice()
  try {
    await Promise.all([loadBaseData(), loadOrders()])
  } catch (error) {
    setNotice(formatError(error), 'error')
  }
}

function resetPartnerForm() {
  partnerForm.id = 0
  partnerForm.name = ''
}

function editPartner(item: SalesPartner) {
  partnerForm.id = item.id
  partnerForm.name = item.name
}

async function submitPartner() {
  clearNotice()
  if (!partnerForm.name.trim()) {
    setNotice('请输入渠道商名称', 'error')
    return
  }
  try {
    if (partnerForm.id) {
      await adminAPI.sales.updatePartner(partnerForm.id, {
        name: partnerForm.name.trim()
      })
    } else {
      const result = await adminAPI.sales.createPartner({
        name: partnerForm.name.trim()
      })
      latestSecret.value = result.secret
      latestSecretPartnerId.value = result.partner.id
    }
    resetPartnerForm()
    await loadBaseData()
  } catch (error) {
    setNotice(formatError(error), 'error')
  }
}

async function handleRotatePartnerSecret(id: number) {
  clearNotice()
  try {
    const result = await adminAPI.sales.rotatePartnerSecret(id)
    latestSecret.value = result.secret
    latestSecretPartnerId.value = result.partner.id
    await loadBaseData()
  } catch (error) {
    setNotice(formatError(error), 'error')
  }
}

async function handleDeletePartner(id: number) {
  if (!window.confirm('确认删除该渠道商吗？')) return
  clearNotice()
  try {
    await adminAPI.sales.deletePartner(id)
    if (activeProductPartnerId.value === id) {
      cancelProductEdit()
    }
    await loadBaseData()
  } catch (error) {
    setNotice(formatError(error), 'error')
  }
}

function resetProductForm() {
  productForm.mapping_id = 0
  productForm.package_id = 0
  productForm.partner_id = 0
  productForm.group_id = 0
  productForm.cycle_unit = 'month'
  productForm.sale_price = 0
}

function startCreateProduct(partnerID: number) {
  clearNotice()
  resetProductForm()
  productForm.partner_id = partnerID
  activeProductPartnerId.value = partnerID
  expandPartner(partnerID)
}

function cancelProductEdit() {
  resetProductForm()
  activeProductPartnerId.value = 0
}

function editProduct(item: SalesMapping) {
  clearNotice()
  const editableCycle = resolveEditableProductCycle(item.package)
  if (!editableCycle) {
    setNotice('当前渠道套餐只支持 1天 或 1月，旧时长套餐请删除后重建', 'error')
    return
  }
  productForm.mapping_id = item.id
  productForm.package_id = item.package_id
  productForm.partner_id = item.partner_id
  productForm.group_id = item.package?.group_id || 0
  productForm.cycle_unit = editableCycle
  productForm.sale_price = item.sale_price
  activeProductPartnerId.value = item.partner_id
  expandPartner(item.partner_id)
}

async function submitProduct() {
  clearNotice()
  if (!productForm.partner_id || !productForm.group_id) {
    setNotice('请先选择分组套餐', 'error')
    return
  }

  try {
    const selectedGroup = getGroupByID(productForm.group_id)
    if (!selectedGroup) {
      setNotice('未找到对应的分组套餐', 'error')
      return
    }
    const cycleConfig = getProductCycleConfig(productForm.cycle_unit)

    const existingRow = mappings.value.find(
      (item) =>
        item.id !== productForm.mapping_id &&
        item.partner_id === productForm.partner_id &&
        item.package?.group_id === productForm.group_id &&
        item.package?.cycle_unit === cycleConfig.cycleUnit &&
        item.package?.cycle_count === cycleConfig.cycleCount
    )

    const externalPackageCode = buildExternalPackageCode(productForm.group_id, productForm.cycle_unit)
    const externalPackageName = `${selectedGroup.name} ${cycleConfig.suffix}`
    let mappingID = productForm.mapping_id
    let packageID = productForm.package_id

    if (!mappingID && existingRow) {
      mappingID = existingRow.id
      packageID = existingRow.package_id
    }

    if (packageID) {
      await adminAPI.sales.updatePackage(packageID, {
        name: externalPackageName,
        group_id: productForm.group_id,
        description: '',
        cycle_unit: cycleConfig.cycleUnit,
        cycle_count: cycleConfig.cycleCount,
        validity_days: cycleConfig.validityDays,
        key_policy: 'reuse_current',
        auto_create_key: true,
        status: 'active',
        store_visible: false
      })
    } else {
      const createdPackage = await adminAPI.sales.createPackage({
        code: buildInternalProductCode(productForm.partner_id, productForm.group_id, productForm.cycle_unit),
        name: externalPackageName,
        group_id: productForm.group_id,
        cycle_unit: cycleConfig.cycleUnit,
        cycle_count: cycleConfig.cycleCount,
        validity_days: cycleConfig.validityDays,
        key_policy: 'reuse_current',
        auto_create_key: true,
        status: 'active',
        store_visible: false
      })
      packageID = createdPackage.id
    }

    await adminAPI.sales.upsertMapping({
      id: mappingID || undefined,
      partner_id: productForm.partner_id,
      package_id: packageID,
      external_package_code: externalPackageCode,
      external_package_name: externalPackageName,
      sale_price: productForm.sale_price,
      currency: 'CNY',
      status: 'active'
    })

    await loadBaseData()
    cancelProductEdit()
  } catch (error) {
    setNotice(formatError(error), 'error')
  }
}

async function handleDeleteProduct(item: SalesMapping) {
  if (!window.confirm('确认删除该套餐价格吗？')) return
  clearNotice()
  try {
    await adminAPI.sales.deleteMapping(item.id)
    if (item.package_id) {
      try {
        await adminAPI.sales.deletePackage(item.package_id)
      } catch {
        // Package cleanup is best effort only.
      }
    }
    if (productForm.mapping_id === item.id) {
      cancelProductEdit()
    }
    await loadBaseData()
  } catch (error) {
    setNotice(formatError(error), 'error')
  }
}

function resetOrderFilters() {
  orderFilters.partner_id = 0
  orderFilters.status = ''
  orderFilters.search = ''
  void loadOrders()
}

async function loadOrderDetail(id: number) {
  orderDetailLoadingId.value = id
  try {
    selectedOrderDetail.value = await adminAPI.sales.getOrder(id)
  } catch (error) {
    setNotice(formatError(error), 'error')
  } finally {
    if (orderDetailLoadingId.value === id) {
      orderDetailLoadingId.value = 0
    }
  }
}

function isOrderDetailOpen(id: number): boolean {
  return selectedOrderDetail.value?.order?.id === id
}

function closeOrderDetail() {
  selectedOrderDetail.value = null
  orderDetailLoadingId.value = 0
}

async function toggleOrderDetail(id: number) {
  if (isOrderDetailOpen(id)) {
    closeOrderDetail()
    return
  }
  await loadOrderDetail(id)
}

function toggleOrderSelection(id: number, checked?: boolean) {
  if (checked) {
    selectedOrderIds.value.add(id)
  } else {
    selectedOrderIds.value.delete(id)
  }
}

const isAllOrdersSelected = computed(() => {
  return orders.value.length > 0 && orders.value.every((o) => selectedOrderIds.value.has(o.id))
})

function toggleSelectAll(checked?: boolean) {
  selectedOrderIds.value.clear()
  if (checked) {
    orders.value.forEach((o) => selectedOrderIds.value.add(o.id))
  }
}

async function handleDeleteOrder(order: SalesOrder) {
  if (!window.confirm(`确认删除订单 ${order.external_order_id} 吗？此操作仅删除订单记录，不回收订阅或 Key。`)) return
  clearNotice()
  try {
    await adminAPI.sales.deleteOrder(order.id)
    if (isOrderDetailOpen(order.id)) {
      closeOrderDetail()
    }
    await loadOrders()
  } catch (error) {
    setNotice(formatError(error), 'error')
  }
}

async function handleBatchDeleteOrders() {
  if (selectedOrderIds.value.size === 0) return
  const ids = Array.from(selectedOrderIds.value)
  if (!window.confirm(`确认批量删除 ${ids.length} 条订单记录？此操作仅删除订单记录，不回收订阅或 Key。`)) return
  clearNotice()
  try {
    await adminAPI.sales.batchDeleteOrders({ ids })
    if (selectedOrderDetail.value && selectedOrderIds.value.has(selectedOrderDetail.value.order.id)) {
      closeOrderDetail()
    }
    await loadOrders()
  } catch (error) {
    setNotice(formatError(error), 'error')
  }
}

function orderPartnerLabel(order?: SalesOrder | null): string {
  if (!order) return '-'
  const partnerName = order.partner?.name?.trim()
  const partnerCode = order.partner?.code?.trim()
  if (partnerCode && partnerName && partnerCode !== partnerName) {
    return `${partnerName} (${partnerCode})`
  }
  return partnerCode || partnerName || String(order.partner_id)
}

async function copySecret() {
  if (!latestSecret.value) {
    setNotice('当前没有可复制的最新密钥', 'error')
    return
  }
  try {
    await navigator.clipboard.writeText(latestSecret.value)
    clearNotice()
  } catch {
    setNotice('复制失败', 'error')
  }
}

async function copyPartnerSecret(partnerID: number) {
  if (latestSecretPartnerId.value !== partnerID || !latestSecret.value) {
    setNotice('历史密钥不可逆，如需复制请先重置密钥', 'error')
    return
  }
  await copySecret()
}

onMounted(() => {
  loadAll()
})
</script>
