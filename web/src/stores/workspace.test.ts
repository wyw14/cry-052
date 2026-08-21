import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useWorkspaceStore } from './workspace'

describe('workspace selection', () => {
  beforeEach(() => { setActivePinia(createPinia()); vi.restoreAllMocks() })
  it('tracks selected masking inputs', () => { const store = useWorkspaceStore(); store.selectSource('local'); store.selectPolicy('restricted'); expect(store.hasSelection).toBe(true) })
  it('keeps the batch filter explicit', () => { const store = useWorkspaceStore(); store.filterBatches('failed'); expect(store.batchFilter).toBe('failed') })

  it('loads governance pages from backend APIs', async () => {
    const responses: Record<string, unknown> = {
      '/api/v1/data-sources?size=100&sort=name': { items: [{ id: 'ds', name: '本地库', kind: 'local_sample', connection_reference: 'secret/local', status: 'ready', version: 2 }], page: 1, size: 100, total: 1 },
      '/api/v1/policies?size=100&sort=name': { items: [{ id: 'policy', group_id: 'group', name: '保护', version: 1, status: 'approved', scopes: ['sample.people'], strategies: [{ kind: 'mask', parameters: {} }], revision: 3, change_summary: '初始' }], page: 1, size: 100, total: 1 },
      '/api/v1/approvals?size=100&sort=created_at&filter_decision=pending': { items: [], page: 1, size: 100, total: 0 },
      '/api/v1/batches?size=100&sort=created_at': { items: [], page: 1, size: 100, total: 0 },
      '/api/v1/audit?size=100&sort=created_at': { items: [], page: 1, size: 100, total: 0 },
      '/api/v1/tables?data_source_id=ds&size=100&sort=qualified_name': { items: [{ id: 'table', data_source_id: 'ds', schema: 'sample', name: 'people', fields: [], fingerprint: 'fp', version: 1 }], page: 1, size: 100, total: 1 }
    }
    vi.stubGlobal('fetch', vi.fn(async (path: string) => new Response(JSON.stringify(responses[path]), { status: 200, headers: { 'Content-Type': 'application/json' } })))
    const store = useWorkspaceStore(); await store.loadWorkspace()
    expect(store.sources[0].id).toBe('ds'); expect(store.selectedPolicy).toBe('policy'); expect(store.tables[0].fingerprint).toBe('fp')
    expect(fetch).toHaveBeenCalledTimes(6)
  })

  it('runs and confirms a preview through public endpoints', async () => {
    const calls: string[] = []
    vi.stubGlobal('fetch', vi.fn(async (path: string) => { calls.push(path); const confirmed = path.includes('/confirm'); return new Response(JSON.stringify({ id: 'preview', input_fingerprint: 'fp', rows: [], conflicts: [], strategy_counters: {}, ...(confirmed ? { confirmed_at: '2026-08-21T00:00:00Z' } : {}) }), { status: 200, headers: { 'Content-Type': 'application/json' } }) }))
    const store = useWorkspaceStore(); store.selectedPolicy = 'policy'; await store.createPreview(); await store.confirmPreview()
    expect(calls).toEqual(['/api/v1/previews', '/api/v1/previews/preview/confirm']); expect(store.preview?.confirmed_at).toBeTruthy()
  })

  it('creates, submits, and approves a policy through role-bound sessions', async () => {
    const calls: Array<{ path: string; authorization: string }> = []
    vi.stubGlobal('fetch', vi.fn(async (path: string, init?: RequestInit) => {
      const headers = init?.headers as Record<string, string> | undefined
      calls.push({ path, authorization: headers?.Authorization || '' })
      if (path === '/api/v1/policies' && init?.method === 'POST') return new Response(JSON.stringify({ id: 'policy', status: 'draft' }), { status: 201, headers: { 'Content-Type': 'application/json' } })
      if (path.includes('/submit')) return new Response(JSON.stringify({ id: 'approval', policy_version_id: 'policy', decision: 'pending', revision: 1 }), { status: 201, headers: { 'Content-Type': 'application/json' } })
      if (path.includes('/decision')) return new Response(JSON.stringify({ id: 'policy', status: 'approved' }), { status: 200, headers: { 'Content-Type': 'application/json' } })
      if (path.includes('/approvals?')) return new Response(JSON.stringify({ items: [], page: 1, size: 100, total: 0 }), { status: 200, headers: { 'Content-Type': 'application/json' } })
      return new Response(JSON.stringify({ items: [{ id: 'policy', group_id: 'group', name: 'Protection', version: 1, status: 'review', scopes: ['sample.people'], strategies: [{ kind: 'mask', parameters: {} }], revision: 2, change_summary: 'initial' }], page: 1, size: 100, total: 1 }), { status: 200, headers: { 'Content-Type': 'application/json' } })
    }))
    const store = useWorkspaceStore()
    const policy = { id: 'policy', group_id: 'group', name: 'Protection', version: 1, status: 'draft', scopes: ['sample.people'], strategies: [{ kind: 'mask', parameters: {} }], revision: 1, change_summary: 'initial' }
    await store.createPolicy({ name: 'Protection', scope: 'sample.people', strategy: 'mask', changeSummary: 'initial' })
    await store.submitPolicy(policy)
    await store.decidePolicy({ ...policy, status: 'review', revision: 2 }, 'approved')
    expect(calls.map(item => item.path)).toContain('/api/v1/approvals/approval/decision')
    expect(calls.map(item => item.path)).toContain('/api/v1/policies')
    expect(calls.find(item => item.path.includes('/decision'))?.authorization).toBe('Bearer local-reviewer-session')
  })

  it('executes data source, catalog, batch recovery, rollback, and report workflows', async () => {
    const calls: string[] = []
    vi.stubGlobal('fetch', vi.fn(async (path: string) => {
      calls.push(path)
      if (path.includes('/report')) return new Response(JSON.stringify({ batch_id: 'batch', source_rows: 2, target_rows: 2, rejected_rows: 0, difference: 0, strategy_counters: { mask: 2 }, rollback_ready: true }), { status: 200, headers: { 'Content-Type': 'application/json' } })
      if (path.includes('/data-sources?')) return new Response(JSON.stringify({ items: [{ id: 'ds', name: 'Local', kind: 'local_sample', connection_reference: 'secret/local', status: 'ready', version: 2 }], page: 1, size: 100, total: 1 }), { status: 200, headers: { 'Content-Type': 'application/json' } })
      if (path.includes('/tables?')) return new Response(JSON.stringify({ items: [{ id: 'table', data_source_id: 'ds', schema: 'sample', name: 'people', fields: [{ name: 'mobile', data_type: 'text', sensitivity: 'restricted', category: 'phone', scopes: ['masked_export'] }], fingerprint: 'fp', version: 2 }], page: 1, size: 100, total: 1 }), { status: 200, headers: { 'Content-Type': 'application/json' } })
      if (path.includes('/batches?')) return new Response(JSON.stringify({ items: [], page: 1, size: 100, total: 0 }), { status: 200, headers: { 'Content-Type': 'application/json' } })
      return new Response(JSON.stringify({ id: 'ok' }), { status: 200, headers: { 'Content-Type': 'application/json' } })
    }))
    const store = useWorkspaceStore(); store.selectedSource = 'ds'
    const table = { id: 'table', data_source_id: 'ds', schema: 'sample', name: 'people', fields: [{ name: 'mobile', data_type: 'text', sensitivity: 'restricted', category: 'phone', scopes: ['masked_export'] }], fingerprint: 'fp', version: 1 }
    const batch = { id: 'batch', preview_id: 'preview', input_fingerprint: 'fp', status: 'failed' as const, progress: { read: 1, written: 1, rejected: 0 }, version: 4 }
    await store.createSource('Local')
    await store.updateClassification(table)
    await store.cancelBatch({ ...batch, status: 'running' })
    await store.recoverBatch(batch)
    await store.rollbackBatch(batch)
    await store.loadReport(batch.id)
    expect(store.report?.difference).toBe(0)
    expect(calls).toEqual(expect.arrayContaining(['/api/v1/data-sources', '/api/v1/tables/table/classification', '/api/v1/batches/batch/cancel', '/api/v1/batches/batch/recover', '/api/v1/batches/batch/rollback', '/api/v1/batches/batch/report?format=json']))
  })
})
