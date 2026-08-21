import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import './styles.css'
import './workflows.css'

const governanceConsole = createApp(App)
const state = createPinia()

governanceConsole.config.errorHandler = (error, _instance, context) => {
  console.error('治理控制台渲染失败', { context, error })
}
governanceConsole.use(state)
governanceConsole.use(router)
governanceConsole.mount(document.querySelector<HTMLElement>('#app')!)
