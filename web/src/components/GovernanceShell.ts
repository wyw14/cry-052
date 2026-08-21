import { computed, defineComponent, h } from 'vue'
import { RouterLink, RouterView, useRoute } from 'vue-router'
import { Bell, Database, FileBarChart, GitCompare, LayoutDashboard, PlayCircle, ScanSearch, ScrollText, Search, ShieldCheck, Tags } from 'lucide-vue-next'

const navigation = [
  { path: '/', label: '总览', icon: LayoutDashboard },
  { path: '/sources', label: '数据源', icon: Database },
  { path: '/catalog', label: '字段分类', icon: Tags },
  { path: '/policies', label: '策略库', icon: ShieldCheck },
  { path: '/preview', label: '预演对比', icon: ScanSearch },
  { path: '/batches', label: '执行任务', icon: PlayCircle },
  { path: '/reports', label: '报告', icon: FileBarChart },
  { path: '/audit', label: '审计', icon: ScrollText },
  { path: '/versions', label: '版本', icon: GitCompare }
] as const

export default defineComponent({
  name: 'GovernanceShell',
  setup() {
    const route = useRoute()
    const currentSection = computed(() => navigation.find(item => item.path === route.path)?.label ?? '治理工作台')
    return () => h('div', { class: 'shell' }, [
      h('aside', { class: 'sidebar' }, [
        h('div', { class: 'brand' }, [
          h('div', { class: 'brand-mark' }, '净'),
          h('div', [h('strong', '净界'), h('small', '离线脱敏治理')])
        ]),
        h('nav', navigation.map(item => h(RouterLink, { key: item.path, to: item.path }, {
          default: () => [h(item.icon, { size: 17 }), h('span', item.label)]
        }))),
        h('div', { class: 'account' }, [
          h('span', { class: 'avatar' }, '管'),
          h('div', [h('strong', '数据管理员'), h('small', '本地服务账户')])
        ])
      ]),
      h('main', [
        h('header', { class: 'topbar' }, [
          h('div', { class: 'search' }, [
            h(Search, { size: 16 }),
            h('input', { 'aria-label': `${currentSection.value}内搜索`, placeholder: '搜索数据源、策略或批次' })
          ]),
          h('div', { class: 'top-actions' }, [
            h('span', { class: 'offline' }, '离线环境'),
            h('button', { class: 'icon-button', title: '通知', type: 'button' }, [h(Bell, { size: 18 })])
          ])
        ]),
        h('section', { class: 'workspace' }, [h(RouterView)])
      ])
    ])
  }
})
