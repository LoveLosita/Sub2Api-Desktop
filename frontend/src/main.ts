import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import './style.css'

const app = createApp(App)

app.config.errorHandler = (err, instance, info) => {
  console.error('Vue error:', err, info)
}

app.use(createPinia())
app.use(router)
app.mount('#app')
