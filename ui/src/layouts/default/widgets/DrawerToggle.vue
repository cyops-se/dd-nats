<template>
  <v-btn
    class="ml-0 mr-4"
    elevation="1"
    fab
    small
    @click="mini = !mini"
  >
    <v-icon>
      {{ mini ? "mdi-chevron-double-right" : "mdi-chevron-double-left" }}
    </v-icon>
  </v-btn>
</template>

<script>
// Utilities
  import { sync } from 'vuex-pathify'

  export default {
    name: 'DefaultDrawerToggle',

    computed: {
      mini: sync('app/mini'),
    },
  }
</script>
