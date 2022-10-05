// Imports
import Vue from 'vue'
import Router from 'vue-router'
// import { trailingSlash } from '@/util/helpers'
import {
  layout,
  route,
} from '@/util/routes'

Vue.use(Router)

const router = new Router({
  mode: 'history',
  // base: '/admin/', // process.env.BASE_URL,
  base: '/ui', // process.env.BASE_URL,
  scrollBehavior: (to, from, savedPosition) => {
    if (to.hash) return { selector: to.hash }
    if (savedPosition) return savedPosition

    return { x: 0, y: 0 }
  },
  routes: [
    layout('Default', [
      route('Dashboard', null, '/'),

      // Pages
      // Inner views
      route('inner/opcda/ServerTable', null, 'pages/opc/servers'),
      route('inner/opcda/TagBrowser', null, 'pages/opc/browse/:serverid'),
      route('inner/opcda/GroupTable', null, 'pages/opc/groups'),
      route('inner/opcda/TagTable', null, 'pages/opc/tags'),
      route('inner/opcda/Settings', null, 'pages/opc/settings'),
      route('inner/modbus/ModbusSlaves', null, 'pages/modbus/slaves'),
      route('inner/modbus/DataPoints', null, 'pages/modbus/datapoints'),
      route('inner/modbus/Settings', null, 'pages/modbus/settings'),
      route('inner/filetransfer/File Transfer', null, 'pages/file/transfer'),
      route('inner/filetransfer/Settings', null, 'pages/file/settings'),
      route('inner/proxy/Settings', null, 'pages/proxy/settings'),
      route('History', null, 'pages/process/cache'),

      // General
      route('System Settings', null, 'pages/systemsettings'),

      // Tables
      route('logs/Logs', null, 'pages/logs/all'),
      route('logs/InfoLogs', null, 'pages/logs/info'),
      route('logs/ErrorLogs', null, 'pages/logs/errors'),
      route('logs/Settings', null, 'pages/logs/settings'),
    ]),
  ],
})

router.beforeEach((to, from, next) => {
  return next() // to.path.endsWith('/') ? next() : next(trailingSlash(to.path))
})

export default router
