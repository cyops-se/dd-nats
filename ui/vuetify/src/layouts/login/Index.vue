<template>
  <div
    id="app"
  >
    <v-app
      id="inspire"
      class="theme--dark"
    >
      <v-main>
        <v-img
          :style="style"
          src="@/assets/login.jpg"
        />
        <v-container
          class="fill-height theme--dark"
          fluid
        >
          <v-snackbar
            v-model="snackbar"
            :top="true"
            :right="true"
            dark
            elevation="15"
            :type="snackbarType"
          >
            {{ snackbarMessage }}
          </v-snackbar>
          <login />
        </v-container>
      </v-main>
    </v-app>
  </div>
</template>

<style>
input:-webkit-autofill,
input:-webkit-autofill:hover,
input:-webkit-autofill:focus,
textarea:-webkit-autofill,
textarea:-webkit-autofill:hover,
textarea:-webkit-autofill:focus,
select:-webkit-autofill,
select:-webkit-autofill:hover,
select:-webkit-autofill:focus {
  border: 1px solid transparent;
  -webkit-text-fill-color: white;
  -webkit-box-shadow: 0 0 0px 1000px transparent inset;
  transition: background-color 5000s ease-in-out 0s;
}
</style>

<script>
  import Login from '@/views/Login.vue'
  export default {
    components: {
      Login,
    },

    props: {
      source: String,
    },

    data: () => ({
      snackbar: false,
      snackbarMessage: '',
      snackbarType: 'warning',
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
      this.$vuetify.theme.dark = true
    },

    methods: {
      successMessage (text) {
        this.snackbarMessage = text
        this.snackbarType = 'success'
        this.snackbar = true
      },

      failureMessage (text) {
        this.snackbarMessage = text
        this.snackbarType = 'error'
        this.snackbar = true
      },
    },
  }
</script>
