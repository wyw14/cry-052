import { defineStore } from 'pinia'
import { request, reviewerSessionToken, sessionToken, type Page } from '../services/api'

export type BatchState = 'pending' | 'running' | 'failed' | 'completed' | 'cancelled'
export type DataSource = { id: string; name: string; kind: string; connection_reference: string; status: string; version: number }
export type Field = { name: string; data_type: string; sensitivity: string; category: string; scopes: string[] }
export type TableSchema = { id: string; data_source_id: string; schema: string; name: string; fields: Field[]; fingerprint: string; version: number }
export type Strategy = { kind: string; parameters: Record<string, string> }
export type Policy = { id: string; group_id: string; name: string; version: number; status: string; scopes: string[]; strategies: Strategy[]; revision: number; change_summary: string }
export type Approval = { id: string; policy_version_id: string; requested_by: string; reviewer?: string; decision: string; revision: number; created_at: string }
export type Preview = { id: string; input_fingerprint: string; rows: Array<{ row_key: string; changes: Array<{ field: string; before: string; after: string; rule: string }> }>; conflicts: Array<{ field: string; code: string; message: string }>; strategy_counters: Record<string, number>; confirmed_at?: string }
export type Batch = { id: string; preview_id: string; input_fingerprint: string; status: BatchState | 'recovering' | 'cancelling' | 'rolled_back'; progress: { read: number; written: number; rejected: number; last_row_id?: string }; version: number; failure_code?: string }
export type AuditEvent = { id: string; actor: string; action: string; resource: string; outcome: string; request_id: string; created_at: string }
export type Report = { batch_id: string; source_rows: number; target_rows: number; rejected_rows: number; difference: number; strategy_counters: Record<string, number>; rollback_ready: boolean }

const mappings = [
  { source_field: 'customer_id', target_field: 'customer_id', strategies: [{ kind: 'keep', parameters: {} }] },
  { source_field: 'full_name', target_field: 'full_name', strategies: [{ kind: 'mask', parameters: { prefix: '1', suffix: '0' } }] },
  { source_field: 'mobile', target_field: 'mobile', strategies: [{ kind: 'mask', parameters: { prefix: '3', suffix: '4' } }] },
  { source_field: 'city', target_field: 'city', strategies: [{ kind: 'keep', parameters: {} }] }
]

export const useWorkspaceStore = defineStore('workspace', {
  state: () => ({
    selectedSource: '', selectedPolicy: '', batchFilter: 'all' as BatchState | 'all',
    sources: [] as DataSource[], tables: [] as TableSchema[], policies: [] as Policy[], batches: [] as Batch[], audits: [] as AuditEvent[],
    approvals: [] as Approval[], pendingApprovals: {} as Record<string, Approval>, preview: null as Preview | null, report: null as Report | null, loading: false, error: ''
  }),
  getters: {
    hasSelection: state => Boolean(state.selectedSource && state.selectedPolicy),
    filteredBatches: state => state.batchFilter === 'all' ? state.batches : state.batches.filter(item => item.status === state.batchFilter),
    activePolicies: state => state.policies.filter(item => item.status === 'approved')
  },
  actions: {
    selectSource(id: string) { this.selectedSource = id },
    selectPolicy(id: string) { this.selectedPolicy = id },
    filterBatches(state: BatchState | 'all') { this.batchFilter = state },
    async guard(work: () => Promise<void>) { this.loading = true; this.error = ''; try { await work() } catch (error) { this.error = error instanceof Error ? error.message : String(error) } finally { this.loading = false } },
    async loadSources() { await this.guard(async () => { const page = await request<Page<DataSource>>('/api/v1/data-sources?size=100&sort=name'); this.sources = page.items; if (!this.selectedSource && page.items[0]) this.selectedSource = page.items[0].id }) },
    async loadTables() { if (!this.selectedSource) return; await this.guard(async () => { const page = await request<Page<TableSchema>>(`/api/v1/tables?data_source_id=${encodeURIComponent(this.selectedSource)}&size=100&sort=qualified_name`); this.tables = page.items }) },
    async loadPolicies() { await this.guard(async () => { const page = await request<Page<Policy>>('/api/v1/policies?size=100&sort=name'); this.policies = page.items; const approved = page.items.find(item => item.status === 'approved'); if (!this.selectedPolicy && approved) this.selectedPolicy = approved.id }) },
    async loadApprovals() { await this.guard(async () => { const page = await request<Page<Approval>>('/api/v1/approvals?size=100&sort=created_at&filter_decision=pending', { headers: { Authorization: `Bearer ${reviewerSessionToken}` } }); this.approvals = page.items; this.pendingApprovals = Object.fromEntries(page.items.map(item => [item.policy_version_id, item])) }) },
    async loadBatches() { await this.guard(async () => { const page = await request<Page<Batch>>('/api/v1/batches?size=100&sort=created_at'); this.batches = page.items }) },
    async loadAudit() { await this.guard(async () => { const page = await request<Page<AuditEvent>>('/api/v1/audit?size=100&sort=created_at'); this.audits = page.items }) },
    async loadWorkspace() { await Promise.all([this.loadSources(), this.loadPolicies(), this.loadApprovals(), this.loadBatches(), this.loadAudit()]); await this.loadTables() },
    async createSource(name: string) { await this.guard(async () => { await request('/api/v1/data-sources', { method: 'POST', body: JSON.stringify({ name, kind: 'local_sample', connection_reference: `secret/${name}` }) }); await this.loadSources() }) },
    async updateClassification(table: TableSchema) { await this.guard(async () => { await request(`/api/v1/tables/${table.id}/classification`, { method: 'PATCH', body: JSON.stringify({ version: table.version, fields: table.fields }) }); await this.loadTables() }) },
    async createPolicy(input: { name: string; scope: string; strategy: string; changeSummary: string }) { await this.guard(async () => { await request('/api/v1/policies', { method: 'POST', body: JSON.stringify({ name: input.name, version: 1, scopes: [input.scope], strategies: [{ kind: input.strategy, parameters: {} }], change_summary: input.changeSummary }) }); await this.loadPolicies() }) },
    async submitPolicy(policy: Policy) { await this.guard(async () => { const approval = await request<Approval>(`/api/v1/policies/${policy.id}/submit`, { method: 'POST', body: JSON.stringify({ revision: policy.revision }) }); this.pendingApprovals[policy.id] = approval; await this.loadPolicies() }) },
    async decidePolicy(policy: Policy, decision: 'approved' | 'rejected') { const approval = this.pendingApprovals[policy.id]; if (!approval) return; await this.guard(async () => { await request(`/api/v1/approvals/${approval.id}/decision`, { method: 'POST', headers: { Authorization: `Bearer ${reviewerSessionToken}` }, body: JSON.stringify({ decision, reason: decision === 'rejected' ? 'policy scope needs revision' : '', revision: approval.revision }) }); delete this.pendingApprovals[policy.id]; await Promise.all([this.loadPolicies(), this.loadApprovals()]) }) },
    async createPreview() { await this.guard(async () => { this.preview = await request<Preview>('/api/v1/previews', { method: 'POST', body: JSON.stringify({ source_table_id: 'demo-customers', target_table_id: 'demo-customers-masked', policy_version_id: this.selectedPolicy || 'demo-policy-v1', mappings, limit: 20 }) }) }) },
    async confirmPreview() { if (!this.preview) return; await this.guard(async () => { this.preview = await request<Preview>(`/api/v1/previews/${this.preview!.id}/confirm`, { method: 'POST' }) }) },
    async createAndRunBatch() { if (!this.preview?.confirmed_at) return; await this.guard(async () => { const batch = await request<Batch>('/api/v1/batches', { method: 'POST', headers: { 'Idempotency-Key': `web-${this.preview!.id}` }, body: JSON.stringify({ preview_id: this.preview!.id }) }); await request(`/api/v1/batches/${batch.id}/run`, { method: 'POST', body: JSON.stringify({ mappings }) }); await this.loadBatches() }) },
    async cancelBatch(batch: Batch) { await this.guard(async () => { await request(`/api/v1/batches/${batch.id}/cancel`, { method: 'POST', body: JSON.stringify({ version: batch.version }) }); await this.loadBatches() }) },
    async recoverBatch(batch: Batch) { await this.guard(async () => { await request(`/api/v1/batches/${batch.id}/recover`, { method: 'POST', body: JSON.stringify({ version: batch.version, input_fingerprint: batch.input_fingerprint }) }); await this.loadBatches() }) },
    async rollbackBatch(batch: Batch) { await this.guard(async () => { await request(`/api/v1/batches/${batch.id}/rollback`, { method: 'POST' }); await this.loadBatches() }) },
    async loadReport(batchID: string) { await this.guard(async () => { this.report = await request<Report>(`/api/v1/batches/${batchID}/report?format=json`) }) },
    async exportReport(batchID: string) { const response = await fetch(`/api/v1/batches/${batchID}/report?format=csv`, { headers: { Authorization: `Bearer ${sessionToken}` } }); const blob = await response.blob(); const url = URL.createObjectURL(blob); const link = document.createElement('a'); link.href = url; link.download = `${batchID}.csv`; link.click(); URL.revokeObjectURL(url) }
  }
})
