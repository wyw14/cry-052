import { defineComponent, h, onMounted } from 'vue'
import { Pause, RefreshCw, RotateCcw, Undo2 } from 'lucide-vue-next'
import StatusPill from './StatusPill.vue'
import { useWorkspaceStore } from '../stores/workspace'

const batchTone = (status: string) => status === 'completed' ? 'green' : status === 'failed' ? 'red' : status === 'running' ? 'amber' : 'gray'

export default defineComponent({
  name: 'BatchWorkspace',
  setup() {
    const store = useWorkspaceStore()
    onMounted(() => store.loadBatches())
    const actionButton = (title: string, icon: typeof Pause, action: () => unknown) => h('button', { class: 'icon-button', title, type: 'button', onClick: action }, [h(icon, { size: 16 })])
    const row = (batch: typeof store.batches[number]) => h('tr', { key: batch.id }, [
      h('td', [h('strong', batch.id), h('small', batch.preview_id)]),
      h('td', [h('code', batch.input_fingerprint)]),
      h('td', batch.progress.last_row_id || '未开始'),
      h('td', `${batch.progress.written} / ${batch.progress.read}`),
      h('td', [h(StatusPill, { tone: batchTone(batch.status), label: batch.status })]),
      h('td', [
        batch.status === 'running' ? actionButton('取消', Pause, () => store.cancelBatch(batch)) : null,
        batch.status === 'failed' ? actionButton('恢复', RotateCcw, () => store.recoverBatch(batch)) : null,
        ['completed', 'failed', 'cancelled'].includes(batch.status) ? actionButton('回滚', Undo2, () => store.rollbackBatch(batch)) : null
      ])
    ])
    return () => h('div', { class: 'page' }, [
      h('div', { class: 'page-heading' }, [
        h('div', [h('p', { class: 'eyebrow' }, '批次执行'), h('h1', '执行任务'), h('p', '监控真实进度、取消、失败恢复和目标表回滚。')]),
        h('button', { class: 'icon-button', title: '刷新', type: 'button', onClick: () => store.loadBatches() }, [h(RefreshCw, { size: 17 })])
      ]),
      store.error ? h('p', { class: 'error' }, store.error) : null,
      h('div', { class: 'tabs' }, [
        h('button', { class: { active: store.batchFilter === 'all' }, onClick: () => store.filterBatches('all') }, `全部 ${store.batches.length}`),
        h('button', { onClick: () => store.filterBatches('running') }, '执行中'),
        h('button', { onClick: () => store.filterBatches('failed') }, '待恢复'),
        h('button', { onClick: () => store.filterBatches('completed') }, '已完成')
      ]),
      h('section', { class: 'section' }, [h('div', { class: 'table-wrap' }, [h('table', [
        h('thead', [h('tr', ['批次', '输入快照', '进度', '行数', '状态', '操作'].map(label => h('th', label)))]),
        h('tbody', store.filteredBatches.map(row))
      ])])])
    ])
  }
})
