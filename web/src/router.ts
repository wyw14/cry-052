import { createRouter, createWebHistory } from 'vue-router'
import Dashboard from './pages/Dashboard.vue'
import DataSources from './pages/DataSources.vue'
import Catalog from './pages/Catalog.vue'
import Policies from './pages/Policies.vue'
import Preview from './pages/Preview.vue'
import Batches from './pages/Batches.vue'
import Reports from './pages/Reports.vue'
import Audit from './pages/Audit.vue'
import Versions from './pages/Versions.vue'

export default createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: Dashboard },
    { path: '/sources', component: DataSources },
    { path: '/catalog', component: Catalog },
    { path: '/policies', component: Policies },
    { path: '/preview', component: Preview },
    { path: '/batches', component: Batches },
    { path: '/reports', component: Reports },
    { path: '/audit', component: Audit },
    { path: '/versions', component: Versions }
  ]
})
