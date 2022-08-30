<template>
  <v-row
    align="center"
    justify="center"
  >
    <v-col
      cols="12"
      sm="8"
      md="4"
    >
      <v-card class="elevation-12">
        <v-form
          ref="form"
          lazy-validation
        >
          <v-toolbar
            color="primary"
            dark
            flat
          >
            <v-toolbar-title>Logging out ...</v-toolbar-title>
          </v-toolbar>
        </v-form>
      </v-card>
    </v-col>
  </v-row>
</template>

<script>
  import { get } from 'vuex-pathify'
  export default {
    props: {
      source: String,
    },

    computed: { version: get('app/version') },

    mounted () {
      this.$store.dispatch('auth/logout')
    },

    created () {
      setTimeout(function () { window.location.reload() }, 1000)
    },
  }
</script>

<style>
</style>
