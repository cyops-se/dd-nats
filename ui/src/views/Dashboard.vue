<template>
  <v-container fluid>
    <inner-dashboard v-if="side === 'inner'" />
    <outer-dashboard v-if="side === 'outer'" />
  </v-container>
</template>

<script>
  // Utilities
  import { sync } from 'vuex-pathify'
  import InnerDashboard from './inner/Dashboard.vue'
  import OuterDashboard from './outer/Dashboard.vue'

  export default {
    name: 'DashboardView',

    components: {
      InnerDashboard,
      OuterDashboard,
    },

    data: () => ({
    }),

    computed: {
      ...sync('usvc', [
        'services',
        'lastseen',
        'side',
      ]),
    },

    created () {
    },
  }
</script>
