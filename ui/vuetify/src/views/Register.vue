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
        <v-toolbar
          color="primary"
          dark
          flat
        >
          <v-toolbar-title>New user</v-toolbar-title>
          <v-spacer />
          <span
            style="font-size:0.6rem; "
          >{{ sysinfo.gitversion }}</span>
        </v-toolbar>
        <v-card-text>
          <v-form
            ref="form"
            lazy-validation
          >
            <v-text-field
              v-model="fullname"
              label="Full name"
              prepend-icon="mdi-account"
              type="text"
              required
            />

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

            <v-text-field
              id="password2"
              v-model="password2"
              label="Password (confirm)"
              prepend-icon="mdi-lock"
              type="password"
              :rules="[v => !!v || 'Password confirmation is required']"
              required
            />
            Already have an account? Login <router-link
              :to="{name: 'Login'}"
            >
              here!
            </router-link>
          </v-form>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn
            color="primary"
            type="submit"
            block
            @click="submit"
          >
            Register
          </v-btn>
        </v-card-actions>
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

    data: () => ({
      fullname: null,
      email: null,
      password: null,
      password2: null,
      emailRules: [
        v => !!v || 'Email is required',
        v => /.+@.+/.test(v) || 'E-mail must be valid',
      ],
    }),

    computed: { sysinfo: get('app/sysinfo') },

    methods: {
      async submit () {
        const form = this.$refs.form
        if (form.validate()) {
          try {
            this.loading = true
            // clear existing errors
            const fullname = this.fullname
            const email = this.email
            const password = this.password

            this.$store
              .dispatch('auth/register', { fullname, email, password })
              .then(() => this.$router.push({ name: 'Login' }))
              .catch(() => {})
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
</style>
