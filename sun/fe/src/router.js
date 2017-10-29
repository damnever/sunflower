import Vue from 'vue'
import VueRouter from 'vue-router'

import Login from './views/Login.vue'
import Index from './views/Index.vue'
import Agent from './views/Agent.vue'
import Tunnel from './views/Tunnel.vue'
import Profile from './views/Profile.vue'
import Admin from './views/admin/Admin.vue'


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
    'name': 'Tunnel',
    'path': '/agent/:ahash|:etag/tunnel/:thash|:ttag',
    'component': Tunnel,
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
]

const router = new VueRouter({
  routes
})
export default router
