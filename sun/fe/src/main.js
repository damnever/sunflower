require('element-ui/lib/theme-default/index.css')
require('./assets/css/style.css')
require('./assets/icons/favicon.ico')

import Vue from 'vue'
Vue.config.errorHandler = (err, vm) => {
  console.log(err, vm)
}

import ElementUI from 'element-ui'
import VueResource from 'vue-resource'
import App from './App.vue'
import router from './router'


Vue.use(ElementUI)
Vue.use(VueResource)
Vue.http.options.emulateJSON = true;

new Vue({
  el: '#app',
  router: router,
  render: h => h(App)
})
