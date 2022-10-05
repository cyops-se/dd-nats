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

    <usvc-mini-card
      usvc="ddnatsinnerproxy"
    />
    <usvc-mini-card
      v-if="outer"
      usvc="ddnatsouterproxy"
    />

    <!-- default-go-home / -->

    <!-- default-account / -->
  </v-app-bar>
</template>

<script>
  // Utilities
  import { get, sync } from 'vuex-pathify'
  import UsvcMiniCard from '../../components/usvc/UsvcMiniCard.vue'
  import WebsocketService from '@/services/websocket.service'

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
      UsvcMiniCard,
    },

    data: () => ({
      outer: false,
    }),

    computed: {
      ...sync('app', [
        'drawer',
        'mini',
      ]),
      name: get('route/name'),
    },

    created () {
      WebsocketService.topic('system.heartbeat', this, function (topic, msg, t) {
        var appname = msg.appname.replaceAll('-', '')
        if (appname === 'ddnatsouterproxy') t.outer = true
      })
    },
  }
</script>

<style lang="sass">
#default-drawer
  .v-list-group__items
    .v-list-item
      min-height: 28px
      .v-list-item__title
        padding-left: 10px
        font-size: .75rem

  .v-list-group
    .v-list-group__header__append-icon
      margin-left: 0px
</style>
