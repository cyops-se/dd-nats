import Vue from 'vue'
import App from './App.vue'
import router from './router'
import vuetify from './plugins/vuetify'
import './plugins'
import store from './store'
import { sync } from 'vuex-router-sync'
import VueNotification from '@kugatsu/vuenotification'

Vue.config.productionTip = false
Vue.use(VueNotification, {
  timer: 5,
  showCloseIcn: true,
})

sync(store, router)

router.beforeEach((to, from, next) => {
  // Ensure we checked auth before each page load
  // store.dispatch('auth/verify')
  // .then(data => {
  //   next()
  // }).catch(response => {
  //   if (to.name !== 'Login' && to.name !== 'Register') {
  //     next('/auth/login')
  //   } else next()
  // })

  next()

  // Scroll page to top on every route change
  setTimeout(() => {
    window.scrollTo(0, 0)
  }, 100)
})

new Vue({
  router,
  vuetify,
  store,
  render: h => h(App),
}).$mount('#app')
