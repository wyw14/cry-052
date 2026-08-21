import { computed, defineComponent, h, onMounted } from 'vue'
import StatusPill from './StatusPill.vue'
import { useWorkspaceStore } from '../stores/workspace'

export default defineComponent({
  name: 'AuditWorkspace',
  setup() {
    const store = useWorkspaceStore()
    const outcomeCounts = computed(() => store.audits.reduce<Record<string, number>>((counts, event) => {
      counts[event.outcome] = (counts[event.outcome] ?? 0) + 1
      return counts
    }, {}))
    onMounted(() => store.loadAudit())

    const auditRow = (event: typeof store.audits[number]) => h('tr', { key: event.id }, [
      h('td', new Date(event.created_at).toLocaleString()),
      h('td', event.actor),
      h('td', [h('code', event.action)]),
      h('td', event.resource),
      h('td', [h(StatusPill, { tone: event.outcome === 'success' ? 'green' : event.outcome === 'intent' ? 'amber' : 'red', label: event.outcome })]),
      h('td', [h('code', event.request_id)])
    ])

    return () => h('div', { class: 'page' }, [
      h('div', { class: 'page-heading' }, [
        h('div', [h('p', { class: 'eyebrow' }, '可追溯操作'), h('h1', '审计日志'), h('p', '敏感参数已清洗，保留请求标识和业务结果。')]),
        h('button', { class: 'secondary', type: 'button', onClick: () => store.loadAudit() }, '刷新')
      ]),
      h('div', { class: 'metrics' }, [
        h('div', [h('small', '当前事件'), h('strong', String(store.audits.length)), h('span', '按时间顺序展示')]),
        h('div', [h('small', '审计意图'), h('strong', String(outcomeCounts.value.intent ?? 0)), h('span', '副作用开始前持久化')]),
        h('div', [h('small', '成功结果'), h('strong', String(outcomeCounts.value.success ?? 0)), h('span', '可按 request_id 追踪')]),
        h('div', [h('small', '失败结果'), h('strong', String(outcomeCounts.value.failed ?? 0)), h('span', '动态错误内容已清洗')])
      ]),
      store.error ? h('p', { class: 'error' }, store.error) : null,
      h('section', { class: 'section' }, [h('div', { class: 'table-wrap' }, [
        h('table', [
          h('thead', [h('tr', ['时间', '操作人', '动作', '资源', '结果', 'request_id'].map(label => h('th', label)))]),
          h('tbody', store.audits.map(auditRow))
        ])
      ])])
    ])
  }
})
