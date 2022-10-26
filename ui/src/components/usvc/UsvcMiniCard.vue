<template>
  <v-btn
    x-large
    :color="color"
  >
    <div v-if="!alive">
      WARNING&nbsp;
    </div>
    <v-icon>
      {{ icon }}
    </v-icon>
  </v-btn>
</template>

<script>
  import { sync } from 'vuex-pathify'
  export default {
    name: 'UsvcMiniCard',

    props: {
      usvc: String,
    },

    data: () => ({
      service: {},
      alive: false,
      color: 'error',
      icon: 'mdi-access-point-off',
    }),

    computed: {
      ...sync('usvc', [
        'services',
        'lastseen',
      ]),
    },

    watch: {
      // whenever question changes, this function will run
      lastseen (news, olds) {
        this.service = this.services[this.usvc]
        if (!this.service) return
        // console.log('this.service: ' + JSON.stringify(this.service))
        for (const i in this.service) {
          var instance = this.service[i]
          if (instance === true) continue
          // console.log('service: ' + this.usvc + ', instance: ' + instance + ', state: ' + instance.state)
          if (instance.state === 'alive') {
            this.alive = true
            this.color = 'success'
          } else if (instance.state === 'stalling') {
            // console.log('service instance stalling: ' + instance.appname + ': ' + JSON.stringify(instance))
            this.alive = false
            this.color = 'warning'
          } else {
            this.alive = false
            this.color = 'error'
          }
        }

        this.icon = this.alive ? 'mdi-access-point' : 'mdi-access-point-off'
      },
    },
  }
</script>

<style lang="sass">
  .v-card.v-card--material
    > .v-card__title
      > .v-card--material__title
        flex: 1 1 auto
        word-break: break-word
</style>
