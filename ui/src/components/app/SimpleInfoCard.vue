<template>
  <v-sheet
    :color="color"
    elevation="6"
    class="mb-2"
    rounded
  >
    <v-row>
      <v-col md="2">
        <div
          v-if="icon"
          class="pa-5"
        >
          <v-icon
            large
            v-text="icon"
          />
        </div>
      </v-col>
      <v-col>
        <div class="pr-5">
          <div class="text-right">
            {{ title }}
          </div>
          <div class="text-h2 white--text text-right">
            {{ value }}
          </div>
        </div>
      </v-col>
    </v-row>
  </v-sheet>
</template>

<script>
  export default {
    name: 'SimpleInfoCard',

    props: {
      color: String,
      heading: String,
      icon: String,
      subtitle: String,
      title: String,
      value: String,
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
