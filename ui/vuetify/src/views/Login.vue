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
      <v-row
        justify="center"
      >
        <v-img
          class="rounded-circle elevation-12"
          src="@/assets/cyops-se.png"
          max-width="128"
          contain
          :aspect-ratio="1"
        />
      </v-row>
      <v-card class="elevation-12 theme--dark">
        <v-toolbar
          color="primary"
          dark
          flat
        >
          <v-toolbar-title>Login</v-toolbar-title>
          <v-spacer />
          <span
            style="font-size:0.6rem; "
          >{{ sysinfo.gitversion }}</span>
        </v-toolbar>
        <v-form
          ref="form"
          lazy-validation
          @submit.prevent="submit"
        >
          <v-card-text>
            <v-text-field
              v-model="email"
              label="E-mail"
              prepend-icon="mdi-account"
              type="text"
              :rules="emailRules"
              required
            />

            <v-text-field
              id="password"
              v-model="password"
              label="Password"
              prepend-icon="mdi-lock"
              type="password"
              :rules="[v => !!v || 'Password is required']"
              required
            />
            No account? Register <router-link
              :to="{name: 'Register'}"
            >
              here!
            </router-link>
          </v-card-text>
          <v-card-actions>
            <v-spacer />
            <v-btn
              color="primary"
              type="submit"
              block
            >
              Login
            </v-btn>
          </v-card-actions>
        </v-form>
      </v-card>
      <v-spacer />

      <v-snackbar
        v-model="snackbar"
        timeout="2000"
        color="error"
        top
        center
        elevation="15"
        type="error"
      >
        Login failed! Please try again.
      </v-snackbar>
    </v-col>
  </v-row>
</template>

<script>
  import { get } from 'vuex-pathify'
  export default {
    props: {
      source: String,
    },

    data: () => ({
      snackbar: false,
      email: null,
      password: null,
      emailRules: [
        v => !!v || 'Email is required',
        v => /.+@.+/.test(v) || 'E-mail must be valid',
      ],
    }),

    computed: { sysinfo: get('app/sysinfo') },

    mounted () {
    },

    methods: {
      async submit () {
        const form = this.$refs.form
        if (form.validate()) {
          try {
            this.loading = true
            // clear existing errors
            const email = this.email
            const password = this.password

            this.$store
              .dispatch('auth/login', { email, password })
              .then((data) => {
                this.$store.dispatch('user/populate', data.user)
                this.$router.push({ name: 'Dashboard' })
              })
              .catch(() => {
                this.snackbar = true
              })
          } catch (e) {
            this.responseStatus = e.responseStatus || e
          } finally {
            this.loading = false
            form.resetValidation()
          }
        }
      },
    },
  }
</script>

<style>
.v-image {
  position: absolute;
  background-color: rgb(4, 82, 82);
  z-index: 99;
  transform: translateY(-50px);
}
</style>
