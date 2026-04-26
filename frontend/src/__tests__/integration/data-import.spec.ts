import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import ImportDataModal from '@/components/admin/account/ImportDataModal.vue'

const showError = vi.fn()
const showSuccess = vi.fn()

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess
  })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      list: vi.fn(),
      importData: vi.fn(),
      importDataUpload: vi.fn()
    }
  }
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

describe('ImportDataModal', () => {
  beforeEach(async () => {
    showError.mockReset()
    showSuccess.mockReset()
    const { adminAPI } = await import('@/api/admin')
    vi.mocked(adminAPI.accounts.list).mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 100,
      pages: 0
    } as any)
    vi.mocked(adminAPI.accounts.importData).mockReset()
    vi.mocked(adminAPI.accounts.importDataUpload).mockReset()
  })

  it('未选择文件时提示错误', async () => {
    const wrapper = mount(ImportDataModal, {
      props: { show: true },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' }
        }
      }
    })

    await wrapper.find('form').trigger('submit')
    expect(showError).toHaveBeenCalledWith('admin.accounts.dataImportSelectFile')
  })

  it('无效 JSON 时提示解析失败', async () => {
    const wrapper = mount(ImportDataModal, {
      props: { show: true },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' }
        }
      }
    })

    const input = wrapper.find('input[type="file"]')
    const file = new File(['invalid json'], 'data.json', { type: 'application/json' })
    Object.defineProperty(file, 'text', {
      value: () => Promise.resolve('invalid json')
    })
    Object.defineProperty(input.element, 'files', {
      value: [file]
    })

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await Promise.resolve()

    expect(showError).toHaveBeenCalledWith('admin.accounts.dataImportParseFailed')
  })

  it('ZIP 文件时调用上传导入接口并透传域名过滤', async () => {
    const { adminAPI } = await import('@/api/admin')
    vi.mocked(adminAPI.accounts.list).mockResolvedValue({
      items: [
        {
          id: 123,
          name: 'template@outlook.com',
          groups: [{ id: 2, name: 'CODEX' }]
        }
      ] as any,
      total: 1,
      page: 1,
      page_size: 100,
      pages: 1
    })
    vi.mocked(adminAPI.accounts.importDataUpload).mockResolvedValue({
      proxy_created: 0,
      proxy_reused: 0,
      proxy_failed: 0,
      account_created: 2,
      account_skipped_existing: 1,
      account_failed: 0
    })

    const wrapper = mount(ImportDataModal, {
      props: { show: false },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' }
        }
      }
    })
    await wrapper.setProps({ show: true })
    await Promise.resolve()
    await Promise.resolve()

    await wrapper.find('input[type="text"]').setValue('outlook.com, gmail.com')
    await wrapper.find('select').setValue('123')

    const input = wrapper.find('input[type="file"]')
    const file = new File(['zip'], 'data.zip', { type: 'application/zip' })
    Object.defineProperty(input.element, 'files', {
      value: [file]
    })

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await Promise.resolve()

    expect(adminAPI.accounts.importDataUpload).toHaveBeenCalledWith({
      file,
      skip_default_group_bind: true,
      include_email_domains: ['outlook.com', 'gmail.com'],
      template_account_id: 123
    })
  })

  it('CPA JSON 文件时调用上传导入接口并透传模板账号', async () => {
    const { adminAPI } = await import('@/api/admin')
    vi.mocked(adminAPI.accounts.list).mockResolvedValue({
      items: [
        {
          id: 456,
          name: 'template@outlook.com',
          groups: [{ id: 2, name: 'CODEX' }]
        }
      ] as any,
      total: 1,
      page: 1,
      page_size: 100,
      pages: 1
    })
    vi.mocked(adminAPI.accounts.importDataUpload).mockResolvedValue({
      proxy_created: 0,
      proxy_reused: 0,
      proxy_failed: 0,
      account_created: 1,
      account_skipped_existing: 0,
      account_failed: 0
    })

    const wrapper = mount(ImportDataModal, {
      props: { show: false },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' }
        }
      }
    })
    await wrapper.setProps({ show: true })
    await Promise.resolve()
    await Promise.resolve()

    await wrapper.find('input[type="text"]').setValue('outlook.com')
    await wrapper.find('select').setValue('456')

    const input = wrapper.find('input[type="file"]')
    const file = new File(
      ['{"email":"demo@outlook.com","access_token":"at","refresh_token":"rt"}'],
      'oauth.json',
      { type: 'application/json' }
    )
    Object.defineProperty(file, 'text', {
      value: () => Promise.resolve('{"email":"demo@outlook.com","access_token":"at","refresh_token":"rt"}')
    })
    Object.defineProperty(input.element, 'files', {
      value: [file]
    })

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await Promise.resolve()

    expect(adminAPI.accounts.importDataUpload).toHaveBeenCalledWith({
      file,
      skip_default_group_bind: true,
      include_email_domains: ['outlook.com'],
      template_account_id: 456
    })
    expect(adminAPI.accounts.importData).not.toHaveBeenCalled()
  })
})
