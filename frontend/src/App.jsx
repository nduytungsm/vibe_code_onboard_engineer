import { useState } from 'react'
import { FileText, GitBranch, Database, Server, Code2, BarChart3 } from 'lucide-react'

function App() {
  const [activeTab, setActiveTab] = useState('overview')

  const mockData = {
    project: {
      name: "Repository Analysis Dashboard",
      type: "Backend",
      confidence: "Very High",
      architecture: "microservices",
      techStack: ["Go", "React", "PostgreSQL"]
    },
    services: [
      { name: "api-gateway", type: "HTTP", port: "8080" },
      { name: "user-service", type: "gRPC", port: "50051" },
      { name: "payment-service", type: "HTTP", port: "8082" }
    ],
    database: {
      tables: ["users", "orders", "products", "categories"],
      relationships: 3
    }
  }

  const tabs = [
    { id: 'overview', name: 'Overview', icon: BarChart3 },
    { id: 'services', name: 'Services', icon: Server },
    { id: 'database', name: 'Database', icon: Database },
    { id: 'relationships', name: 'Dependencies', icon: GitBranch },
    { id: 'files', name: 'Files', icon: FileText },
    { id: 'analysis', name: 'Analysis', icon: Code2 }
  ]

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white shadow-sm border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center">
              <Code2 className="h-8 w-8 text-primary-600" />
              <h1 className="ml-3 text-xl font-semibold text-gray-900">
                Repository Analyzer
              </h1>
            </div>
            <div className="flex items-center space-x-4">
              <span className="badge badge-success">Active</span>
              <button className="btn btn-primary">
                Analyze New Repository
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* Navigation */}
      <nav className="bg-white shadow-sm">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex space-x-8">
            {tabs.map((tab) => {
              const Icon = tab.icon
              return (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={`flex items-center px-1 py-4 text-sm font-medium border-b-2 transition-colors ${
                    activeTab === tab.id
                      ? 'border-primary-500 text-primary-600'
                      : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                  }`}
                >
                  <Icon className="h-4 w-4 mr-2" />
                  {tab.name}
                </button>
              )
            })}
          </div>
        </div>
      </nav>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {activeTab === 'overview' && (
          <div className="space-y-6">
            {/* Project Summary */}
            <div className="card">
              <div className="card-header">
                <h2>Project Analysis Summary</h2>
              </div>
              <div className="card-content">
                <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                  <div className="text-center">
                    <div className="text-3xl font-bold text-primary-600">
                      {mockData.project.type}
                    </div>
                    <div className="text-sm text-gray-500">Project Type</div>
                  </div>
                  <div className="text-center">
                    <div className="text-3xl font-bold text-green-600">
                      {mockData.project.confidence}
                    </div>
                    <div className="text-sm text-gray-500">Confidence</div>
                  </div>
                  <div className="text-center">
                    <div className="text-3xl font-bold text-blue-600">
                      {mockData.services.length}
                    </div>
                    <div className="text-sm text-gray-500">Services</div>
                  </div>
                </div>
              </div>
            </div>

            {/* Quick Stats */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
              <div className="card">
                <div className="card-content">
                  <div className="flex items-center">
                    <Server className="h-8 w-8 text-blue-500" />
                    <div className="ml-4">
                      <div className="text-sm font-medium text-gray-500">Architecture</div>
                      <div className="text-lg font-semibold">{mockData.project.architecture}</div>
                    </div>
                  </div>
                </div>
              </div>

              <div className="card">
                <div className="card-content">
                  <div className="flex items-center">
                    <Database className="h-8 w-8 text-green-500" />
                    <div className="ml-4">
                      <div className="text-sm font-medium text-gray-500">Database Tables</div>
                      <div className="text-lg font-semibold">{mockData.database.tables.length}</div>
                    </div>
                  </div>
                </div>
              </div>

              <div className="card">
                <div className="card-content">
                  <div className="flex items-center">
                    <GitBranch className="h-8 w-8 text-purple-500" />
                    <div className="ml-4">
                      <div className="text-sm font-medium text-gray-500">Dependencies</div>
                      <div className="text-lg font-semibold">{mockData.database.relationships}</div>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            {/* Tech Stack */}
            <div className="card">
              <div className="card-header">
                <h3>Technology Stack</h3>
              </div>
              <div className="card-content">
                <div className="flex flex-wrap gap-2">
                  {mockData.project.techStack.map((tech) => (
                    <span key={tech} className="badge badge-primary">
                      {tech}
                    </span>
                  ))}
                </div>
              </div>
            </div>
          </div>
        )}

        {activeTab === 'services' && (
          <div className="space-y-6">
            <div className="card">
              <div className="card-header">
                <h2>Discovered Services</h2>
              </div>
              <div className="card-content">
                <div className="space-y-4">
                  {mockData.services.map((service) => (
                    <div key={service.name} className="flex items-center justify-between p-4 border border-gray-200 rounded-lg">
                      <div className="flex items-center">
                        <Server className="h-6 w-6 text-blue-500 mr-3" />
                        <div>
                          <div className="font-medium">{service.name}</div>
                          <div className="text-sm text-gray-500">Port: {service.port}</div>
                        </div>
                      </div>
                      <span className={`badge ${service.type === 'gRPC' ? 'badge-warning' : 'badge-primary'}`}>
                        {service.type}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>
        )}

        {activeTab === 'database' && (
          <div className="space-y-6">
            <div className="card">
              <div className="card-header">
                <h2>Database Schema</h2>
              </div>
              <div className="card-content">
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                  {mockData.database.tables.map((table) => (
                    <div key={table} className="p-4 border border-gray-200 rounded-lg text-center">
                      <Database className="h-6 w-6 text-green-500 mx-auto mb-2" />
                      <div className="font-medium">{table}</div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Placeholder for other tabs */}
        {!['overview', 'services', 'database'].includes(activeTab) && (
          <div className="card">
            <div className="card-content">
              <div className="text-center py-12">
                <Code2 className="h-12 w-12 text-gray-400 mx-auto mb-4" />
                <h3 className="text-lg font-medium text-gray-900 mb-2">
                  {tabs.find(t => t.id === activeTab)?.name} Coming Soon
                </h3>
                <p className="text-gray-500">
                  This section will display detailed {activeTab} information.
                </p>
              </div>
            </div>
          </div>
        )}
      </main>
    </div>
  )
}

export default App
