import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import About from '@/components/About.vue'
import Playground from '@/components/Playground.vue'

const routeNames = {
  Playground: 'playground',
  About: 'about'
}

const routeDefinitions: RouteRecordRaw[] = [
  {
    path: '/',
    component: Playground,
    name: routeNames.Playground
  },
  {
    path: '/about',
    component: About,
    name: routeNames.About
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes: routeDefinitions
})

export { routeDefinitions, routeNames, router }
