<template>
  <v-list-item class="mb-0 justify-space-between pl-3">
    <v-list-item-avatar
      color="transparent"
      rounded="0"
    >
      <v-img
        contain
        :src="
          require('@/assets/cyops-se.png')"
      />
    </v-list-item-avatar>

    <v-list-item-content class="pl-2">
      <v-list-item-title class="text-h4">
        <strong class="mr-1 font-weight-black">DD</strong>
        <span class="accent--text">-OPCDA</span>
      </v-list-item-title>
      <span style="font-size: .7rem">{{ sysinfo.gitversion }}</span>
      <span
        style="font-size: .7rem"
        class="mr-1 font-weight-black"
      >CYOPS-SE</span>
    </v-list-item-content>
  </v-list-item>
</template>

<script>
  // Utilities
  import { sync } from 'vuex-pathify'

  export default {
    name: 'DefaultDrawerHeader',

    computed: {
      sysinfo: sync('app/sysinfo'),
    },
  }
</script>
