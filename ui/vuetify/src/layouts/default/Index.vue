<template>
  <v-app ref="app">
    <default-bar />

    <default-drawer />

    <default-view />

    <default-footer />
  </v-app>
</template>

<script>
  export default {
    name: 'DefaultLayout',

    components: {
      DefaultBar: () => import(
        /* webpackChunkName: "default-app-bar" */
        './AppBar'
      ),
      DefaultDrawer: () => import(
        /* webpackChunkName: "default-drawer" */
        './Drawer'
      ),
      DefaultFooter: () => import(
        /* webpackChunkName: "default-footer" */
        './Footer'
      ),
      DefaultView: () => import(
        /* webpackChunkName: "default-view" */
        './View'
      ),
      // DashboardCoreSettings: () => import(
      //   './Settings'
      // ),
    },

    created () {
      this.$vuetify.theme.dark = true
    },
  }
</script>
