<template>
  <v-card
    v-bind="$attrs"
    class="v-card--material mt-4"
  >
    <v-card-title class="align-start">
      <v-sheet
        :color="color"
        width="100%"
        class="overflow-hidden mt-n9 transition-swing v-card--material__sheet d-flex justify-center"
        justify="center"
        elevation="6"
        max-width="100%"
        rounded
      >
        <div class="ml-3 my-auto">
          <v-icon large>
            mdi-tag-multiple
          </v-icon>
        </div>
        <v-spacer />
        <div class="pa-3 white--text my-auto">
          <span class="text-h3 text-no-wrap">
            {{ copy.name }}
          </span>
          <!-- <span class="text-h4 text-no-wrap">
            {{ copy.state >= 2 ? 'RUNNING' : 'STOPPED' }}
          </span> -->
          <div>Send count: {{ copy.counter }}</div>
        </div>
        <v-spacer />
        <div class="my-auto mr-3">
          <v-btn @click="startStop">
            <div v-html="copy.state >= 2 ? 'STOP' : 'START'" />
          </v-btn>
        </div>
      </v-sheet>

      <div class="pl-3 text-h4 v-card--material__title">
        <div class="text-subtitle-1 mb-n4 mt-4">
          <template>
            {{ copy.description }}
          </template>
        </div>
      </div>
    </v-card-title>

    <slot />

    <template>
      <v-divider class="mt-2 mx-4" />

      <v-card-actions class="px-4 text-caption grey--text">
        <v-icon
          class="mr-1"
          small
        >
          mdi-clock-outline
        </v-icon>

        <span
          v-if="copy && copy.lastrun"
          class="text-caption grey--text font-weight-light"
          v-text="'Last run: ' + copy.lastrun.replace('T', ' ').substring(0, 19)"
        />
      </v-card-actions>
    </template>
  </v-card>
</template>

<script>
  import ApiService from '@/services/api.service'
  import WebsocketService from '@/services/websocket.service'
  export default {
    name: 'MaterialGroupCard',

    inheritAttrs: false,

    props: {
      group: {
        type: Object,
        default: () => ({}),
      },
      eventHandlers: {
        type: Array,
        default: () => ([]),
      },
    },

    data: () => ({
      color: 'error',
      copy: {},
    }),

    watch: {
      $route (to, from) {
        console.log('route change: ', to, from)
      },
    },

    created () {
      this.copy = Object.assign({}, this.group)
      this.color = this.group.state < 2 ? 'error' : this.group.state === 2 ? 'success' : 'warning'
      var subject = 'system.event.group.' + this.group.id + '.*'
      WebsocketService.topic(subject, this, function (topic, group, target) {
        target.updateGroup(target, group)
      })
    },

    methods: {
      startStop () {
        var action = this.copy.state >= 2 ? 'stop' : 'start'
        var request = { subject: 'usvc.opc.groups.' + action, payload: { value: parseInt(this.group.id) } }
        ApiService.post('nats/request', request)
          .then(response => {
            if (response.data.success) {
              // this.copy = response.data.item
              this.$notification.success('Group ' + action + ' succeeded')
            } else {
              this.$notification.error('Failed to ' + action + ' group: ' + response.data.statusmsg)
            }
          }).catch(response => {
            console.log('ERROR response: ' + response.message)
            this.$notification.error('Failed to ' + action + ' group: ' + response.message)
          })
      },

      refresh () {
        var request = { subject: 'usvc.opc.groups.getbyid', payload: { value: parseInt(this.group.id) } }
        ApiService.post('nats/request', request)
          .then(response => {
            this.copy = response.data.item
          }).catch(response => {
            console.log('ERROR response: ' + response.message)
            this.$notification.error('Failed to get groups: ' + response.message)
          })
      },

      updateGroup (target, group) {
        target.copy.state = group.state
        target.color = group.state < 2 ? 'error' : group.state === 2 ? 'success' : 'warning'
        target.copy = group
      },
    },
  }
</script>

<style lang="sass">
.group-button
  font-size: .875rem !important
  margin-left: auto
  text-align: right
</style>
