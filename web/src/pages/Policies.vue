<script setup lang="ts">
import { onMounted, reactive } from 'vue'
import { Check, Plus, RefreshCw, Send, X } from 'lucide-vue-next'
import StatusPill from '../components/StatusPill.vue'
import { useWorkspaceStore } from '../stores/workspace'

const store = useWorkspaceStore()
const draft = reactive({ name: '', scope: 'sample.customers', strategy: 'mask', changeSummary: '' })
onMounted(() => Promise.all([store.loadPolicies(), store.loadApprovals()]))

async function createPolicy() {
  if (!draft.name || !draft.scope || !draft.changeSummary) return
  await store.createPolicy(draft)
  draft.name = ''
  draft.changeSummary = ''
}
</script>

<template>
  <div class="page">
    <div class="page-heading">
      <div><p class="eyebrow">规则配置</p><h1>策略库</h1><p>管理策略组合、作用范围、版本提交和复核决定。</p></div>
      <button class="icon-button" title="刷新" @click="store.loadPolicies"><RefreshCw :size="17" /></button>
    </div>
    <p v-if="store.error" class="error">{{ store.error }}</p>
    <section class="section policy-editor">
      <div class="section-title"><h2>新建策略草稿</h2></div>
      <div class="form-grid">
        <label>名称<input v-model="draft.name" placeholder="客户联系方式保护" /></label>
        <label>作用域<input v-model="draft.scope" placeholder="sample.customers" /></label>
        <label>策略<select v-model="draft.strategy"><option value="mask">掩码</option><option value="replace">替换</option><option value="generalize">泛化</option><option value="hash">哈希</option><option value="keep">保留</option></select></label>
        <label>变更说明<input v-model="draft.changeSummary" placeholder="说明本版本的治理目的" /></label>
      </div>
      <button class="primary" :disabled="store.loading" @click="createPolicy"><Plus :size="16" />创建草稿</button>
    </section>
    <div class="item-grid">
      <article v-for="policy in store.policies" :key="policy.id" class="item-card">
        <div class="item-card-head"><span class="policy-symbol">策</span><StatusPill :tone="policy.status === 'approved' ? 'green' : policy.status === 'review' ? 'amber' : 'gray'" :label="policy.status" /></div>
        <h2>{{ policy.name }}</h2><p>{{ policy.strategies.map(item => item.kind).join(' / ') }}</p>
        <div class="meta"><span>v{{ policy.version }} · r{{ policy.revision }}</span><span>{{ policy.scopes.length }} 个作用域</span></div>
        <div class="card-actions">
          <button v-if="policy.status === 'draft'" class="secondary" @click="store.submitPolicy(policy)"><Send :size="15" />提交复核</button>
          <template v-else-if="policy.status === 'review' && store.pendingApprovals[policy.id]">
            <button class="secondary" @click="store.decidePolicy(policy, 'approved')"><Check :size="15" />批准</button>
            <button class="secondary danger" @click="store.decidePolicy(policy, 'rejected')"><X :size="15" />驳回</button>
          </template>
          <button v-else class="secondary" @click="store.selectPolicy(policy.id)">用于预演</button>
        </div>
      </article>
    </div>
  </div>
</template>
