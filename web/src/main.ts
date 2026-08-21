import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import './styles.css'
import './workflows.css'

const governanceConsole = createApp(App)
const state = createPinia()

type UIBootstrap = { framework: string; locale: string; routes: string[]; version: number }

async function loadBootstrap(): Promise<UIBootstrap> {
  const response = await fetch('/api/v1/ui/bootstrap')
  return response.json() as Promise<UIBootstrap>
}

governanceConsole.config.errorHandler = (error, _instance, context) => {
  console.error('治理控制台渲染失败', { context, error })
}
governanceConsole.use(state)
governanceConsole.use(router)
loadBootstrap().finally(() => governanceConsole.mount(document.querySelector<HTMLElement>('#app')!))
