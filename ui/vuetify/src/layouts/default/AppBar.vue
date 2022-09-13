<template>
  <v-app-bar
    id="default-app-bar"
    app
    fixed
    class="v-bar--underline"
    :clipped-left="$vuetify.rtl"
    :clipped-right="!$vuetify.rtl"
    height="70"
    flat
  >
    <v-app-bar-nav-icon
      class="hidden-md-and-up"
      @click="drawer = !drawer"
    />

    <default-drawer-toggle class="hidden-sm-and-down" />

    <v-toolbar-title
      class="font-weight-light text-h5"
      v-text="name"
    />

    <v-spacer />

    <usvc-mini-card usvc="dd-nats-inner-proxy" />

    <default-go-home />

    <!-- default-account / -->
  </v-app-bar>
</template>

<script>
  // Utilities
  import { get, sync } from 'vuex-pathify'
  import UsvcMiniCard from '../../components/usvc/UsvcMiniCard.vue'

  export default {
    name: 'DefaultBar',

    components: {
      // DefaultAccount: () => import(
      //   /* webpackChunkName: "default-account" */
      //   './widgets/Account'
      // ),
      DefaultDrawerToggle: () => import(
        /* webpackChunkName: "default-drawer-toggle" */
        './widgets/DrawerToggle'
      ),
      DefaultGoHome: () => import(
        /* webpackChunkName: "default-go-home" */
        './widgets/GoHome'
      ),
      UsvcMiniCard,
    },

    computed: {
      ...sync('app', [
        'drawer',
        'mini',
      ]),
      name: get('route/name'),
    },
  }
</script>
