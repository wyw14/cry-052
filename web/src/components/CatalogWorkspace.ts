import { computed, defineComponent, h, onMounted, ref } from 'vue'
import { Save } from 'lucide-vue-next'
import StatusPill from './StatusPill.vue'
import { useWorkspaceStore, type TableSchema } from '../stores/workspace'

export default defineComponent({
  name: 'CatalogWorkspace',
  setup() {
    const store = useWorkspaceStore()
    const selectedID = ref('')
    const selected = computed(() => store.tables.find(table => table.id === selectedID.value) ?? store.tables[0])
    onMounted(async () => {
      await Promise.all([store.loadSources(), store.loadTables()])
      selectedID.value = store.tables[0]?.id ?? ''
    })
    const scopeInput = (table: TableSchema, index: number) => h('input', {
      value: table.fields[index]?.scopes.join(',') ?? '',
      onInput: (event: Event) => {
        const field = table.fields[index]
        if (field) field.scopes = (event.target as HTMLInputElement).value.split(',').map(item => item.trim()).filter(Boolean)
      }
    })
    const fieldRow = (table: TableSchema, index: number) => {
      const field = table.fields[index]!
      return h('tr', { key: field.name }, [
        h('td', [h('code', field.name)]), h('td', field.data_type),
        h('td', [h('select', { value: field.sensitivity, onChange: (event: Event) => { field.sensitivity = (event.target as HTMLSelectElement).value } }, [
          h('option', { value: 'public' }, '公开'), h('option', { value: 'internal' }, '内部'), h('option', { value: 'confidential' }, '机密'), h('option', { value: 'restricted' }, '受限')
        ])]),
        h('td', [h('input', { value: field.category, onInput: (event: Event) => { field.category = (event.target as HTMLInputElement).value } })]),
        h('td', [scopeInput(table, index)])
      ])
    }
    return () => h('div', { class: 'page' }, [
      h('div', { class: 'page-heading' }, [h('div', [h('p', { class: 'eyebrow' }, '元数据目录'), h('h1', '字段分类'), h('p', '维护敏感级别、数据分类和允许访问的作用域。')])]),
      store.error ? h('p', { class: 'error' }, store.error) : null,
      h('div', { class: 'split' }, [
        h('aside', { class: 'subnav' }, [h('strong', '本地目录'), ...store.tables.map(table => h('button', { key: table.id, class: { active: table.id === selected.value?.id }, onClick: () => { selectedID.value = table.id } }, [table.schema + '.' + table.name, h('small', `${table.fields.length} 个字段`)]))]),
        selected.value ? h('section', { class: 'section' }, [
          h('div', { class: 'section-title' }, [h('h2', `${selected.value.schema}.${selected.value.name}`), h(StatusPill, { tone: 'green', label: `目录版本 ${selected.value.version}` })]),
          h('div', { class: 'table-wrap' }, [h('table', [h('thead', [h('tr', ['字段', '类型', '敏感级别', '分类', '访问范围'].map(label => h('th', label)))]), h('tbody', selected.value.fields.map((_field, index) => fieldRow(selected.value!, index)))])]),
          h('div', { class: 'form-actions' }, [h('button', { class: 'primary', disabled: store.loading, onClick: () => store.updateClassification(selected.value!) }, [h(Save, { size: 16 }), '保存分类'])])
        ]) : null
      ])
    ])
  }
})
