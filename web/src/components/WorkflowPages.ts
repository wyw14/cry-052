import { computed, defineComponent, h, onMounted, ref, type VNodeChild } from 'vue'
import { RouterLink } from 'vue-router'
import { ArrowRight, Check, Database, Download, GitCommit, Play, Plus, Search } from 'lucide-vue-next'
import StatusPill from './StatusPill.vue'
import { useWorkspaceStore } from '../stores/workspace'

const heading = (eyebrow: string, title: string, description: string, action?: VNodeChild) => h('div', { class: 'page-heading' }, [h('div', [h('p', { class: 'eyebrow' }, eyebrow), h('h1', title), h('p', description)]), action])
const metric = (label: string, value: string | number, detail = '') => h('div', [h('small', label), h('strong', String(value)), detail ? h('span', detail) : null])

export const DashboardPage = defineComponent({ name: 'DashboardPage', setup() {
  const store = useWorkspaceStore(); onMounted(() => store.loadWorkspace())
  const completed = computed(() => store.batches.filter(item => item.status === 'completed').length)
  return () => h('div', { class: 'page' }, [
    heading('治理总览', '离线脱敏工作台', '本地样例数据与生产批次实时状态', h(RouterLink, { class: 'primary', to: '/preview' }, { default: () => ['新建预演', h(ArrowRight, { size: 16 })] })),
    store.error ? h('p', { class: 'error' }, store.error) : null,
    h('div', { class: 'metrics' }, [metric('登记数据源', store.sources.length, `${store.sources.filter(item => item.status === 'ready').length} 个已就绪`), metric('目录字段', store.tables.reduce((sum, item) => sum + item.fields.length, 0), '本地元数据目录'), metric('生效策略', store.activePolicies.length, `${store.policies.filter(item => item.status === 'review').length} 个待审批`), metric('执行批次', store.batches.length, `${completed.value} 个已完成`)]),
    h('section', { class: 'section' }, [h('div', { class: 'section-title' }, [h('h2', '执行态势'), h(RouterLink, { to: '/batches' }, { default: () => '查看全部' })]), h('div', { class: 'coverage' }, store.batches.slice(0, 5).map(batch => h('div', { key: batch.id }, [h('span', `${batch.id} · ${batch.status}`), h('strong', `${batch.progress.written}/${batch.progress.read}`), h('i', { style: { width: `${batch.progress.read ? Math.round(batch.progress.written / batch.progress.read * 100) : 0}%` } })])))])
  ])
} })

export const DataSourcesPage = defineComponent({ name: 'DataSourcesPage', setup() {
  const store = useWorkspaceStore(); const name = ref(''); onMounted(() => store.loadSources())
  const create = async () => { const value = name.value.trim(); if (value) { await store.createSource(value); name.value = '' } }
  return () => h('div', { class: 'page' }, [heading('连接管理', '数据源', '仅登记本地连接引用，明文参数不会进入页面或日志.', h('div', { class: 'toolbar' }, [h('input', { value: name.value, placeholder: '新数据源名称', onInput: (event: Event) => { name.value = (event.target as HTMLInputElement).value } }), h('button', { class: 'primary', onClick: create }, [h(Plus, { size: 16 }), '登记'])])), h('section', { class: 'section' }, store.sources.map(row => h('div', { class: 'item-card', key: row.id }, [h(Database, { size: 18 }), h('h2', row.name), h('p', row.connection_reference), h(StatusPill, { tone: row.status === 'ready' ? 'green' : 'gray', label: row.status })])))])
} })

export const PreviewPage = defineComponent({ name: 'PreviewPage', setup() {
  const store = useWorkspaceStore(); onMounted(() => store.loadPolicies())
  return () => h('div', { class: 'page' }, [heading('生产前置', '预演对比', '确认字段映射与策略结果后，才能创建生产批次。', h('button', { class: 'primary', disabled: store.loading, onClick: () => store.createPreview() }, [h(Play, { size: 16 }), '运行预演'])), h('div', { class: 'preview-grid' }, [h('section', { class: 'section' }, [h('h2', '预演设置'), h('p', 'sample.customers → sample.customers_masked'), h('select', { value: store.selectedPolicy, onChange: (event: Event) => { store.selectedPolicy = (event.target as HTMLSelectElement).value } }, store.activePolicies.map(policy => h('option', { value: policy.id }, `${policy.name} v${policy.version}`)))]), h('section', { class: 'section' }, [h('div', { class: 'section-title' }, [h('h2', '样例变化'), h('span', { class: 'tag' }, `${store.preview?.rows.length ?? 0} 行`)]), ...(store.preview?.rows.flatMap(row => row.changes).map(change => h('div', { class: 'compare', key: change.field + change.before }, [h('code', change.before), h(ArrowRight, { size: 16 }), h('code', change.after)])) ?? []), store.preview ? h('button', { class: 'primary', onClick: () => store.preview?.confirmed_at ? store.createAndRunBatch() : store.confirmPreview() }, [h(Check, { size: 16 }), store.preview.confirmed_at ? '创建并执行' : '确认预演']) : null])])])
} })

export const ReportsPage = defineComponent({ name: 'ReportsPage', setup() {
  const store = useWorkspaceStore(); const selected = ref(''); onMounted(async () => { await store.loadBatches(); const batch = store.batches.find(item => item.status === 'completed'); if (batch) { selected.value = batch.id; await store.loadReport(batch.id) } })
  return () => h('div', { class: 'page' }, [heading('执行证据', '报告', '查看真实行数对账、差异摘要与本地导出。', h('div', { class: 'toolbar' }, [h('select', { value: selected.value, onChange: (event: Event) => { selected.value = (event.target as HTMLSelectElement).value } }, store.batches.map(batch => h('option', { value: batch.id }, batch.id))), h('button', { class: 'icon-button', title: '加载', onClick: () => selected.value && store.loadReport(selected.value) }, [h(Search, { size: 16 })]), h('button', { class: 'primary', disabled: !selected.value, onClick: () => store.exportReport(selected.value) }, [h(Download, { size: 16 }), '导出 CSV'])])), store.report ? h('div', { class: 'metrics' }, [metric('源行数', store.report.source_rows), metric('目标写入', store.report.target_rows), metric('拒绝行', store.report.rejected_rows), metric('未解释差异', store.report.difference)]) : null])
} })

export const VersionsPage = defineComponent({ name: 'VersionsPage', setup() {
  const store = useWorkspaceStore(); const group = ref(''); onMounted(async () => { await store.loadPolicies(); group.value = store.policies[0]?.group_id ?? '' }); const versions = computed(() => store.policies.filter(item => item.group_id === group.value).sort((left, right) => right.version - left.version))
  return () => h('div', { class: 'page' }, [heading('策略历史', '版本', '比较后端保存的变更摘要、审批状态和生效范围。'), h('div', { class: 'split' }, [h('aside', { class: 'subnav' }, [h('strong', '策略分组'), ...store.policies.map(policy => h('button', { class: { active: policy.group_id === group.value }, onClick: () => { group.value = policy.group_id } }, [policy.name, h('small', policy.group_id)]))]), versions.value[0] ? h('section', { class: 'section' }, [h('div', { class: 'section-title' }, [h('h2', `v${versions.value[0].version} 变更`), h('span', { class: 'tag' }, versions.value[0].status)]), h('ul', { class: 'timeline' }, versions.value.map(version => h('li', { key: version.id }, [h(GitCommit, { size: 17 }), h('div', [h('strong', `v${version.version} · ${version.status}`), h('small', `${version.change_summary} · ${version.scopes.join(' · ')}`)])])))]) : null])])
} })
