<template>
  <div id="app">
    <v-app id="inspire">
      <v-main>
        <v-img
          :style="style"
          src="@/assets/login.jpg"
        />
        <v-container
          class="fill-height"
          fluid
        >
          <logout />
        </v-container>
      </v-main>
    </v-app>
  </div>
</template>

<script>
  import Logout from '@/views/Logout.vue'
  export default {
    components: {
      Logout,
    },

    props: {
      source: String,
    },

    data: () => ({
      drawer: true,
      style: {
        'background-image': 'url(~@/assets/login.jpg)',
        'z-index': 0,
        position: 'absolute',
        width: '100%',
        height: '100%',
        opacity: 0.05,
      },
    }),

    created () {
      // this.$vuetify.theme.dark = true
    },
  }
</script>
