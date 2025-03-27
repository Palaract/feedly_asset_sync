<template>
    <div class="container">
      <h1>Feedly Sync</h1>
      
      <div class="config-section">
        <h2>Configuration</h2>
        <div class="form-group">
          <label>Upload URL:</label>
          <input v-model="config.upload_url" type="text" />
        </div>
        <div class="form-group">
          <label>API Key:</label>
          <input v-model="config.api_key" type="password" />
        </div>
        <button @click="saveConfig" :disabled="saving">
          {{ saving ? 'Saving...' : 'Save Configuration' }}
        </button>
      </div>
  
      <div class="sync-section">
        <h2>Upload CSV and Sync</h2>
        <div 
          class="drop-zone" 
          @drop.prevent="handleDrop"
          @dragover.prevent="dragover = true"
          @dragleave.prevent="dragover = false"
          :class="{ 'dragover': dragover }"
        >
          <div v-if="selectedFile">
            <div class="file-info">
              <span>{{ selectedFile.name }}</span>
              <button class="remove-file" @click.stop="removeFile">×</button>
            </div>
          </div>
          <div v-else>
            <p>Drag and drop your CSV file here</p>
            <p>or</p>
            <button class="upload-btn" @click="triggerFileInput">Select File</button>
          </div>
          <input 
            type="file" 
            ref="fileInput" 
            style="display: none" 
            accept=".csv"
            @change="handleFileSelect"
          >
        </div>
  
        <button 
          @click="syncData" 
          :disabled="syncing || !selectedFile" 
          class="sync-button"
        >
          {{ syncing ? 'Syncing...' : 'Start Sync' }}
        </button>
  
        <div v-if="syncMessage" :class="['message', syncMessage.includes('Error') ? 'error' : 'success']">
          {{ syncMessage }}
        </div>
      </div>
    </div>
  </template>
  
  <script>
  export default {
    data() {
      return {
        config: {
          upload_url: '',
          api_key: ''
        },
        saving: false,
        syncing: false,
        syncMessage: '',
        selectedFile: null,
        dragover: false
      }
    },
    async mounted() {
      try {
        const config = await window.go.main.App.GetConfig()
        this.config = config
      } catch (error) {
        console.error('Error loading config:', error)
      }
    },
    methods: {
      async saveConfig() {
        this.saving = true
        try {
          await window.go.main.App.UpdateConfig(this.config)
          this.syncMessage = 'Configuration saved successfully'
        } catch (error) {
          this.syncMessage = `Error saving configuration: ${error}`
        }
        this.saving = false
      },
      
      triggerFileInput() {
        this.$refs.fileInput.click()
      },
  
      handleFileSelect(event) {
        const file = event.target.files[0]
        if (file && file.type === 'text/csv') {
          this.selectedFile = file
          this.syncMessage = ''
        } else {
          this.syncMessage = 'Please select a valid CSV file'
        }
      },
  
      handleDrop(event) {
        this.dragover = false
        const file = event.dataTransfer.files[0]
        if (file && file.type === 'text/csv') {
          this.selectedFile = file
          this.syncMessage = ''
        } else {
          this.syncMessage = 'Please drop a valid CSV file'
        }
      },
  
      removeFile() {
        this.selectedFile = null
        this.syncMessage = ''
        this.$refs.fileInput.value = ''
      },
  
      async syncData() {
        if (!this.selectedFile) {
          this.syncMessage = 'Please select a CSV file first'
          return
        }
  
        this.syncing = true
        this.syncMessage = ''
  
        try {
          const csvContent = await this.readFileContent(this.selectedFile)
          const result = await window.go.main.App.ProcessCSVData(csvContent)
          this.syncMessage = result
          this.selectedFile = null
          this.$refs.fileInput.value = ''
        } catch (error) {
          this.syncMessage = `Error during sync: ${error}`
        }
        this.syncing = false
      },
  
      readFileContent(file) {
        return new Promise((resolve, reject) => {
          const reader = new FileReader()
          reader.onload = (event) => resolve(event.target.result)
          reader.onerror = (error) => reject(error)
          reader.readAsText(file)
        })
      }
    }
  }
  </script>
  
  <style>
  .container {
    max-width: 800px;
    margin: 0 auto;
    padding: 20px;
  }
  
  h1 {
    text-align: center;
    color: #333;
  }
  
  .config-section, .sync-section {
    background: #f5f5f5;
    padding: 20px;
    border-radius: 8px;
    margin: 20px 0;
  }
  
  .form-group {
    margin-bottom: 15px;
  }
  
  label {
    display: block;
    margin-bottom: 5px;
    font-weight: bold;
  }
  
  input {
    width: 100%;
    padding: 8px;
    border: 1px solid #ddd;
    border-radius: 4px;
  }
  
  button {
    background: #4CAF50;
    color: white;
    padding: 10px 20px;
    border: none;
    border-radius: 4px;
    cursor: pointer;
  }
  
  button:disabled {
    background: #cccccc;
    cursor: not-allowed;
  }
  
  .drop-zone {
    border: 2px dashed #ccc;
    border-radius: 8px;
    padding: 40px;
    text-align: center;
    margin: 20px 0;
    transition: all 0.3s ease;
    background: white;
  }
  
  .drop-zone.dragover {
    border-color: #4CAF50;
    background: #f0f9f0;
  }
  
  .upload-btn {
    background: #666;
    margin-top: 10px;
  }
  
  .file-info {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 10px;
  }
  
  .remove-file {
    background: #ff4444;
    padding: 2px 8px;
    border-radius: 50%;
    font-size: 16px;
  }
  
  .message {
    margin-top: 10px;
    padding: 10px;
    border-radius: 4px;
  }
  
  .success {
    background: #dff0d8;
    color: #3c763d;
  }
  
  .error {
    background: #f2dede;
    color: #a94442;
  }
  
  .sync-button {
    width: 100%;
    margin-top: 20px;
    padding: 15px;
    font-size: 16px;
  }
  </style>  