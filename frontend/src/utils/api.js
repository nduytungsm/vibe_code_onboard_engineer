import axios from 'axios'

// API base configuration
const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor for logging
api.interceptors.request.use(
  (config) => {
    console.log(`🚀 API Request: ${config.method?.toUpperCase()} ${config.url}`)
    return config
  },
  (error) => {
    console.error('❌ API Request Error:', error)
    return Promise.reject(error)
  }
)

// Response interceptor for error handling
api.interceptors.response.use(
  (response) => {
    console.log(`✅ API Response: ${response.status} ${response.config.url}`)
    return response
  },
  (error) => {
    console.error('❌ API Response Error:', error.response?.data || error.message)
    return Promise.reject(error)
  }
)

// Repository Analysis API
export const repositoryAPI = {
  // Analyze a new repository
  analyzeRepository: async (repositoryPath) => {
    const response = await api.post('/api/analyze', { path: repositoryPath })
    return response.data
  },

  // Get project analysis results
  getProjectAnalysis: async (analysisId) => {
    const response = await api.get(`/api/analysis/${analysisId}`)
    return response.data
  },

  // Get discovered services
  getServices: async (analysisId) => {
    const response = await api.get(`/api/analysis/${analysisId}/services`)
    return response.data
  },

  // Get service relationships
  getRelationships: async (analysisId) => {
    const response = await api.get(`/api/analysis/${analysisId}/relationships`)
    return response.data
  },

  // Get database schema
  getDatabaseSchema: async (analysisId) => {
    const response = await api.get(`/api/analysis/${analysisId}/database`)
    return response.data
  },

  // Get file analysis
  getFileAnalysis: async (analysisId) => {
    const response = await api.get(`/api/analysis/${analysisId}/files`)
    return response.data
  },

  // Get Mermaid graph data
  getMermaidGraph: async (analysisId) => {
    const response = await api.get(`/api/analysis/${analysisId}/mermaid`)
    return response.data
  },

  // Get PlantUML ERD
  getPlantUMLERD: async (analysisId) => {
    const response = await api.get(`/api/analysis/${analysisId}/plantuml`)
    return response.data
  },
}

// Health check
export const healthAPI = {
  check: async () => {
    const response = await api.get('/health')
    return response.data
  },
}

export default api
