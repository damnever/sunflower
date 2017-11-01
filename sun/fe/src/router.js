import Vue from 'vue'
import VueRouter from 'vue-router'

import Login from './views/Login.vue'
import Index from './views/Index.vue'
import Agent from './views/Agent.vue'
import Profile from './views/Profile.vue'
import Admin from './views/admin/Admin.vue'
import Stats from './views/admin/Stats.vue'


Vue.use(VueRouter)

const routes = [
  {
    'name': 'Login',
    'path': '/login',
    'component': Login,
  },
  {
    'name': 'Index',
    'path': '/',
    'component': Index,
  },
  {
    'name': 'Agent',
    'path': '/agent/:ahash|:etag',
    'component': Agent,
  },
  {
    'name': 'Profile',
    'path': '/profile',
    'component': Profile,
  },
  {
    'name': 'Admin',
    'path': '/admin',
    'component': Admin,
  },
  {
    'name': 'Stats',
    'path': '/stats',
    'component': Stats,
  },

]

const router = new VueRouter({
  routes
})
export default router
