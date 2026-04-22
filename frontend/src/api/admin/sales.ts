import { apiClient } from '../client'

export interface SalesGroupSummary {
  id: number
  name: string
  description: string
  platform: string
  status: string
  subscription_type: string
  default_validity_days: number
}

export interface SalesPartner {
  id: number
  code: string
  name: string
  description: string
  status: string
  auth_mode: string
  secret_hint: string
  rate_limit_rpm: number
  created_at: string
  updated_at: string
}

export interface SalesPackage {
  id: number
  code: string
  name: string
  description: string
  platform: string
  group_id: number
  group?: SalesGroupSummary | null
  cycle_unit: 'day' | 'month'
  cycle_count: number
  validity_days: number
  key_policy: 'reuse_current' | 'create_if_missing' | 'rotate_on_renew'
  auto_create_key: boolean
  status: string
  store_visible: boolean
  sort_order: number
  created_at: string
  updated_at: string
}

export interface SalesMapping {
  id: number
  partner_id: number
  package_id: number
  external_package_code: string
  external_package_name: string
  sale_price: number
  currency: string
  status: string
  created_at: string
  updated_at: string
  partner?: SalesPartner | null
  package?: SalesPackage | null
}

export interface SalesUserSummary {
  id: number
  email: string
  username: string
  role: string
  status: string
  balance?: number
  concurrency?: number
}

export interface SalesAPIKeySummary {
  id: number
  user_id: number
  name: string
  key?: string
  group_id?: number | null
  status: string
  expires_at?: string | null
  created_at: string
  updated_at: string
}

export interface SalesOrder {
  id: number
  partner_id: number
  external_order_id: string
  external_user_id: string
  user_id?: number | null
  package_id: number
  order_type: string
  status: string
  subscription_id?: number | null
  api_key_id?: number | null
  amount: number
  currency: string
  package_snapshot: Record<string, unknown>
  raw_payload?: Record<string, unknown>
  result_snapshot?: Record<string, unknown>
  error_message: string
  fulfilled_at?: string | null
  created_at: string
  updated_at: string
  partner?: SalesPartner | null
  package?: SalesPackage | null
  user?: SalesUserSummary | null
}

export interface SalesBinding {
  id: number
  partner_id: number
  external_user_id: string
  user_id: number
  external_email: string
  external_name: string
  metadata: Record<string, unknown>
  created_at: string
  updated_at: string
  partner?: SalesPartner | null
  user?: SalesUserSummary | null
}

export interface SalesProvisionResult {
  order: SalesOrder
  user?: SalesUserSummary | null
  subscription?: any
  api_key?: SalesAPIKeySummary | null
  api_key_value?: string
  created_user: boolean
  created_binding: boolean
  reused_api_key: boolean
  subscription_already_applied: boolean
}

export interface SalesPaginated<T> {
  items: T[]
  total: number
  page: number
  page_size: number
  pages: number
}

export interface CreateSalesPartnerRequest {
  code?: string
  name: string
  description?: string
  status?: string
  rate_limit_rpm?: number
  secret?: string
}

export interface UpdateSalesPartnerRequest {
  name?: string
  description?: string
  status?: string
  rate_limit_rpm?: number
}

export interface CreateSalesPackageRequest {
  code: string
  name: string
  description?: string
  group_id: number
  cycle_unit?: 'day' | 'month'
  cycle_count?: number
  validity_days?: number
  key_policy?: 'reuse_current' | 'create_if_missing' | 'rotate_on_renew'
  auto_create_key?: boolean
  status?: string
  store_visible?: boolean
  sort_order?: number
}

export interface UpdateSalesPackageRequest extends Partial<CreateSalesPackageRequest> {}

export interface UpsertSalesMappingRequest {
  id?: number
  partner_id: number
  package_id: number
  external_package_code: string
  external_package_name?: string
  sale_price?: number
  currency?: string
  status?: string
}

export interface UpsertSalesBindingRequest {
  id?: number
  partner_id: number
  external_user_id: string
  user_id: number
  external_email?: string
  external_name?: string
  metadata?: Record<string, unknown>
}

export interface ProvisionSalesOrderRequest {
  partner_id: number
  external_order_id: string
  external_user_id?: string
  external_email?: string
  external_name?: string
  api_key?: string
  package_id?: number
  external_package_code?: string
  order_type?: 'purchase' | 'renewal' | 'manual'
  amount?: number
  currency?: string
  raw_payload?: Record<string, unknown>
}

export async function listPartners(
  page: number = 1,
  pageSize: number = 20,
  filters?: { status?: string; search?: string }
): Promise<SalesPaginated<SalesPartner>> {
  const { data } = await apiClient.get<SalesPaginated<SalesPartner>>('/admin/sales/partners', {
    params: { page, page_size: pageSize, ...filters }
  })
  return data
}

export async function createPartner(
  payload: CreateSalesPartnerRequest
): Promise<{ partner: SalesPartner; secret: string }> {
  const { data } = await apiClient.post<{ partner: SalesPartner; secret: string }>(
    '/admin/sales/partners',
    payload
  )
  return data
}

export async function updatePartner(
  id: number,
  payload: UpdateSalesPartnerRequest
): Promise<SalesPartner> {
  const { data } = await apiClient.put<SalesPartner>(`/admin/sales/partners/${id}`, payload)
  return data
}

export async function rotatePartnerSecret(
  id: number
): Promise<{ partner: SalesPartner; secret: string }> {
  const { data } = await apiClient.post<{ partner: SalesPartner; secret: string }>(
    `/admin/sales/partners/${id}/rotate-secret`
  )
  return data
}

export async function deletePartner(id: number): Promise<void> {
  await apiClient.delete(`/admin/sales/partners/${id}`)
}

export async function listPackages(
  page: number = 1,
  pageSize: number = 20,
  filters?: { status?: string; platform?: string; search?: string; store_visible?: boolean }
): Promise<SalesPaginated<SalesPackage>> {
  const { data } = await apiClient.get<SalesPaginated<SalesPackage>>('/admin/sales/packages', {
    params: { page, page_size: pageSize, ...filters }
  })
  return data
}

export async function createPackage(payload: CreateSalesPackageRequest): Promise<SalesPackage> {
  const { data } = await apiClient.post<SalesPackage>('/admin/sales/packages', payload)
  return data
}

export async function updatePackage(
  id: number,
  payload: UpdateSalesPackageRequest
): Promise<SalesPackage> {
  const { data } = await apiClient.put<SalesPackage>(`/admin/sales/packages/${id}`, payload)
  return data
}

export async function deletePackage(id: number): Promise<void> {
  await apiClient.delete(`/admin/sales/packages/${id}`)
}

export async function listMappings(
  page: number = 1,
  pageSize: number = 20,
  filters?: { status?: string; partner_id?: number; package_id?: number }
): Promise<SalesPaginated<SalesMapping>> {
  const { data } = await apiClient.get<SalesPaginated<SalesMapping>>('/admin/sales/mappings', {
    params: { page, page_size: pageSize, ...filters }
  })
  return data
}

export async function upsertMapping(payload: UpsertSalesMappingRequest): Promise<SalesMapping> {
  const { data } = await apiClient.post<SalesMapping>('/admin/sales/mappings', payload)
  return data
}

export async function deleteMapping(id: number): Promise<void> {
  await apiClient.delete(`/admin/sales/mappings/${id}`)
}

export async function listBindings(
  page: number = 1,
  pageSize: number = 20,
  filters?: { partner_id?: number; user_id?: number; search?: string }
): Promise<SalesPaginated<SalesBinding>> {
  const { data } = await apiClient.get<SalesPaginated<SalesBinding>>('/admin/sales/bindings', {
    params: { page, page_size: pageSize, ...filters }
  })
  return data
}

export async function getBinding(id: number): Promise<SalesBinding> {
  const { data } = await apiClient.get<SalesBinding>(`/admin/sales/bindings/${id}`)
  return data
}

export async function upsertBinding(payload: UpsertSalesBindingRequest): Promise<SalesBinding> {
  const { data } = await apiClient.post<SalesBinding>('/admin/sales/bindings', payload)
  return data
}

export async function deleteBinding(id: number): Promise<void> {
  await apiClient.delete(`/admin/sales/bindings/${id}`)
}

export async function listOrders(
  page: number = 1,
  pageSize: number = 20,
  filters?: {
    status?: string
    partner_id?: number
    package_id?: number
    user_id?: number
    search?: string
  }
): Promise<SalesPaginated<SalesOrder>> {
  const { data } = await apiClient.get<SalesPaginated<SalesOrder>>('/admin/sales/orders', {
    params: { page, page_size: pageSize, ...filters }
  })
  return data
}

export async function getOrder(id: number): Promise<SalesProvisionResult> {
  const { data } = await apiClient.get<SalesProvisionResult>(`/admin/sales/orders/${id}`)
  return data
}

export async function deleteOrder(id: number): Promise<void> {
  await apiClient.delete(`/admin/sales/orders/${id}`)
}

export async function provisionOrder(
  payload: ProvisionSalesOrderRequest
): Promise<SalesProvisionResult> {
  const { data } = await apiClient.post<SalesProvisionResult>(
    '/admin/sales/orders/provision',
    payload
  )
  return data
}

export async function batchDeleteOrders(payload: { ids: number[] }): Promise<{ deleted: number }> {
  const { data } = await apiClient.post<{ deleted: number }>('/admin/sales/orders/batch-delete', payload)
  return data
}

const salesAPI = {
  listPartners,
  createPartner,
  updatePartner,
  rotatePartnerSecret,
  deletePartner,
  listPackages,
  createPackage,
  updatePackage,
  deletePackage,
  listMappings,
  upsertMapping,
  deleteMapping,
  listBindings,
  getBinding,
  upsertBinding,
  deleteBinding,
  listOrders,
  getOrder,
  deleteOrder,
  batchDeleteOrders,
  provisionOrder
}

export default salesAPI
