<template>
  <!--v-dialog
    :value="dialog"
    max-width="450px"
    @click:outside="closeDialog"
  -->
  <v-dialog
    :value="dialog"
    max-width="450px"
  >
    <v-card
      :class="{ 'grey lighten-2': dragover }"
      @drop.prevent="onDrop($event)"
      @dragover.prevent="dragover = true"
      @dragenter.prevent="dragover = true"
      @dragleave.prevent="dragover = false"
    >
      <v-card-text>
        <v-row
          class="d-flex flex-column"
          dense
          align="center"
          justify="center"
        >
          <v-icon
            :class="[dragover ? 'mt-2, mb-6' : 'mt-5']"
            size="60"
          >
            mdi-cloud-upload
          </v-icon>
          <p>
            {{ text }}
          </p>
        </v-row>
        <v-virtual-scroll
          v-if="uploadedFiles.length > 0"
          :items="uploadedFiles"
          height="150"
          item-height="50"
        >
          <template v-slot:default="{ item }">
            <v-list-item :key="item.name">
              <v-list-item-content>
                <v-list-item-title>
                  {{ item.name }}
                  <span class="ml-3 text--secondary">
                    {{ item.size }} bytes</span>
                </v-list-item-title>
              </v-list-item-content>

              <v-list-item-action>
                <v-btn
                  icon
                  @click.stop="removeFile(item.name)"
                >
                  <v-icon> mdi-close-circle </v-icon>
                </v-btn>
              </v-list-item-action>
            </v-list-item>

            <v-divider />
          </template>
        </v-virtual-scroll>
      </v-card-text>
      <v-card-actions>
        <v-spacer />

        <v-btn
          icon
          @click="closeDialog"
        >
          <v-icon id="close-button">
            mdi-close
          </v-icon>
        </v-btn>

        <v-btn
          icon
          @click.stop="submit"
        >
          <v-icon id="upload-button">
            mdi-upload
          </v-icon>
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script>
  export default {
    name: 'FileDrop',

    props: {
      dialog: {
        type: Boolean,
        required: true,
      },
      multiple: {
        type: Boolean,
        default: false,
      },
      text: String,
    },

    data () {
      return {
        dragover: false,
        uploadedFiles: [],
      }
    },

    methods: {
      closeDialog () {
        // Remove all the uploaded files
        this.uploadedFiles = []
        // Close the dialog box
        this.$emit('update:dialog', false)
      },

      removeFile (fileName) {
        // Find the index of the
        const index = this.uploadedFiles.findIndex(
          file => file.name === fileName,
        )

        // If file is in uploaded files remove it
        if (index > -1) this.uploadedFiles.splice(index, 1)
      },

      onDrop (e) {
        this.dragover = false

        // If there are already uploaded files remove them
        if (this.uploadedFiles.length > 0) this.uploadedFiles = []

        // If user has uploaded multiple files but the component is not multiple throw error
        if (!this.multiple && e.dataTransfer.files.length > 1) {
          this.$notification.error('Only one file can be uploaded at a time ...')
        } else {
          e.dataTransfer.files.forEach(element =>
            this.uploadedFiles.push(element),
          )
        }
      },

      submit () {
        // If there aren't any files to be uploaded throw error
        if (!this.uploadedFiles.length > 0) {
          this.$notification.error('There are no files to upload ...')
        } else {
          // Send uploaded files to parent component
          console.log('files to upload: ' + this.uploadedFiles[0].name)
          this.$emit('filesUploaded', this.uploadedFiles)
          // Close the dialog box
          this.closeDialog()
        }
      },
    },
  }
</script>
