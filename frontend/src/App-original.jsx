import { useState, useEffect } from "react";
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
import { repositoryAPI } from "./utils/api";

function App() {
  const [activeTab, setActiveTab] = useState("overview");
  const [repositoryUrl, setRepositoryUrl] = useState("");
  const [isAnalyzing, setIsAnalyzing] = useState(false);
  const [analysisComplete, setAnalysisComplete] = useState(false);
  const [analysisResults, setAnalysisResults] = useState(null);
  const [error, setError] = useState(null);
  const [analysisProgress, setAnalysisProgress] = useState(0);
  const [analysisStage, setAnalysisStage] = useState("");
  const [showTokenInput, setShowTokenInput] = useState(false); // Change to true to test UI
  const [githubToken, setGithubToken] = useState("");
  const [analysisCache, setAnalysisCache] = useState(new Map());

  // Effect to track showTokenInput state changes
  useEffect(() => {
    if (showTokenInput) {
      console.log("🔐 Authentication required - showing token input");
    }
  }, [showTokenInput]);

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

  const handleAnalyzeRepository = async (useToken = false) => {
    if (!repositoryUrl.trim()) return;

    // Create cache key (include token status for private repos)
    const cacheKey = `${repositoryUrl}${useToken ? '_with_token' : ''}`;
    
    // Check cache first
    if (analysisCache.has(cacheKey)) {
      console.log('📋 Using cached analysis results');
      const cachedResults = analysisCache.get(cacheKey);
      setAnalysisResults(cachedResults);
      setAnalysisComplete(true);
      setShowTokenInput(false);
      return;
    }

    setIsAnalyzing(true);
    setError(null);
    setAnalysisProgress(0);
    setAnalysisStage("");

    // Progress simulation to keep user informed
    const simulateProgress = () => {
      const stages = [
        { stage: "Cloning repository...", progress: 10 },
        { stage: "Scanning file structure...", progress: 25 },
        { stage: "Detecting project type...", progress: 40 },
        { stage: "Analyzing microservices...", progress: 60 },
        { stage: "Discovering databases...", progress: 75 },
        { stage: "Mapping relationships...", progress: 90 },
        { stage: "Finalizing analysis...", progress: 95 },
      ];

      let currentStage = 0;
      const progressInterval = setInterval(() => {
        if (currentStage < stages.length && isAnalyzing) {
          setAnalysisStage(stages[currentStage].stage);
          setAnalysisProgress(stages[currentStage].progress);
          currentStage++;
        } else {
          clearInterval(progressInterval);
        }
      }, 8000); // Update every 8 seconds

      return progressInterval;
    };

    const progressInterval = simulateProgress();

    try {
      // Send GitHub URL to the API endpoint
      console.log(
        `🚀 Analyzing repository: ${repositoryUrl}${
          useToken ? " (with token)" : ""
        }`
      );
      setAnalysisStage("Initializing analysis...");
      setAnalysisProgress(5);

      const token = useToken ? githubToken : null;
      const response = await repositoryAPI.analyzeRepository(
        repositoryUrl,
        token
      );

      console.log("✅ Analysis response:", response);
      setAnalysisResults(response);
      setAnalysisComplete(true);
      setAnalysisProgress(100);
      setAnalysisStage("Analysis completed!");
      setShowTokenInput(false); // Hide token input on success
      
      // Cache the successful result
      setAnalysisCache(prevCache => new Map(prevCache.set(cacheKey, response)));
    } catch (err) {
      console.error("❌ Analysis failed:", err);
      clearInterval(progressInterval);

      // Handle different types of errors
      if (
        err.response?.status === 401 &&
        err.response?.data?.status === "auth_required"
      ) {
        // Repository requires authentication
        setShowTokenInput(true);
        setError(
          "This repository appears to be private. Please provide a GitHub Personal Access Token below."
        );
      } else if (
        err.code === "ECONNABORTED" ||
        err.message.includes("timeout")
      ) {
        setError(
          "Analysis timed out. The repository may be too large or complex. Please try with a smaller repository."
        );
      } else if (err.response?.status === 408) {
        setError(
          "Analysis timed out on the server after 30 minutes. The repository may be too large or complex for analysis."
        );
      } else if (err.response?.data?.error) {
        setError(err.response.data.error);
      } else {
        setError(
          err.message ||
            "Analysis failed. Please check the repository URL and try again."
        );
      }

      setAnalysisProgress(0);
      setAnalysisStage("");
    } finally {
      clearInterval(progressInterval);
      setIsAnalyzing(false);
    }
  };

  const handleTokenSubmit = () => {
    if (!githubToken.trim()) {
      setError("Please enter a valid GitHub Personal Access Token.");
      return;
    }
    handleAnalyzeRepository(true);
  };

  // Helper function to get analysis data safely
  const getAnalysisData = () => {
    if (!analysisResults || !analysisResults.results) {
      return null;
    }
    
    const results = analysisResults.results;
    return {
      projectSummary: results.project_summary || {},
      projectType: results.project_type || {},
      services: results.services || [],
      relationships: results.relationships || [],
      databaseSchema: results.database_schema || null,
      fileSummaries: results.file_summaries || {},
      folderSummaries: results.folder_summaries || {},
      stats: results.stats || {}
    };
  };

  // Check if analysis failed or returned insufficient data
  const isAnalysisIncomplete = () => {
    if (!analysisResults) return true;
    
    const data = getAnalysisData();
    if (!data) return true;
    
    // Check if we have at least some meaningful data
    const hasProjectInfo = data.projectType.primary_type || data.projectSummary.purpose;
    const hasFileStats = data.stats.total_files > 0;
    const hasLanguages = data.projectSummary.languages && Object.keys(data.projectSummary.languages).length > 0;
    
    return !hasProjectInfo && !hasFileStats && !hasLanguages;
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
              <p className="ml-3 text-lg ml-5 font-semibold text-gray-800">
                Repository Analyzer
              </p>
            </div>
            <div className="flex items-center space-x-4">
              <span className="badge badge-success">Active</span>
              <button
                className="btn btn-primary"
                onClick={() => {
                  setAnalysisComplete(false);
                  setAnalysisResults(null);
                  setError(null);
                  setAnalysisProgress(0);
                  setAnalysisStage("");
                  setShowTokenInput(false);
                  setGithubToken("");
                  setRepositoryUrl("");
                  setActiveTab("overview");
                  // Clear cache is optional - keeping it for repeated analysis of same repo
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
                    {isAnalyzing ? (
                      <div className="flex items-center">
                        <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
                        Analyzing...
                      </div>
                    ) : (
                      "Analyze"
                    )}
                  </button>
                </div>
              </div>

              {/* Progress Indicator */}
              {isAnalyzing && (
                <div className="mt-6 p-4 bg-blue-50 border border-blue-200 rounded-lg">
                  <div className="flex items-center justify-between mb-2">
                    <div className="text-sm font-medium text-blue-800">
                      Analysis in Progress
                    </div>
                    <div className="text-sm text-blue-600">
                      {analysisProgress}%
                    </div>
                  </div>

                  {/* Progress Bar */}
                  <div className="w-full bg-blue-200 rounded-full h-2 mb-3">
                    <div
                      className="bg-blue-600 h-2 rounded-full transition-all duration-1000 ease-out"
                      style={{ width: `${analysisProgress}%` }}
                    ></div>
                  </div>

                  {/* Current Stage */}
                  {analysisStage && (
                    <div className="text-sm text-blue-700 flex items-center">
                      <div className="animate-pulse w-2 h-2 bg-blue-500 rounded-full mr-2"></div>
                      {analysisStage}
                    </div>
                  )}

                  {/* Time Estimate */}
                  <div className="mt-2 text-xs text-blue-600">
                    This may take several minutes for large repositories. Please
                    keep this tab open.
                  </div>
                </div>
              )}

              {/* GitHub Token Input */}
              {showTokenInput && (
                <div className="mt-6 p-6 bg-yellow-50 border border-yellow-200 rounded-lg">
                  <div className="flex items-start">
                    <div className="flex-shrink-0">
                      <svg
                        className="h-5 w-5 text-yellow-400"
                        viewBox="0 0 20 20"
                        fill="currentColor"
                      >
                        <path
                          fillRule="evenodd"
                          d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
                          clipRule="evenodd"
                        />
                      </svg>
                    </div>
                    <div className="ml-3 flex-1">
                      <h3 className="text-sm font-medium text-yellow-800">
                        Authentication Required
                      </h3>
                      <div className="mt-2 text-sm text-yellow-700">
                        <p>
                          This repository is private and requires
                          authentication.
                        </p>
                      </div>

                      <div className="mt-4">
                        <label
                          htmlFor="github-token"
                          className="block text-sm font-medium text-yellow-800 mb-2"
                        >
                          GitHub Personal Access Token
                        </label>
                        <div className="flex gap-2">
                          <input
                            type="password"
                            id="github-token"
                            className="flex-1 px-3 py-2 border border-yellow-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                            placeholder="ghp_xxxxxxxxxxxxxxxxxxxx"
                            value={githubToken}
                            onChange={(e) => setGithubToken(e.target.value)}
                            onKeyPress={(e) =>
                              e.key === "Enter" && handleTokenSubmit()
                            }
                            disabled={isAnalyzing}
                          />
                          <button
                            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                            onClick={handleTokenSubmit}
                            disabled={isAnalyzing || !githubToken.trim()}
                          >
                            {isAnalyzing ? "Analyzing..." : "Analyze"}
                          </button>
                        </div>

                        <div className="mt-3 text-xs text-yellow-600">
                          <p className="mb-1">
                            📝{" "}
                            <strong>
                              How to create a Personal Access Token:
                            </strong>
                          </p>
                          <ol className="list-decimal list-inside space-y-1 ml-4">
                            <li>
                              Go to GitHub Settings → Developer settings →
                              Personal access tokens
                            </li>
                            <li>
                              Click "Generate new token" → "Generate new token
                              (classic)"
                            </li>
                            <li>
                              Select scopes:{" "}
                              <code className="bg-yellow-100 px-1 rounded">
                                repo
                              </code>{" "}
                              (for private repos)
                            </li>
                            <li>Copy the token and paste it above</li>
                          </ol>
                          <p className="mt-2 text-yellow-500">
                            🔒 Your token is only used for this analysis and is
                            not stored.
                          </p>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              )}

              {/* Error Display */}
              {error && (
                <div className="mt-4 p-4 bg-red-50 border border-red-200 rounded-lg">
                  <div className="flex">
                    <div className="flex-shrink-0">
                      <svg
                        className="h-5 w-5 text-red-400"
                        viewBox="0 0 20 20"
                        fill="currentColor"
                      >
                        <path
                          fillRule="evenodd"
                          d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                          clipRule="evenodd"
                        />
                      </svg>
                    </div>
                    <div className="ml-3">
                      <h3 className="text-sm font-medium text-red-800">
                        Analysis Failed
                      </h3>
                      <div className="mt-2 text-sm text-red-700">
                        <p>{error}</p>
                      </div>
                    </div>
                  </div>
                </div>
              )}

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
            {/* Analysis Failed Error */}
            {analysisComplete && isAnalysisIncomplete() ? (
              <div className="card bg-red-50 border-red-200">
                <div className="card-content">
                  <div className="text-center py-12">
                    <div className="text-red-400 mb-4">
                      <svg className="h-16 w-16 mx-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                    </div>
                    <h3 className="text-xl font-semibold text-red-800 mb-2">
                      Analysis Failed or Incomplete
                    </h3>
                    <p className="text-red-600 mb-6 max-w-md mx-auto">
                      The analysis process couldn't extract sufficient information from this repository. 
                      This might happen if the repository is empty, private without access token, or contains 
                      unsupported project types.
                    </p>
                    <div className="space-y-2 text-sm text-red-500">
                      <p><strong>Possible solutions:</strong></p>
                      <ul className="list-disc list-inside space-y-1 max-w-md mx-auto">
                        <li>Verify the repository URL is correct and accessible</li>
                        <li>For private repositories, ensure you provided a valid GitHub token</li>
                        <li>Check that the repository contains source code files</li>
                        <li>Try analyzing a different repository</li>
                      </ul>
                    </div>
                    <button
                      onClick={() => {
                        setAnalysisComplete(false);
                        setAnalysisResults(null);
                        setError(null);
                        setAnalysisProgress(0);
                        setAnalysisStage("");
                        setShowTokenInput(false);
                        setGithubToken("");
                        setRepositoryUrl("");
                        setActiveTab("overview");
                      }}
                      className="mt-6 bg-red-600 hover:bg-red-700 text-white font-semibold py-2 px-6 rounded-lg transition-colors"
                    >
                      Try Another Repository
                    </button>
                  </div>
                </div>
              </div>
            ) : (
              <>
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
                          {(() => {
                            const data = getAnalysisData();
                            return data?.projectType?.primary_type || "Unknown";
                          })()}
                        </div>
                        <div className="text-sm text-gray-500">
                          Project Type
                        </div>
                      </div>
                      <div className="text-center">
                        <div className="text-3xl font-bold text-green-600">
                          {(() => {
                            const data = getAnalysisData();
                            return data?.projectType?.confidence || 0;
                          })()}%
                        </div>
                        <div className="text-sm text-gray-500">Confidence</div>
                      </div>
                      <div className="text-center">
                        <div className="text-3xl font-bold text-blue-600">
                          {(() => {
                            const data = getAnalysisData();
                            return data?.services?.length || 0;
                          })()}
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
                            {(() => {
                              const data = getAnalysisData();
                              return data?.projectSummary?.detailed_analysis?.architecture || "Unknown";
                            })()}
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
                            {(() => {
                              const data = getAnalysisData();
                              const databaseSchema = data?.databaseSchema;
                              return databaseSchema?.tables ? Object.keys(databaseSchema.tables).length : 0;
                            })()}
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
                            {(() => {
                              const data = getAnalysisData();
                              return data?.relationships?.length || 0;
                            })()}
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
                      {(() => {
                        const data = getAnalysisData();
                        const languages = data?.projectSummary?.languages || {};
                        const languageEntries = Object.entries(languages);
                        
                        if (languageEntries.length === 0) {
                          return (
                            <div className="text-center py-4 text-gray-500 w-full">
                              <Code2 className="h-8 w-8 mx-auto mb-2 opacity-50" />
                              <p>No programming languages detected</p>
                            </div>
                          );
                        }
                        
                        return languageEntries.map(([lang, lines]) => (
                          <span key={lang} className="badge badge-primary">
                            {lang} ({lines} lines)
                          </span>
                        ));
                      })()}
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
                      {(() => {
                        const data = getAnalysisData();
                        const services = data?.services || [];
                        
                        if (services.length === 0) {
                          return (
                            <div className="text-center py-8 text-gray-500">
                              <Server className="h-12 w-12 mx-auto mb-4 opacity-50" />
                              <p>No microservices detected in this repository</p>
                            </div>
                          );
                        }
                        
                        return services.map((service) => (
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
                                  {service.port && `Port: ${service.port}`}
                                </div>
                                {service.description && (
                                  <div className="text-xs text-gray-400 mt-1">
                                    {service.description}
                                  </div>
                                )}
                              </div>
                            </div>
                            <span
                              className={`badge ${
                                service.api_type === "grpc"
                                  ? "badge-warning"
                                  : "badge-primary"
                              }`}
                            >
                              {service.api_type}
                            </span>
                          </div>
                        ));
                      })()}
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
                      {(() => {
                        const data = getAnalysisData();
                        const databaseSchema = data?.databaseSchema;
                        
                        if (!databaseSchema || !databaseSchema.tables || Object.keys(databaseSchema.tables).length === 0) {
                          return (
                            <div className="col-span-full text-center py-8 text-gray-500">
                              <Database className="h-12 w-12 mx-auto mb-4 opacity-50" />
                              <p>No database schema detected in this repository</p>
                            </div>
                          );
                        }
                        
                        return Object.entries(databaseSchema.tables).map(([tableName, tableInfo]) => (
                          <div
                            key={tableName}
                            className="notion-db-item p-4 text-center"
                          >
                            <Database className="h-6 w-6 text-green-600 mx-auto mb-2" />
                            <div className="font-medium text-gray-800">
                              {tableName}
                            </div>
                            <div className="text-xs text-gray-500 mt-1">
                              {Object.keys(tableInfo.columns || {}).length} columns
                            </div>
                          </div>
                        ));
                      })()}
                    </div>
                  </div>
                </div>
              </div>
            )}

            {activeTab === "relationships" && (
              <div className="space-y-6">
                <div className="card">
                  <div className="card-header">
                    <h2 className="text-gray-800 font-semibold">
                      Service Dependencies
                    </h2>
                  </div>
                  <div className="card-content">
                    {(() => {
                      const data = getAnalysisData();
                      const relationships = data?.relationships || [];
                      
                      if (relationships.length === 0) {
                        return (
                          <div className="text-center py-8 text-gray-500">
                            <GitBranch className="h-12 w-12 mx-auto mb-4 opacity-50" />
                            <p>No service dependencies detected</p>
                          </div>
                        );
                      }
                      
                      return (
                        <div className="space-y-4">
                          {relationships.map((rel, index) => (
                            <div key={index} className="notion-service-item p-4 border-l-4 border-blue-500">
                              <div className="flex items-center justify-between">
                                <div className="flex items-center">
                                  <GitBranch className="h-5 w-5 text-blue-600 mr-3" />
                                  <span className="font-medium text-gray-800">{rel.from}</span>
                                  <span className="mx-2 text-gray-400">→</span>
                                  <span className="font-medium text-gray-800">{rel.to}</span>
                                </div>
                                <div className="text-right">
                                  <span className="badge badge-secondary">{rel.evidence_type}</span>
                                  {rel.confidence && (
                                    <div className="text-xs text-gray-500 mt-1">
                                      {rel.confidence}% confidence
                                    </div>
                                  )}
                                </div>
                              </div>
                              {rel.evidence_path && (
                                <div className="mt-2 text-xs text-gray-400">
                                  Evidence: {rel.evidence_path}
                                </div>
                              )}
                            </div>
                          ))}
                        </div>
                      );
                    })()}
                  </div>
                </div>
              </div>
            )}

            {activeTab === "files" && (
              <div className="space-y-6">
                <div className="card">
                  <div className="card-header">
                    <h2 className="text-gray-800 font-semibold">
                      File Analysis
                    </h2>
                  </div>
                  <div className="card-content">
                    {(() => {
                      const data = getAnalysisData();
                      const stats = data?.stats || {};
                      const fileSummaries = data?.fileSummaries || {};
                      
                      return (
                        <div className="space-y-6">
                          {/* File Statistics */}
                          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                            <div className="text-center p-4 bg-gray-50 rounded-lg">
                              <div className="text-2xl font-bold text-gray-800">
                                {stats.total_files || 0}
                              </div>
                              <div className="text-sm text-gray-500">Total Files</div>
                            </div>
                            <div className="text-center p-4 bg-gray-50 rounded-lg">
                              <div className="text-2xl font-bold text-gray-800">
                                {(stats.total_size_mb || 0).toFixed(1)} MB
                              </div>
                              <div className="text-sm text-gray-500">Total Size</div>
                            </div>
                            <div className="text-center p-4 bg-gray-50 rounded-lg">
                              <div className="text-2xl font-bold text-gray-800">
                                {Object.keys(stats.extensions || {}).length}
                              </div>
                              <div className="text-sm text-gray-500">File Types</div>
                            </div>
                          </div>
                          
                          {/* File Extensions */}
                          {stats.extensions && Object.keys(stats.extensions).length > 0 && (
                            <div>
                              <h3 className="text-lg font-semibold text-gray-800 mb-3">File Extensions</h3>
                              <div className="flex flex-wrap gap-2">
                                {Object.entries(stats.extensions).map(([ext, count]) => (
                                  <span key={ext} className="badge badge-secondary">
                                    {ext}: {count} files
                                  </span>
                                ))}
                              </div>
                            </div>
                          )}
                        </div>
                      );
                    })()}
                  </div>
                </div>
              </div>
            )}

            {activeTab === "analysis" && (
              <div className="space-y-6">
                <div className="card">
                  <div className="card-header">
                    <h2 className="text-gray-800 font-semibold">
                      Detailed Analysis
                    </h2>
                  </div>
                  <div className="card-content">
                    {(() => {
                      const data = getAnalysisData();
                      const projectSummary = data?.projectSummary || {};
                      
                      return (
                        <div className="space-y-6">
                          {/* Purpose */}
                          {projectSummary.purpose && (
                            <div>
                              <h3 className="text-lg font-semibold text-gray-800 mb-2">Purpose</h3>
                              <p className="text-gray-700">{projectSummary.purpose}</p>
                            </div>
                          )}
                          
                          {/* Architecture */}
                          {projectSummary.architecture && (
                            <div>
                              <h3 className="text-lg font-semibold text-gray-800 mb-2">Architecture</h3>
                              <p className="text-gray-700">{projectSummary.architecture}</p>
                            </div>
                          )}
                          
                          {/* Data Models */}
                          {projectSummary.data_models && projectSummary.data_models.length > 0 && (
                            <div>
                              <h3 className="text-lg font-semibold text-gray-800 mb-2">Data Models</h3>
                              <div className="flex flex-wrap gap-2">
                                {projectSummary.data_models.map((model) => (
                                  <span key={model} className="badge badge-primary">{model}</span>
                                ))}
                              </div>
                            </div>
                          )}
                          
                          {/* External Services */}
                          {projectSummary.external_services && projectSummary.external_services.length > 0 && (
                            <div>
                              <h3 className="text-lg font-semibold text-gray-800 mb-2">External Services</h3>
                              <div className="flex flex-wrap gap-2">
                                {projectSummary.external_services.map((service) => (
                                  <span key={service} className="badge badge-secondary">{service}</span>
                                ))}
                              </div>
                            </div>
                          )}
                        </div>
                      );
                    })()}
                  </div>
                </div>
              </div>
            )}

            {/* Placeholder for unimplemented tabs */}
            {!["overview", "services", "database", "relationships", "files", "analysis"].includes(activeTab) && (
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
              </>
            )}
          </div>
        )}
      </main>
    </div>
  );
}

export default App;
