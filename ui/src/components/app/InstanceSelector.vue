<template>
  <v-container
    fluid
    class="instance-selector"
  >
    <v-combobox
      v-model="selected"
      :items="items"
      item-text="key"
      outlined
      dense
      hide-details
      placeholder="select instance"
      @change="onchange"
    />
  </v-container>
</template>

<script>
  import { sync } from 'vuex-pathify'
  export default {
    name: 'InstanceSelector',

    props: {
      svcname: String,
    },

    data: () => ({
      items: [],
    }),

    computed: {
      ...sync('context', [
        'selected',
      ]),
      ...sync('usvc', [
        'services',
        'lastseen',
        'statechange',
      ]),
    },

    watch: {
      // whenever question changes, this function will run
      statechange (news, olds) {
        this.refresh()
      },
    },

    created () {
      this.refresh()
    },

    mounted () {
    },

    methods: {
      refresh () {
        if (!this.services[this.svcname]) return
        this.items = []
        const keys = Object.keys(this.services[this.svcname])
        for (var i = 0; i < keys.length; i++) {
          var key = keys[i]
          if (this.services[this.svcname][key].alive) {
            this.items.push({ key: key, value: this.services[this.svcname][key] })
          }
        }

        this.items.sort()

        if (this.items && this.items.length > 0 && !(typeof this.selected === 'object')) {
          this.selected = this.items[0]
          this.$emit('change')
        }
      },

      onchange () {
        this.$emit('change')
      },
    },
  }
</script>
