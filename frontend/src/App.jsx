import { useState } from "react";
import {
  FileText,
  GitBranch,
  Database,
  Server,
  Code2,
  BarChart3,
  Github,
  Zap,
  Eye,
  Download,
} from "lucide-react";

function App() {
  const [activeTab, setActiveTab] = useState("overview");
  const [repositoryUrl, setRepositoryUrl] = useState("");
  const [isAnalyzing, setIsAnalyzing] = useState(false);
  const [analysisComplete, setAnalysisComplete] = useState(false);

  const mockData = {
    project: {
      name: "Repository Analysis Dashboard",
      type: "Backend",
      confidence: "Very High",
      architecture: "microservices",
      techStack: ["Go", "React", "PostgreSQL"],
    },
    services: [
      { name: "api-gateway", type: "HTTP", port: "8080" },
      { name: "user-service", type: "gRPC", port: "50051" },
      { name: "payment-service", type: "HTTP", port: "8082" },
    ],
    database: {
      tables: ["users", "orders", "products", "categories"],
      relationships: 3,
    },
  };

  const tabs = [
    { id: "overview", name: "Overview", icon: BarChart3 },
    { id: "services", name: "Services", icon: Server },
    { id: "database", name: "Database", icon: Database },
    { id: "relationships", name: "Dependencies", icon: GitBranch },
    { id: "files", name: "Files", icon: FileText },
    { id: "analysis", name: "Analysis", icon: Code2 },
  ];

  const handleAnalyzeRepository = async () => {
    if (!repositoryUrl.trim()) return;

    setIsAnalyzing(true);

    // TODO: Replace with actual API call
    setTimeout(() => {
      setIsAnalyzing(false);
      setAnalysisComplete(true);
    }, 3000);
  };

  const features = [
    {
      icon: Zap,
      title: "Lightning Fast",
      description:
        "Advanced algorithms analyze your repository in seconds, not minutes.",
    },
    {
      icon: Eye,
      title: "Deep Insights",
      description:
        "Discover hidden patterns, dependencies, and architectural insights.",
    },
    {
      icon: Github,
      title: "GitHub Integration",
      description:
        "Seamlessly works with any public or private GitHub repository.",
    },
    {
      icon: Download,
      title: "Export Ready",
      description: "Download reports, diagrams, and documentation instantly.",
    },
  ];

  return (
    <div className="min-h-screen">
      {/* Header */}
      <header className="notion-header">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center">
              <Code2 className="h-8 w-8 text-gray-600" />
              <h1 className="ml-3 text-lg ml-5 font-semibold text-gray-800">
                Repository Analyzer
              </h1>
            </div>
            <div className="flex items-center space-x-4">
              <span className="badge badge-success">Active</span>
              <button
                className="btn btn-primary"
                onClick={() => {
                  setAnalysisComplete(false);
                  setRepositoryUrl("");
                  setActiveTab("overview");
                }}
              >
                {analysisComplete ? "Start New Analysis" : "Analyze Repository"}
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* Navigation - Only show when analysis is complete */}
      {analysisComplete && (
        <nav className="notion-nav">
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <div className="flex space-x-8">
              {tabs.map((tab) => {
                const Icon = tab.icon;
                return (
                  <button
                    key={tab.id}
                    onClick={() => setActiveTab(tab.id)}
                    className={`notion-tab flex items-center px-1 text-sm font-medium ${
                      activeTab === tab.id ? "active" : ""
                    }`}
                  >
                    <Icon className="h-4 w-4 mr-2" />
                    {tab.name}
                  </button>
                );
              })}
            </div>
          </div>
        </nav>
      )}

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {!analysisComplete ? (
          /* Hero Section with Repository Input */
          <div className="notion-hero">
            <h1 className="notion-hero-title text-lg">
              Analyze Any GitHub Repository
            </h1>
            <p className="notion-hero-subtitle">
              Get instant insights into project architecture, dependencies,
              database schemas, and microservice relationships
            </p>

            {/* Repository Input */}
            <div className="max-w-2xl mx-auto mb-8">
              <div className="notion-input-container">
                <div className="notion-input-group">
                  <input
                    type="url"
                    className="notion-input"
                    placeholder="https://github.com/owner/repository"
                    value={repositoryUrl}
                    onChange={(e) => setRepositoryUrl(e.target.value)}
                    onKeyPress={(e) =>
                      e.key === "Enter" && handleAnalyzeRepository()
                    }
                    disabled={isAnalyzing}
                  />
                  <button
                    className="notion-input-button"
                    onClick={handleAnalyzeRepository}
                    disabled={isAnalyzing || !repositoryUrl.trim()}
                  >
                    {isAnalyzing ? "Analyzing..." : "Analyze"}
                  </button>
                </div>
              </div>

              {/* Example URLs */}
              <div className="text-center mt-4">
                <p className="text-gray-500 text-sm mb-2">
                  Try these examples:
                </p>
                <div className="flex flex-wrap justify-center gap-2">
                  {[
                    "https://github.com/microsoft/vscode",
                    "https://github.com/facebook/react",
                    "https://github.com/kubernetes/kubernetes",
                  ].map((url) => (
                    <button
                      key={url}
                      onClick={() => setRepositoryUrl(url)}
                      className="text-xs text-gray-500 hover:text-blue-600 transition-colors px-3 py-2 rounded-lg bg-gray-100 hover:bg-blue-50 border border-gray-200 hover:border-blue-300"
                      disabled={isAnalyzing}
                    >
                      {url.split("/").slice(-2).join("/")}
                    </button>
                  ))}
                </div>
              </div>
            </div>

            {/* Features Grid */}
            <div className="notion-features">
              {features.map((feature, index) => {
                const Icon = feature.icon;
                return (
                  <div key={index} className="notion-feature">
                    <Icon className="notion-feature-icon" />
                    <h3 className="notion-feature-title">{feature.title}</h3>
                    <p className="notion-feature-description">
                      {feature.description}
                    </p>
                  </div>
                );
              })}
            </div>
          </div>
        ) : (
          /* Dashboard Content */
          <div className="notion-container mx-4 mt-6 p-8">
            {activeTab === "overview" && (
              <div className="space-y-6">
                {/* Project Summary */}
                <div className="card">
                  <div className="card-header">
                    <h2 className="text-gray-800 font-semibold">
                      Project Analysis Summary
                    </h2>
                  </div>
                  <div className="card-content">
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                      <div className="text-center">
                        <div className="text-3xl font-bold text-gray-800">
                          {mockData.project.type}
                        </div>
                        <div className="text-sm text-gray-500">
                          Project Type
                        </div>
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
                        <Server className="h-8 w-8 text-blue-600" />
                        <div className="ml-4">
                          <div className="text-sm font-medium text-gray-500">
                            Architecture
                          </div>
                          <div className="text-lg font-semibold text-gray-800">
                            {mockData.project.architecture}
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div className="card">
                    <div className="card-content">
                      <div className="flex items-center">
                        <Database className="h-8 w-8 text-green-600" />
                        <div className="ml-4">
                          <div className="text-sm font-medium text-gray-500">
                            Database Tables
                          </div>
                          <div className="text-lg font-semibold text-gray-800">
                            {mockData.database.tables.length}
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div className="card">
                    <div className="card-content">
                      <div className="flex items-center">
                        <GitBranch className="h-8 w-8 text-purple-600" />
                        <div className="ml-4">
                          <div className="text-sm font-medium text-gray-500">
                            Dependencies
                          </div>
                          <div className="text-lg font-semibold text-gray-800">
                            {mockData.database.relationships}
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>

                {/* Tech Stack */}
                <div className="card">
                  <div className="card-header">
                    <h3 className="text-gray-800 font-semibold">
                      Technology Stack
                    </h3>
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

            {activeTab === "services" && (
              <div className="space-y-6">
                <div className="card">
                  <div className="card-header">
                    <h2 className="text-gray-800 font-semibold">
                      Discovered Services
                    </h2>
                  </div>
                  <div className="card-content">
                    <div className="space-y-4">
                      {mockData.services.map((service) => (
                        <div
                          key={service.name}
                          className="notion-service-item flex items-center justify-between p-4"
                        >
                          <div className="flex items-center">
                            <Server className="h-6 w-6 text-blue-600 mr-3" />
                            <div>
                              <div className="font-medium text-gray-800">
                                {service.name}
                              </div>
                              <div className="text-sm text-gray-500">
                                Port: {service.port}
                              </div>
                            </div>
                          </div>
                          <span
                            className={`badge ${
                              service.type === "gRPC"
                                ? "badge-warning"
                                : "badge-primary"
                            }`}
                          >
                            {service.type}
                          </span>
                        </div>
                      ))}
                    </div>
                  </div>
                </div>
              </div>
            )}

            {activeTab === "database" && (
              <div className="space-y-6">
                <div className="card">
                  <div className="card-header">
                    <h2 className="text-gray-800 font-semibold">
                      Database Schema
                    </h2>
                  </div>
                  <div className="card-content">
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                      {mockData.database.tables.map((table) => (
                        <div
                          key={table}
                          className="notion-db-item p-4 text-center"
                        >
                          <Database className="h-6 w-6 text-green-600 mx-auto mb-2" />
                          <div className="font-medium text-gray-800">
                            {table}
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                </div>
              </div>
            )}

            {/* Placeholder for other tabs */}
            {!["overview", "services", "database"].includes(activeTab) && (
              <div className="card">
                <div className="card-content">
                  <div className="text-center py-12">
                    <Code2 className="h-12 w-12 text-gray-400 mx-auto mb-4" />
                    <h3 className="text-lg font-medium text-gray-800 mb-2">
                      {tabs.find((t) => t.id === activeTab)?.name} Coming Soon
                    </h3>
                    <p className="text-gray-500">
                      This section will display detailed {activeTab}{" "}
                      information.
                    </p>
                  </div>
                </div>
              </div>
            )}
          </div>
        )}
      </main>
    </div>
  );
}

export default App;
