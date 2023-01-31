<template>
  <v-fade-transition mode="out-in">
    <router-view />
  </v-fade-transition>
</template>

<script>
  // Styles
  import '@/styles/overrides.sass'
  import { dispatch, sync } from 'vuex-pathify'
  import WebsocketService from '@/services/websocket.service'

  export default {
    name: 'App',
    metaInfo: {
      title: 'dd-console',
      titleTemplate: '%s | cyops-se admin',
      htmlAttrs: { lang: 'en' },
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
      ],
    },

    computed: {
      sysinfo: sync('app/sysinfo'),
    },

    created () {
      WebsocketService.connect(this.onclose)
      dispatch('app/init')
      dispatch('usvc/init')
    },

    methods: {
      onclose () {
        console.log('Websocket closed')
        WebsocketService.connect(this.onclose)
      },
    },
  }
</script>
