import { Quasar, type QuasarPluginOptions } from 'quasar'
import quasarIconSet from 'quasar/icon-set/svg-mdi-v7'
import { createApp } from 'vue'

import { router } from './router'

import 'quasar/src/css/index.sass'

import App from './App.vue'

import './worker/editorWorker'
import axios from 'axios'
import { serverBaseUrl } from './config/server'
import { serviceOptions } from './swagger-client/serviceOptions'

serviceOptions.axios = axios.create({
  baseURL: serverBaseUrl,
  timeout: 30000
})

createApp(App)
  .use(Quasar, <QuasarPluginOptions>{
    iconSet: quasarIconSet,
    plugins: {},
    config: {}
  })
  .use(router)
  .mount('#app')
