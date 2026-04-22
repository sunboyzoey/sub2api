<template>
  <div class="relative min-h-screen overflow-hidden bg-gray-50 dark:bg-dark-950">
    <div class="pointer-events-none absolute inset-0 bg-mesh-gradient opacity-80"></div>
    <div class="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top,rgba(59,130,246,0.12),transparent_34%),radial-gradient(circle_at_bottom_right,rgba(16,185,129,0.12),transparent_28%)]"></div>

    <header class="relative px-4 pt-4 md:px-6 md:pt-6">
      <div class="mx-auto flex max-w-7xl flex-col gap-4 rounded-3xl border border-white/60 bg-white/85 px-5 py-5 shadow-[0_24px_80px_-40px_rgba(15,23,42,0.55)] backdrop-blur dark:border-white/10 dark:bg-slate-950/75 md:flex-row md:items-center md:justify-between">
        <router-link to="/home" class="flex items-center gap-3">
          <div class="flex h-11 w-11 items-center justify-center overflow-hidden rounded-2xl bg-white shadow-md dark:bg-slate-900">
            <img :src="siteLogo" alt="Logo" class="h-full w-full object-contain" />
          </div>
          <div>
            <p class="text-xs font-semibold uppercase tracking-[0.24em] text-sky-600 dark:text-sky-300">Check CX</p>
            <h1 class="text-lg font-semibold text-slate-900 dark:text-white">账号池监控</h1>
          </div>
        </router-link>

        <div class="flex flex-wrap items-center gap-2">
          <span class="inline-flex items-center gap-2 rounded-full border border-emerald-200 bg-emerald-50 px-3 py-2 text-xs font-medium text-emerald-700 dark:border-emerald-900/60 dark:bg-emerald-950/30 dark:text-emerald-200">
            <span class="h-2 w-2 rounded-full bg-emerald-500"></span>
            已接入 Check CX
          </span>
          <button
            @click="reloadFrame"
            class="rounded-2xl border border-slate-200 bg-white px-3 py-2 text-sm text-slate-600 transition hover:border-sky-300 hover:text-sky-700 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-300 dark:hover:border-sky-500 dark:hover:text-sky-200"
          >
            刷新监控
          </button>
          <a
            :href="monitorUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="rounded-2xl border border-slate-200 bg-white px-3 py-2 text-sm text-slate-600 transition hover:border-sky-300 hover:text-sky-700 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-300 dark:hover:border-sky-500 dark:hover:text-sky-200"
          >
            新窗口打开
          </a>
          <button
            @click="toggleTheme"
            class="rounded-2xl border border-slate-200 bg-white p-2.5 text-slate-600 transition hover:border-sky-300 hover:text-sky-700 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-300 dark:hover:border-sky-500 dark:hover:text-sky-200"
            :title="isDark ? '切换浅色模式' : '切换深色模式'"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>
        </div>
      </div>
    </header>

    <main class="relative px-4 pb-6 pt-5 md:px-6 md:pb-8 md:pt-6">
      <div class="mx-auto max-w-7xl space-y-4">
        <section class="rounded-[28px] border border-white/60 bg-white/85 p-5 shadow-[0_24px_70px_-38px_rgba(15,23,42,0.5)] backdrop-blur dark:border-white/10 dark:bg-slate-950/72">
          <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <h2 class="text-2xl font-semibold tracking-tight text-slate-900 dark:text-white md:text-3xl">
                使用 Check CX 统一查看账号池健康状态
              </h2>
              <p class="mt-2 max-w-3xl text-sm leading-6 text-slate-600 dark:text-slate-300">
                原有的 Sub2 账号池统计页已替换为 Check CX。当前页面会直接加载 `check-cx`
                面板，用于查看可用性、延迟、分组状态和历史趋势。
              </p>
            </div>
            <div class="rounded-2xl border border-slate-200/80 bg-slate-50/80 px-4 py-3 text-sm text-slate-600 dark:border-slate-800 dark:bg-slate-900/80 dark:text-slate-300">
              监控地址：<span class="font-mono text-xs text-slate-700 dark:text-slate-200">{{ monitorPath }}</span>
            </div>
          </div>
        </section>

        <section class="relative overflow-hidden rounded-[32px] border border-white/60 bg-white/85 shadow-[0_28px_90px_-42px_rgba(15,23,42,0.6)] backdrop-blur dark:border-white/10 dark:bg-slate-950/78">
          <div class="flex items-center justify-between border-b border-slate-200/80 px-4 py-3 text-sm text-slate-500 dark:border-slate-800 dark:text-slate-400">
            <div class="flex items-center gap-2">
              <span class="inline-block h-2.5 w-2.5 rounded-full" :class="frameLoaded ? 'bg-emerald-500' : 'bg-amber-500'"></span>
              <span>{{ frameLoaded ? '监控面板已加载' : '正在加载监控面板' }}</span>
            </div>
            <span>来源：Check CX</span>
          </div>

          <div
            v-if="!frameLoaded"
            class="absolute inset-x-0 top-[49px] z-10 flex h-[calc(100vh-250px)] min-h-[720px] items-center justify-center bg-white/65 backdrop-blur-sm dark:bg-slate-950/55"
          >
            <div class="flex flex-col items-center gap-4 text-center">
              <div class="h-10 w-10 animate-spin rounded-full border-2 border-sky-500 border-t-transparent"></div>
              <div>
                <p class="text-base font-medium text-slate-900 dark:text-white">Check CX 加载中</p>
                <p class="mt-1 text-sm text-slate-500 dark:text-slate-400">首次打开可能需要几秒钟，请稍候。</p>
              </div>
            </div>
          </div>

          <iframe
            :key="frameKey"
            :src="monitorUrl"
            class="block h-[calc(100vh-250px)] min-h-[720px] w-full border-0 bg-transparent"
            allowfullscreen
            @load="handleFrameLoad"
          ></iframe>
        </section>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useAppStore } from '@/stores'
import Icon from '@/components/icons/Icon.vue'

const appStore = useAppStore()

const isDark = ref(false)
const frameLoaded = ref(false)
const frameKey = ref(0)

const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || '/logo.png')
const monitorPath = '/check-cx'
const monitorUrl = computed(() => {
  if (typeof window === 'undefined') return monitorPath
  return new URL(monitorPath, window.location.origin).toString()
})

function handleFrameLoad() {
  frameLoaded.value = true
}

function reloadFrame() {
  frameLoaded.value = false
  frameKey.value += 1
}

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

onMounted(async () => {
  const savedTheme = localStorage.getItem('theme')
  if (
    savedTheme === 'dark' ||
    (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)
  ) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }

  if (!appStore.publicSettingsLoaded) {
    await appStore.fetchPublicSettings()
  }
})
</script>
