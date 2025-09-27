import { useState, useEffect } from "react";
import {
  FileText, // Still needed for DatabaseTab migration display
  GitBranch,
  Database,
  Server,
  Code2,
  BarChart3,
  Github,
  Zap,
  Eye,
  Download,
  AlertCircle,
  Table,
  Key,
  Link,
  ChevronDown,
  ChevronRight,
  Brain,
  Shield,
} from "lucide-react";
import { repositoryAPI } from "./utils/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Separator } from "@/components/ui/separator";
import ZoomableMermaid from "./components/ZoomableMermaid";

function App() {
  const [activeTab, setActiveTab] = useState("overview");
  const [repositoryUrl, setRepositoryUrl] = useState("");
  const [isAnalyzing, setIsAnalyzing] = useState(false);
  const [analysisComplete, setAnalysisComplete] = useState(false);
  const [analysisResults, setAnalysisResults] = useState(null);
  const [error, setError] = useState(null);
  const [analysisProgress, setAnalysisProgress] = useState(0);
  const [analysisStage, setAnalysisStage] = useState("");
  const [showTokenInput, setShowTokenInput] = useState(false);
  const [githubToken, setGithubToken] = useState("");
  const [analysisCache, setAnalysisCache] = useState(new Map());

  // Effect to track showTokenInput state changes
  useEffect(() => {
    if (showTokenInput) {
      console.log("🔐 Authentication required - showing token input");
    }
  }, [showTokenInput]);

  const tabs = [
    { id: "overview", name: "Overview", icon: BarChart3 },
    { id: "analysis", name: "Analysis", icon: Code2 },
    { id: "services", name: "Services", icon: Server },
    { id: "relationships", name: "Dependencies", icon: GitBranch },
    { id: "database", name: "Database", icon: Database },
    { id: "secrets", name: "Secrets", icon: Shield },
    { id: "questions", name: "Helpful Questions", icon: Eye },
    // Files tab removed - not needed for architectural understanding and improves performance
  ];

  const handleAnalyzeRepository = async (useToken = false) => {
    if (!repositoryUrl.trim()) return;

    // Create cache key (include token status for private repos)
    const cacheKey = `${repositoryUrl}${useToken ? "_with_token" : ""}`;

    // Check cache first
    if (analysisCache.has(cacheKey)) {
      console.log("📋 Using cached analysis results");
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

    try {
      console.log(
        `🚀 Starting streaming analysis: ${repositoryUrl}${
          useToken ? " (with token)" : ""
        }`
      );

      const token = useToken ? githubToken : null;

      // Use streaming API for real-time progress updates
      repositoryAPI.analyzeRepositoryStream(
        repositoryUrl,
        token,
        // onProgress callback
        (stage, message, progress, data) => {
          setAnalysisStage(stage);
          setAnalysisProgress(progress);

          // Only update partial data for specific data events, not completion
          if (data && stage !== "🎉 Analysis complete!") {
            setAnalysisResults((prevResults) => ({
              ...prevResults,
              ...data,
            }));
          }

          console.log(`📊 Progress: ${progress}% - ${stage}: ${message}`);
        },
        // onComplete callback
        (finalResults) => {
          console.log("✅ Analysis completed:", finalResults);
          console.log(
            "🗄️ Database schema in results:",
            finalResults?.database_schema
          );
          setAnalysisResults(finalResults);
          setAnalysisComplete(true);
          setAnalysisProgress(100);
          setAnalysisStage("🎉 Analysis complete!");
          setShowTokenInput(false);
          setIsAnalyzing(false);

          // Cache the successful result
          setAnalysisCache(
            (prevCache) => new Map(prevCache.set(cacheKey, finalResults))
          );
        },
        // onError callback
        (errorMessage) => {
          console.error("❌ Streaming analysis failed:", errorMessage);
          setIsAnalyzing(false);

          // Handle different types of errors
          if (
            errorMessage.includes("auth_required") ||
            errorMessage.includes("private")
          ) {
            setShowTokenInput(true);
            setError(
              "This repository appears to be private. Please provide a GitHub Personal Access Token below."
            );
          } else if (errorMessage.includes("timeout")) {
            setError(
              "Analysis timed out. The repository may be too large or complex. Please try with a smaller repository."
            );
          } else {
            setError(
              errorMessage ||
                "Analysis failed. Please check the repository URL and try again."
            );
          }

          setAnalysisProgress(0);
          setAnalysisStage("");
        }
      );
    } catch (err) {
      console.error("❌ Failed to start streaming analysis:", err);
      setIsAnalyzing(false);
      setError("Failed to start analysis. Please try again.");
      setAnalysisProgress(0);
      setAnalysisStage("");
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
    if (!analysisResults) {
      console.log("🚫 No analysisResults available");
      return null;
    }

    // Handle both streaming API (direct data) and regular API (wrapped in results)
    const results = analysisResults.results || analysisResults;

    if (!results) {
      console.log("🚫 No results in analysisResults:", analysisResults);
      return null;
    }

    const mappedData = {
      projectSummary: results.project_summary || {},
      projectType: results.project_type || {},
      services: results.services || [],
      relationships: results.relationships || [],
      databaseSchema: results.database_schema || null,
      projectSecrets: results.project_secrets || null,
      helpfulQuestions: results.helpful_questions || [],
      fileSummaries: results.file_summaries || {},
      folderSummaries: results.folder_summaries || {},
      stats: results.stats || {},
    };

    console.log(
      "🗄️ DatabaseSchema from getAnalysisData:",
      mappedData.databaseSchema
    );
    console.log(
      "📊 ProjectType confidence from getAnalysisData:",
      mappedData.projectType?.confidence
    );
    return mappedData;
  };

  // Check if analysis failed or returned insufficient data
  const isAnalysisIncomplete = () => {
    if (!analysisResults) return true;

    const data = getAnalysisData();
    if (!data) return true;

    // Check if we have at least some meaningful data
    const hasProjectInfo =
      data.projectType.primary_type || data.projectSummary.purpose;
    const hasFileStats = data.stats.total_files > 0;
    const hasLanguages =
      data.projectSummary.languages &&
      Object.keys(data.projectSummary.languages).length > 0;

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
      icon: Download,
      title: "Export Ready",
      description:
        "Export comprehensive reports in multiple formats for documentation.",
    },
  ];

  return (
    <div
      className="min-h-screen"
      style={{ backgroundColor: "hsl(var(--slate-50))" }}
    >
      {/* Header */}
      <header
        className="border-b backdrop-blur supports-[backdrop-filter]:backdrop-blur"
        style={{
          backgroundColor: "hsl(var(--slate-100))",
          borderColor: "hsl(var(--slate-200))",
        }}
      >
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex h-16 items-center">
            <div
              className="flex items-center gap-3 cursor-pointer hover:opacity-80 transition-opacity"
              onClick={() => {
                setAnalysisResults(null);
                setAnalysisComplete(false);
                setAnalysisRunning(false);
                setAnalysisProgress(0);
                setRepoUrl("");
                setGithubToken("");
                setActiveTab("overview");
              }}
            >
              <div
                className="p-2 rounded-xl"
                style={{ backgroundColor: "hsl(var(--slate-800))" }}
              >
                <Github className="h-5 w-5 text-white" />
              </div>
              <span
                className="text-xl font-semibold tracking-tight"
                style={{ color: "hsl(var(--slate-800))" }}
              >
                Repository Analyzer
              </span>
            </div>
          </div>
        </div>
      </header>

      {/* Navigation - Only show when analysis is complete */}
      {analysisComplete && !isAnalysisIncomplete() && (
        <nav
          className="border-b"
          style={{
            backgroundColor: "hsl(var(--slate-100))",
            borderColor: "hsl(var(--slate-200))",
          }}
        >
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-3">
            <Tabs
              value={activeTab}
              onValueChange={setActiveTab}
              className="w-full"
            >
              <div className="overflow-x-auto">
                <TabsList
                  className="flex min-w-fit w-max mx-auto"
                  style={{ backgroundColor: "hsl(var(--slate-200))" }}
                >
                  {/* Optimized for 7 tabs (Files tab removed for performance) */}
                  {tabs.map((tab) => (
                    <TabsTrigger
                      key={tab.id}
                      value={tab.id}
                      className="flex items-center gap-1 sm:gap-2 data-[state=active]:text-white text-xs sm:text-sm px-2 sm:px-3 py-2 flex-shrink-0 whitespace-nowrap min-w-max"
                      style={{
                        color:
                          activeTab === tab.id
                            ? "white"
                            : "hsl(var(--slate-600))",
                        backgroundColor:
                          activeTab === tab.id
                            ? "hsl(var(--slate-800))"
                            : "transparent",
                      }}
                    >
                      <tab.icon className="h-3 w-3 sm:h-4 sm:w-4 flex-shrink-0" />
                      <span className="hidden sm:inline truncate">
                        {tab.name}
                      </span>
                      <span className="sm:hidden truncate text-xs">
                        {tab.name.split(" ")[0]}
                      </span>
                    </TabsTrigger>
                  ))}
                </TabsList>
              </div>
            </Tabs>
          </div>
        </nav>
      )}

      {/* Main Content */}
      <main className="flex-1">
        {!analysisComplete ? (
          /* Landing/Analysis Page */
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16">
            <div className="text-center space-y-12">
              <div className="space-y-6">
                <div
                  className="inline-flex items-center px-4 py-2 rounded-full text-sm font-medium"
                  style={{
                    backgroundColor: "hsl(var(--slate-200))",
                    color: "hsl(var(--slate-700))",
                  }}
                >
                  ✨ AI-Powered Repository Analysis
                </div>
                <h1
                  className="text-4xl font-bold tracking-tight sm:text-6xl"
                  style={{ color: "hsl(var(--slate-800))" }}
                >
                  Analyze Your{" "}
                  <span style={{ color: "hsl(var(--slate-600))" }}>
                    Repository
                  </span>
                </h1>
                <p
                  className="text-xl max-w-2xl mx-auto"
                  style={{ color: "hsl(var(--slate-600))" }}
                >
                  Get instant insights into your codebase architecture,
                  dependencies, and project structure with AI-powered analysis.
                </p>
              </div>

              {/* Input Section */}
              <Card
                className="max-w-2xl mx-auto shadow-lg border-0"
                style={{
                  backgroundColor: "white",
                  boxShadow:
                    "0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05)",
                }}
              >
                <CardHeader
                  className="pb-4"
                  style={{ backgroundColor: "hsl(var(--slate-50))" }}
                >
                  <CardTitle
                    className="text-2xl"
                    style={{ color: "hsl(var(--slate-800))" }}
                  >
                    Start Analysis
                  </CardTitle>
                  <CardDescription style={{ color: "hsl(var(--slate-600))" }}>
                    Enter a GitHub repository URL to begin comprehensive
                    analysis
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4 pt-6">
                  <div className="flex gap-2">
                    <Input
                      type="text"
                      placeholder="https://github.com/username/repository"
                      value={repositoryUrl}
                      onChange={(e) => setRepositoryUrl(e.target.value)}
                      onKeyDown={(e) =>
                        e.key === "Enter" && handleAnalyzeRepository()
                      }
                      className="flex-1"
                    />
                    <Button
                      onClick={() => handleAnalyzeRepository()}
                      disabled={!repositoryUrl.trim() || isAnalyzing}
                    >
                      {isAnalyzing ? "Analyzing..." : "Analyze"}
                    </Button>
                  </div>

                  {/* Token Input - Only show when needed */}
                  {showTokenInput && (
                    <Card
                      className="border"
                      style={{
                        backgroundColor: "hsl(var(--slate-100))",
                        borderColor: "hsl(var(--slate-300))",
                      }}
                    >
                      <CardHeader>
                        <CardTitle
                          className="text-lg"
                          style={{ color: "hsl(var(--slate-800))" }}
                        >
                          GitHub Authentication
                        </CardTitle>
                        <CardDescription
                          style={{ color: "hsl(var(--slate-600))" }}
                        >
                          This appears to be a private repository. Please
                          provide your GitHub Personal Access Token.
                        </CardDescription>
                      </CardHeader>
                      <CardContent className="space-y-4">
                        <div className="space-y-2">
                          <Input
                            type="password"
                            placeholder="ghp_xxxxxxxxxxxxxxxxxxxx"
                            value={githubToken}
                            onChange={(e) => setGithubToken(e.target.value)}
                            onKeyDown={(e) =>
                              e.key === "Enter" && handleTokenSubmit()
                            }
                          />
                          <div className="flex gap-2">
                            <Button onClick={handleTokenSubmit} size="sm">
                              Analyze with Token
                            </Button>
                            <Button
                              variant="outline"
                              onClick={() => {
                                setShowTokenInput(false);
                                setGithubToken("");
                                setError(null);
                              }}
                              size="sm"
                            >
                              Cancel
                            </Button>
                          </div>
                        </div>
                        <div
                          className="text-xs space-y-1"
                          style={{ color: "hsl(var(--slate-500))" }}
                        >
                          <p>
                            <strong>How to get a token:</strong>
                          </p>
                          <ol className="list-decimal list-inside space-y-1 ml-2">
                            <li>
                              Go to GitHub → Settings → Developer settings →
                              Personal access tokens
                            </li>
                            <li>
                              Generate a new token with "repo" permissions
                            </li>
                            <li>Copy the token and paste it above</li>
                          </ol>
                        </div>
                      </CardContent>
                    </Card>
                  )}

                  {/* Enhanced Progress Display */}
                  {isAnalyzing && (
                    <Card
                      className="border-0 shadow-lg"
                      style={{
                        backgroundColor: "white",
                        boxShadow:
                          "0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05)",
                      }}
                    >
                      <CardHeader
                        className="pb-4"
                        style={{ backgroundColor: "hsl(var(--slate-50))" }}
                      >
                        <CardTitle
                          className="text-lg flex items-center gap-2"
                          style={{ color: "hsl(var(--slate-800))" }}
                        >
                          <div className="w-3 h-3 rounded-full bg-primary animate-pulse"></div>
                          Analyzing Repository
                        </CardTitle>
                        <CardDescription
                          style={{ color: "hsl(var(--slate-600))" }}
                        >
                          AI-powered analysis in progress...
                        </CardDescription>
                      </CardHeader>
                      <CardContent className="space-y-6">
                        {/* Progress Bar Section */}
                        <div className="space-y-4">
                          <div className="flex items-center justify-between">
                            <span
                              className="text-sm font-medium"
                              style={{ color: "hsl(var(--slate-700))" }}
                            >
                              {analysisStage || "🚀 Preparing analysis..."}
                            </span>
                            <div className="flex items-center gap-2">
                              <span
                                className="text-lg font-bold"
                                style={{ color: "hsl(var(--slate-800))" }}
                              >
                                {analysisProgress}%
                              </span>
                            </div>
                          </div>

                          {/* Enhanced Live Progress Bar */}
                          <div className="relative">
                            {/* Live progress indicator */}
                            <div className="flex items-center justify-between mb-3">
                              <div className="flex items-center gap-2">
                                <div className="w-2 h-2 bg-primary rounded-full animate-pulse"></div>
                                <span
                                  className="text-xs font-medium"
                                  style={{ color: "hsl(var(--slate-600))" }}
                                >
                                  Live Progress
                                </span>
                              </div>
                              <div className="text-right">
                                <div
                                  className="text-xl font-bold tabular-nums tracking-tight"
                                  style={{ color: "hsl(var(--slate-800))" }}
                                >
                                  {analysisProgress}%
                                </div>
                                <div
                                  className="text-xs leading-none mt-0.5"
                                  style={{ color: "hsl(var(--slate-500))" }}
                                >
                                  {analysisProgress < 100
                                    ? "Processing..."
                                    : "Complete!"}
                                </div>
                              </div>
                            </div>

                            <div className="relative">
                              <Progress
                                value={analysisProgress}
                                className="w-full h-4 transition-all duration-700 ease-out"
                              />

                              {/* Animated glow effect */}
                              <div
                                className="absolute top-0 left-0 h-full bg-primary rounded-full transition-all duration-700 ease-out opacity-40 blur-sm"
                                style={{
                                  width: `${analysisProgress}%`,
                                  animation:
                                    analysisProgress < 100
                                      ? "pulse 2s ease-in-out infinite"
                                      : "none",
                                }}
                              />

                              {/* Moving shimmer effect for active progress */}
                              {analysisProgress < 100 &&
                                analysisProgress > 0 && (
                                  <div
                                    className="absolute top-0 left-0 h-full overflow-hidden rounded-full"
                                    style={{ width: `${analysisProgress}%` }}
                                  >
                                    <div
                                      className="absolute top-0 left-0 h-full w-8 bg-gradient-to-r from-transparent via-white to-transparent opacity-30"
                                      style={{
                                        animation:
                                          "shimmer 1.5s ease-in-out infinite",
                                        animationDelay: "0.5s",
                                      }}
                                    />
                                  </div>
                                )}

                              {/* Progress milestone markers */}
                              <div className="absolute top-0 left-0 w-full h-full flex items-center pointer-events-none">
                                {[25, 50, 75].map((milestone) => (
                                  <div
                                    key={milestone}
                                    className={`w-0.5 h-6 transition-all duration-500 ${
                                      analysisProgress >= milestone
                                        ? "bg-primary opacity-80 scale-110"
                                        : "bg-slate-300 opacity-40"
                                    }`}
                                    style={{
                                      marginLeft: `${milestone}%`,
                                      transform: `translateX(-50%) ${
                                        analysisProgress >= milestone
                                          ? "scaleY(1.2)"
                                          : ""
                                      }`,
                                    }}
                                  />
                                ))}
                              </div>
                            </div>
                          </div>

                          {/* Progress Stages Indicator */}
                          <div
                            className="flex justify-between text-xs"
                            style={{ color: "hsl(var(--slate-400))" }}
                          >
                            <span
                              className={
                                analysisProgress >= 10 ? "font-medium" : ""
                              }
                              style={{
                                color:
                                  analysisProgress >= 10
                                    ? "hsl(var(--slate-700))"
                                    : "hsl(var(--slate-400))",
                              }}
                            >
                              Clone
                            </span>
                            <span
                              className={
                                analysisProgress >= 30 ? "font-medium" : ""
                              }
                              style={{
                                color:
                                  analysisProgress >= 30
                                    ? "hsl(var(--slate-700))"
                                    : "hsl(var(--slate-400))",
                              }}
                            >
                              Scan
                            </span>
                            <span
                              className={
                                analysisProgress >= 50 ? "font-medium" : ""
                              }
                              style={{
                                color:
                                  analysisProgress >= 50
                                    ? "hsl(var(--slate-700))"
                                    : "hsl(var(--slate-400))",
                              }}
                            >
                              Analyze
                            </span>
                            <span
                              className={
                                analysisProgress >= 75 ? "font-medium" : ""
                              }
                              style={{
                                color:
                                  analysisProgress >= 75
                                    ? "hsl(var(--slate-700))"
                                    : "hsl(var(--slate-400))",
                              }}
                            >
                              Extract
                            </span>
                            <span
                              className={
                                analysisProgress >= 95 ? "font-medium" : ""
                              }
                              style={{
                                color:
                                  analysisProgress >= 95
                                    ? "hsl(var(--slate-700))"
                                    : "hsl(var(--slate-400))",
                              }}
                            >
                              Complete
                            </span>
                          </div>
                        </div>

                        {/* User Engagement - Tips & Time Estimate */}
                        <div className="space-y-3">
                          {/* Dynamic Tips */}
                          <div
                            className="p-3 rounded-lg border-l-4 border-l-primary"
                            style={{ backgroundColor: "hsl(var(--slate-50))" }}
                          >
                            <div className="flex items-center gap-2 mb-1">
                              <div className="w-1.5 h-1.5 bg-primary rounded-full animate-pulse"></div>
                              <div
                                className="text-xs font-medium"
                                style={{ color: "hsl(var(--slate-700))" }}
                              >
                                Did you know?
                              </div>
                            </div>
                            <div
                              className="text-sm"
                              style={{ color: "hsl(var(--slate-600))" }}
                            >
                              {analysisProgress < 25
                                ? "🔍 Our AI analyzes your code architecture, dependencies, and database relationships to provide comprehensive insights."
                                : analysisProgress < 50
                                ? "⚡ We're processing your files with advanced pattern recognition to identify microservices and components."
                                : analysisProgress < 75
                                ? "🏗️ The system is mapping your project structure and discovering service relationships for better understanding."
                                : analysisProgress < 95
                                ? "🗄️ Database schema extraction helps visualize your data relationships and table structures."
                                : "🎉 Almost done! Your comprehensive repository analysis is being finalized with all insights."}
                            </div>
                          </div>

                          {/* Time Estimate with Progress Context */}
                          <div
                            className="flex items-center justify-between p-3 rounded-lg"
                            style={{ backgroundColor: "hsl(var(--slate-100))" }}
                          >
                            <div>
                              <div
                                className="text-xs font-medium mb-1"
                                style={{ color: "hsl(var(--slate-600))" }}
                              >
                                Estimated Time Remaining
                              </div>
                              <div
                                className="text-sm font-medium"
                                style={{ color: "hsl(var(--slate-700))" }}
                              >
                                {analysisProgress < 10
                                  ? "2-30 minutes (analyzing repository size...)"
                                  : analysisProgress < 25
                                  ? `${Math.max(
                                      1,
                                      Math.ceil((100 - analysisProgress) * 0.3)
                                    )} - ${Math.max(
                                      5,
                                      Math.ceil((100 - analysisProgress) * 0.4)
                                    )} minutes`
                                  : analysisProgress < 50
                                  ? `${Math.max(
                                      1,
                                      Math.ceil((100 - analysisProgress) * 0.25)
                                    )} - ${Math.max(
                                      3,
                                      Math.ceil((100 - analysisProgress) * 0.35)
                                    )} minutes`
                                  : analysisProgress < 75
                                  ? `${Math.max(
                                      1,
                                      Math.ceil((100 - analysisProgress) * 0.2)
                                    )} - ${Math.max(
                                      2,
                                      Math.ceil((100 - analysisProgress) * 0.3)
                                    )} minutes`
                                  : analysisProgress < 95
                                  ? "Less than 2 minutes"
                                  : "Almost complete!"}
                              </div>
                            </div>
                            <div className="text-right">
                              <div
                                className="text-xs"
                                style={{ color: "hsl(var(--slate-500))" }}
                              >
                                Progress
                              </div>
                              <div
                                className="text-lg font-bold"
                                style={{ color: "hsl(var(--slate-800))" }}
                              >
                                {Math.round((analysisProgress / 100) * 100)}%
                              </div>
                            </div>
                          </div>

                          {/* Encouraging Message */}
                          <div className="text-center p-2">
                            <div
                              className="text-xs"
                              style={{ color: "hsl(var(--slate-500))" }}
                            >
                              {analysisProgress < 25
                                ? "🚀 Starting deep analysis of your repository..."
                                : analysisProgress < 50
                                ? "💪 Making great progress! Analyzing project structure..."
                                : analysisProgress < 75
                                ? "🎯 More than halfway there! Discovering relationships..."
                                : analysisProgress < 95
                                ? "🔥 Almost finished! Finalizing comprehensive insights..."
                                : "✨ Analysis complete! Preparing your results..."}
                            </div>
                          </div>
                        </div>
                      </CardContent>
                    </Card>
                  )}

                  {/* Error Display */}
                  {error && (
                    <Alert variant="destructive">
                      <AlertCircle className="h-4 w-4" />
                      <AlertTitle>Analysis Failed</AlertTitle>
                      <AlertDescription>{error}</AlertDescription>
                    </Alert>
                  )}
                </CardContent>
              </Card>

              {/* Features */}
              <div className="grid grid-cols-1 md:grid-cols-3 gap-8 max-w-4xl mx-auto">
                {features.map((feature, index) => (
                  <Card
                    key={index}
                    className="text-center border-0 shadow-md hover:shadow-lg transition-shadow"
                    style={{
                      backgroundColor: "white",
                    }}
                  >
                    <CardContent className="pt-8 pb-6 px-6">
                      <div
                        className="p-3 rounded-2xl mx-auto w-fit mb-4"
                        style={{ backgroundColor: "hsl(var(--slate-100))" }}
                      >
                        <feature.icon
                          className="h-8 w-8 mx-auto"
                          style={{ color: "hsl(var(--slate-700))" }}
                        />
                      </div>
                      <h3
                        className="text-lg font-semibold mb-3"
                        style={{ color: "hsl(var(--slate-800))" }}
                      >
                        {feature.title}
                      </h3>
                      <p
                        className="text-sm leading-relaxed"
                        style={{ color: "hsl(var(--slate-600))" }}
                      >
                        {feature.description}
                      </p>
                    </CardContent>
                  </Card>
                ))}
              </div>
            </div>
          </div>
        ) : (
          /* Dashboard Content */
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
            {/* Analysis Failed Error */}
            {analysisComplete && isAnalysisIncomplete() ? (
              <Card className="bg-destructive/10 border-destructive/20">
                <CardContent className="pt-6">
                  <div className="text-center py-12">
                    <AlertCircle className="h-16 w-16 text-destructive mx-auto mb-4" />
                    <CardTitle className="text-xl text-destructive mb-2">
                      Analysis Failed or Incomplete
                    </CardTitle>
                    <p className="text-destructive/80 mb-6 max-w-md mx-auto text-sm">
                      The analysis process couldn't extract sufficient
                      information from this repository. This might happen if the
                      repository is empty, private without access token, or
                      contains unsupported project types.
                    </p>
                    <div className="space-y-2 text-sm text-destructive/70 mb-6">
                      <p>
                        <strong>Possible solutions:</strong>
                      </p>
                      <ul className="list-disc list-inside space-y-1 max-w-md mx-auto">
                        <li>
                          Verify the repository URL is correct and accessible
                        </li>
                        <li>
                          For private repositories, ensure you provided a valid
                          GitHub token
                        </li>
                        <li>
                          Check that the repository contains source code files
                        </li>
                        <li>Try analyzing a different repository</li>
                      </ul>
                    </div>
                    <Button
                      variant="destructive"
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
                    >
                      Try Another Repository
                    </Button>
                  </div>
                </CardContent>
              </Card>
            ) : (
              <Tabs
                value={activeTab}
                onValueChange={setActiveTab}
                className="w-full"
              >
                <TabsContent value="overview" className="space-y-6">
                  <OverviewTab getAnalysisData={getAnalysisData} />
                </TabsContent>
                <TabsContent value="services" className="space-y-6">
                  <ServicesTab getAnalysisData={getAnalysisData} />
                </TabsContent>
                <TabsContent value="database" className="space-y-6">
                  <DatabaseTab getAnalysisData={getAnalysisData} />
                </TabsContent>
                <TabsContent value="secrets" className="space-y-6">
                  <SecretsTab getAnalysisData={getAnalysisData} />
                </TabsContent>
                <TabsContent value="questions" className="space-y-6">
                  <QuestionsTab getAnalysisData={getAnalysisData} />
                </TabsContent>
                <TabsContent value="relationships" className="space-y-6">
                  <RelationshipsTab getAnalysisData={getAnalysisData} />
                </TabsContent>
                {/* Files tab removed for better performance */}
                <TabsContent value="analysis" className="space-y-6">
                  <AnalysisTab getAnalysisData={getAnalysisData} />
                </TabsContent>
              </Tabs>
            )}
          </div>
        )}
      </main>
    </div>
  );
}

// Tab Components
function OverviewTab({ getAnalysisData }) {
  const data = getAnalysisData();

  return (
    <div className="space-y-8">
      {/* Project Summary */}
      <Card className="border-0 shadow-sm" style={{ backgroundColor: "white" }}>
        <CardHeader style={{ backgroundColor: "hsl(var(--slate-50))" }}>
          <CardTitle style={{ color: "hsl(var(--slate-800))" }}>
            Project Analysis Summary
          </CardTitle>
        </CardHeader>
        <CardContent className="pt-8">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
            <div className="text-center">
              <div
                className="p-4 rounded-2xl mb-3"
                style={{ backgroundColor: "hsl(var(--slate-100))" }}
              >
                <div
                  className="text-3xl font-bold"
                  style={{ color: "hsl(var(--slate-800))" }}
                >
                  {data?.projectType?.primary_type || "Unknown"}
                </div>
              </div>
              <div
                className="text-sm font-medium"
                style={{ color: "hsl(var(--slate-600))" }}
              >
                Project Type
              </div>
            </div>
            <div className="text-center">
              <div
                className="p-4 rounded-2xl mb-3"
                style={{ backgroundColor: "hsl(var(--slate-100))" }}
              >
                <div
                  className="text-3xl font-bold"
                  style={{ color: "hsl(var(--slate-700))" }}
                >
                  {Math.round((data?.projectType?.confidence || 0) * 10)}%
                </div>
              </div>
              <div
                className="text-sm font-medium"
                style={{ color: "hsl(var(--slate-600))" }}
              >
                Confidence
              </div>
            </div>
            <div className="text-center">
              <div
                className="p-4 rounded-2xl mb-3"
                style={{ backgroundColor: "hsl(var(--slate-100))" }}
              >
                <div
                  className="text-3xl font-bold"
                  style={{ color: "hsl(var(--slate-700))" }}
                >
                  {data?.services?.length || 0}
                </div>
              </div>
              <div
                className="text-sm font-medium"
                style={{ color: "hsl(var(--slate-600))" }}
              >
                Services
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Quick Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <Card
          className="border-0 shadow-sm"
          style={{ backgroundColor: "white" }}
        >
          <CardContent className="pt-6 pb-6">
            <div className="flex items-center gap-4">
              <div
                className="p-3 rounded-xl"
                style={{ backgroundColor: "hsl(var(--slate-100))" }}
              >
                <Server
                  className="h-6 w-6"
                  style={{ color: "hsl(var(--slate-700))" }}
                />
              </div>
              <div>
                <div
                  className="text-sm font-medium"
                  style={{ color: "hsl(var(--slate-500))" }}
                >
                  Architecture
                </div>
                <div
                  className="text-lg font-semibold"
                  style={{ color: "hsl(var(--slate-800))" }}
                >
                  {data?.projectSummary?.detailed_analysis?.architecture ||
                    "Unknown"}
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card
          className="border-0 shadow-sm"
          style={{ backgroundColor: "white" }}
        >
          <CardContent className="pt-6 pb-6">
            <div className="flex items-center gap-4">
              <div
                className="p-3 rounded-xl"
                style={{ backgroundColor: "hsl(var(--slate-100))" }}
              >
                <Database
                  className="h-6 w-6"
                  style={{ color: "hsl(var(--slate-700))" }}
                />
              </div>
              <div>
                <div
                  className="text-sm font-medium"
                  style={{ color: "hsl(var(--slate-500))" }}
                >
                  Database Tables
                </div>
                <div
                  className="text-lg font-semibold"
                  style={{ color: "hsl(var(--slate-800))" }}
                >
                  {(() => {
                    const databaseSchema = data?.databaseSchema;
                    return databaseSchema?.tables
                      ? Object.keys(databaseSchema.tables).length
                      : 0;
                  })()}
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card
          className="border-0 shadow-sm"
          style={{ backgroundColor: "white" }}
        >
          <CardContent className="pt-6 pb-6">
            <div className="flex items-center gap-4">
              <div
                className="p-3 rounded-xl"
                style={{ backgroundColor: "hsl(var(--slate-100))" }}
              >
                <GitBranch
                  className="h-6 w-6"
                  style={{ color: "hsl(var(--slate-700))" }}
                />
              </div>
              <div>
                <div
                  className="text-sm font-medium"
                  style={{ color: "hsl(var(--slate-500))" }}
                >
                  Dependencies
                </div>
                <div
                  className="text-lg font-semibold"
                  style={{ color: "hsl(var(--slate-800))" }}
                >
                  {data?.relationships?.length || 0}
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Tech Stack */}
      <Card>
        <CardHeader>
          <CardTitle>Technology Stack</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-wrap gap-2">
            {(() => {
              const languages = data?.projectSummary?.languages || {};
              const languageEntries = Object.entries(languages);

              if (languageEntries.length === 0) {
                return (
                  <div className="text-center py-4 text-muted-foreground w-full">
                    <Code2 className="h-8 w-8 mx-auto mb-2 opacity-50" />
                    <p>No programming languages detected</p>
                  </div>
                );
              }

              return languageEntries.map(([lang, lines]) => (
                <Badge key={lang} variant="default">
                  {lang} ({lines} lines)
                </Badge>
              ));
            })()}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function ServicesTab({ getAnalysisData }) {
  const data = getAnalysisData();
  const services = data?.services || [];

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Discovered Services</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {services.length === 0 ? (
              <div className="text-center py-8 text-muted-foreground">
                <Server className="h-12 w-12 mx-auto mb-4 opacity-50" />
                <p className="mb-4">
                  No microservices detected in this repository
                </p>
                <div className="text-xs text-muted-foreground/80 bg-muted/30 rounded-lg p-4 max-w-md mx-auto">
                  <div className="font-medium mb-2 text-muted-foreground">
                    💡 Discovery Requirements:
                  </div>
                  <div className="text-left space-y-1">
                    <div>
                      • Services need a startup command in{" "}
                      <code className="bg-background px-1 rounded">
                        Makefile
                      </code>
                    </div>
                    <div>
                      • Or explicit service definitions in configuration files
                    </div>
                  </div>
                </div>
              </div>
            ) : (
              services.map((service) => (
                <div
                  key={service.name}
                  className="flex items-center justify-between p-4 border rounded-lg"
                >
                  <div className="flex items-center gap-3">
                    <Server className="h-6 w-6 text-primary" />
                    <div>
                      <div className="font-medium">{service.name}</div>
                      <div className="text-sm text-muted-foreground">
                        {service.port && `Port: ${service.port}`}
                      </div>
                      {service.description && (
                        <div className="text-xs text-muted-foreground mt-1">
                          {service.description}
                        </div>
                      )}
                    </div>
                  </div>
                  <Badge
                    variant={
                      service.api_type === "grpc" ? "secondary" : "default"
                    }
                  >
                    {service.api_type}
                  </Badge>
                </div>
              ))
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

// Database Tab Component
function DatabaseTab({ getAnalysisData }) {
  const data = getAnalysisData();
  const databaseSchema = data?.databaseSchema;
  const [expandedTables, setExpandedTables] = useState(new Set());
  const [activeView, setActiveView] = useState("tables");

  const toggleTableExpansion = (tableName) => {
    const newExpanded = new Set(expandedTables);
    if (newExpanded.has(tableName)) {
      newExpanded.delete(tableName);
    } else {
      newExpanded.add(tableName);
    }
    setExpandedTables(newExpanded);
  };

  const generateMermaidERD = () => {
    if (!databaseSchema || !databaseSchema.tables) return "";

    let mermaid = "erDiagram\n";

    // Add tables and their columns with correct Mermaid.js syntax
    Object.entries(databaseSchema.tables).forEach(([tableName, tableInfo]) => {
      mermaid += `    ${tableName} {\n`;

      // Add columns with proper data type format
      Object.entries(tableInfo.columns || {}).forEach(([colName, colInfo]) => {
        const constraints = colInfo.constraints || [];
        const isPK =
          constraints.includes("PK") ||
          (tableInfo.primary_keys || []).includes(colName);
        const isFK = constraints.includes("FK") || colInfo.references;

        // Mermaid ERD format: dataType columnName constraint
        let constraintText = "";
        if (isPK) constraintText = "PK";
        else if (isFK) constraintText = "FK";

        // Clean column type for Mermaid compatibility
        const cleanType = (colInfo.type || "varchar").replace(/[^\w]/g, "");
        mermaid += `        ${cleanType} ${colName}`;
        if (constraintText) mermaid += ` ${constraintText}`;
        mermaid += "\n";
      });

      mermaid += "    }\n\n";
    });

    // Add relationships with correct Mermaid.js ERD syntax
    Object.entries(databaseSchema.tables).forEach(([tableName, tableInfo]) => {
      Object.entries(tableInfo.columns || {}).forEach(([colName, colInfo]) => {
        if (colInfo.references) {
          // Mermaid ERD relationship format: PARENT_TABLE ||--o{ CHILD_TABLE : "relationship_name"
          mermaid += `    ${colInfo.references.table} ||--o{ ${tableName} : has\n`;
        }
      });
    });

    return mermaid;
  };

  if (
    !databaseSchema ||
    !databaseSchema.tables ||
    Object.keys(databaseSchema.tables).length === 0
  ) {
    return (
      <div className="space-y-6">
        <Card
          className="border-0 shadow-sm"
          style={{ backgroundColor: "white" }}
        >
          <CardContent className="pt-6">
            <div className="text-center py-12">
              <Database
                className="h-16 w-16 mx-auto mb-4"
                style={{ color: "hsl(var(--slate-400))" }}
              />
              <CardTitle
                className="text-xl mb-2"
                style={{ color: "hsl(var(--slate-700))" }}
              >
                No Database Schema Detected
              </CardTitle>
              <p
                className="max-w-md mx-auto text-sm mb-4"
                style={{ color: "hsl(var(--slate-500))" }}
              >
                No database migration files or schema definitions found in this
                repository. The analyzer looks for SQL files in migration
                directories.
              </p>
              <div
                className="text-xs bg-slate-50 border rounded-lg p-4 max-w-md mx-auto"
                style={{ color: "hsl(var(--slate-600))" }}
              >
                <div className="font-medium mb-2">
                  💡 Detection Requirements:
                </div>
                <div className="text-left space-y-1">
                  <div>
                    • Database tables need to be defined in the{" "}
                    <code className="bg-white px-1 rounded border">
                      migrations/
                    </code>{" "}
                    folder
                  </div>
                  <div>
                    • Supported formats: SQL migration files, schema definitions
                  </div>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  const tableCount = Object.keys(databaseSchema.tables).length;
  const totalColumns = Object.values(databaseSchema.tables).reduce(
    (sum, table) => sum + Object.keys(table.columns || {}).length,
    0
  );
  const totalRelationships = Object.values(databaseSchema.tables).reduce(
    (sum, table) =>
      sum +
      Object.values(table.columns || {}).filter((col) => col.references).length,
    0
  );

  return (
    <div className="space-y-8">
      {/* Database Overview Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <Card
          className="border-0 shadow-sm"
          style={{ backgroundColor: "white" }}
        >
          <CardContent className="pt-6 pb-6">
            <div className="flex items-center gap-4">
              <div
                className="p-3 rounded-xl"
                style={{ backgroundColor: "hsl(var(--slate-100))" }}
              >
                <Table
                  className="h-6 w-6"
                  style={{ color: "hsl(var(--slate-700))" }}
                />
              </div>
              <div>
                <div
                  className="text-sm font-medium"
                  style={{ color: "hsl(var(--slate-500))" }}
                >
                  Tables
                </div>
                <div
                  className="text-2xl font-bold"
                  style={{ color: "hsl(var(--slate-800))" }}
                >
                  {tableCount}
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card
          className="border-0 shadow-sm"
          style={{ backgroundColor: "white" }}
        >
          <CardContent className="pt-6 pb-6">
            <div className="flex items-center gap-4">
              <div
                className="p-3 rounded-xl"
                style={{ backgroundColor: "hsl(var(--slate-100))" }}
              >
                <Database
                  className="h-6 w-6"
                  style={{ color: "hsl(var(--slate-700))" }}
                />
              </div>
              <div>
                <div
                  className="text-sm font-medium"
                  style={{ color: "hsl(var(--slate-500))" }}
                >
                  Columns
                </div>
                <div
                  className="text-2xl font-bold"
                  style={{ color: "hsl(var(--slate-800))" }}
                >
                  {totalColumns}
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card
          className="border-0 shadow-sm"
          style={{ backgroundColor: "white" }}
        >
          <CardContent className="pt-6 pb-6">
            <div className="flex items-center gap-4">
              <div
                className="p-3 rounded-xl"
                style={{ backgroundColor: "hsl(var(--slate-100))" }}
              >
                <Link
                  className="h-6 w-6"
                  style={{ color: "hsl(var(--slate-700))" }}
                />
              </div>
              <div>
                <div
                  className="text-sm font-medium"
                  style={{ color: "hsl(var(--slate-500))" }}
                >
                  Relationships
                </div>
                <div
                  className="text-2xl font-bold"
                  style={{ color: "hsl(var(--slate-800))" }}
                >
                  {totalRelationships}
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* View Selector */}
      <Card className="border-0 shadow-sm" style={{ backgroundColor: "white" }}>
        <CardHeader style={{ backgroundColor: "hsl(var(--slate-50))" }}>
          <div className="flex items-center justify-between">
            <CardTitle style={{ color: "hsl(var(--slate-800))" }}>
              Database Schema
            </CardTitle>
            <div className="flex gap-2">
              <Button
                variant={activeView === "tables" ? "default" : "outline"}
                size="sm"
                onClick={() => setActiveView("tables")}
              >
                <Table className="h-4 w-4 mr-2" />
                Tables
              </Button>
              <Button
                variant={activeView === "relationships" ? "default" : "outline"}
                size="sm"
                onClick={() => setActiveView("relationships")}
              >
                <Link className="h-4 w-4 mr-2" />
                Relationships
              </Button>
              <Button
                variant={activeView === "migration" ? "default" : "outline"}
                size="sm"
                onClick={() => setActiveView("migration")}
              >
                <FileText className="h-4 w-4 mr-2" />
                Final Migration
              </Button>
              <Button
                variant={
                  activeView === "llm_relationships" ? "default" : "outline"
                }
                size="sm"
                onClick={() => setActiveView("llm_relationships")}
              >
                <Brain className="h-4 w-4 mr-2" />
                LLM Relationships
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent className="pt-6">
          {activeView === "tables" ? (
            <div className="space-y-4">
              {Object.entries(databaseSchema.tables).map(
                ([tableName, tableInfo]) => {
                  const isExpanded = expandedTables.has(tableName);
                  const columns = Object.entries(tableInfo.columns || {});
                  const primaryKeys = tableInfo.primary_keys || [];

                  return (
                    <Card
                      key={tableName}
                      className="border"
                      style={{ borderColor: "hsl(var(--slate-200))" }}
                    >
                      <CardHeader
                        className="pb-3 cursor-pointer hover:bg-slate-50 transition-colors"
                        onClick={() => toggleTableExpansion(tableName)}
                      >
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-3">
                            <div
                              className="p-2 rounded-lg"
                              style={{
                                backgroundColor: "hsl(var(--slate-100))",
                              }}
                            >
                              <Database
                                className="h-5 w-5"
                                style={{ color: "hsl(var(--slate-700))" }}
                              />
                            </div>
                            <div>
                              <CardTitle
                                className="text-lg"
                                style={{ color: "hsl(var(--slate-800))" }}
                              >
                                {tableName}
                              </CardTitle>
                              <div
                                className="text-sm"
                                style={{ color: "hsl(var(--slate-500))" }}
                              >
                                {columns.length} columns
                                {primaryKeys.length > 0 &&
                                  ` • ${primaryKeys.length} primary key${
                                    primaryKeys.length > 1 ? "s" : ""
                                  }`}
                              </div>
                            </div>
                          </div>
                          {isExpanded ? (
                            <ChevronDown
                              className="h-5 w-5"
                              style={{ color: "hsl(var(--slate-500))" }}
                            />
                          ) : (
                            <ChevronRight
                              className="h-5 w-5"
                              style={{ color: "hsl(var(--slate-500))" }}
                            />
                          )}
                        </div>
                      </CardHeader>

                      {isExpanded && (
                        <CardContent className="pt-0">
                          <Separator />
                          <div className="mt-4">
                            <div className="grid gap-3">
                              {columns.map(([colName, colInfo]) => {
                                const isPrimaryKey =
                                  primaryKeys.includes(colName);
                                const constraints = colInfo.constraints || [];
                                const isFK =
                                  constraints.includes("FK") ||
                                  colInfo.references;

                                return (
                                  <div
                                    key={colName}
                                    className="flex items-center justify-between p-3 rounded-lg"
                                    style={{
                                      backgroundColor: "hsl(var(--slate-50))",
                                    }}
                                  >
                                    <div className="flex items-center gap-3">
                                      {isPrimaryKey ? (
                                        <Key
                                          className="h-4 w-4"
                                          style={{
                                            color: "hsl(var(--slate-600))",
                                          }}
                                        />
                                      ) : isFK ? (
                                        <Link
                                          className="h-4 w-4"
                                          style={{
                                            color: "hsl(var(--slate-600))",
                                          }}
                                        />
                                      ) : (
                                        <div className="w-4 h-4" />
                                      )}
                                      <div>
                                        <div
                                          className="font-medium"
                                          style={{
                                            color: "hsl(var(--slate-800))",
                                          }}
                                        >
                                          {colName}
                                        </div>
                                        <div
                                          className="text-sm"
                                          style={{
                                            color: "hsl(var(--slate-600))",
                                          }}
                                        >
                                          {colInfo.type}
                                          {colInfo.references && (
                                            <span
                                              className="ml-2"
                                              style={{
                                                color: "hsl(var(--slate-500))",
                                              }}
                                            >
                                              → {colInfo.references.table}.
                                              {colInfo.references.column}
                                            </span>
                                          )}
                                        </div>
                                      </div>
                                    </div>
                                    <div className="flex gap-1">
                                      {isPrimaryKey && (
                                        <Badge
                                          variant="default"
                                          className="text-xs"
                                        >
                                          PK
                                        </Badge>
                                      )}
                                      {isFK && (
                                        <Badge
                                          variant="secondary"
                                          className="text-xs"
                                        >
                                          FK
                                        </Badge>
                                      )}
                                      {constraints.includes("unique") && (
                                        <Badge
                                          variant="outline"
                                          className="text-xs"
                                        >
                                          Unique
                                        </Badge>
                                      )}
                                      {constraints.includes("not null") && (
                                        <Badge
                                          variant="outline"
                                          className="text-xs"
                                        >
                                          Not Null
                                        </Badge>
                                      )}
                                    </div>
                                  </div>
                                );
                              })}
                            </div>
                          </div>
                        </CardContent>
                      )}
                    </Card>
                  );
                }
              )}
            </div>
          ) : activeView === "relationships" ? (
            <div className="space-y-4">
              <div className="text-center py-4">
                <div
                  className="text-sm"
                  style={{ color: "hsl(var(--slate-600))" }}
                >
                  Database Relationship Diagram
                </div>
              </div>
              <ZoomableMermaid
                mermaidCode={generateMermaidERD()}
                title="Database Entity Relationship Diagram"
                className="min-h-96"
                containerClassName="bg-white"
                initialZoom={0.8}
                maxZoom={3.0}
                minZoom={0.2}
              />
              {totalRelationships === 0 && (
                <div
                  className="text-center py-8"
                  style={{ color: "hsl(var(--slate-500))" }}
                >
                  <Link className="h-12 w-12 mx-auto mb-4 opacity-50" />
                  <p>
                    No foreign key relationships detected in the database schema
                  </p>
                </div>
              )}
            </div>
          ) : activeView === "migration" ? (
            <div className="space-y-4">
              <div className="text-center py-4">
                <div
                  className="text-sm"
                  style={{ color: "hsl(var(--slate-600))" }}
                >
                  Final Migration SQL - Complete Database Schema
                </div>
                <div
                  className="text-xs mt-2"
                  style={{ color: "hsl(var(--slate-500))" }}
                >
                  Run this single file instead of all individual migrations
                </div>
              </div>
              <div
                className="border rounded-lg"
                style={{
                  backgroundColor: "hsl(var(--slate-50))",
                  borderColor: "hsl(var(--slate-200))",
                }}
              >
                {databaseSchema.final_migration_sql ? (
                  <div className="p-4">
                    <div className="flex items-center justify-between mb-4">
                      <div
                        className="text-sm font-medium"
                        style={{ color: "hsl(var(--slate-700))" }}
                      >
                        Generated Migration (
                        {databaseSchema.final_migration_sql.length} characters)
                      </div>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => {
                          navigator.clipboard.writeText(
                            databaseSchema.final_migration_sql
                          );
                        }}
                      >
                        <Download className="h-4 w-4 mr-2" />
                        Copy SQL
                      </Button>
                    </div>
                    <pre
                      className="text-sm overflow-x-auto p-4 rounded-lg font-mono whitespace-pre-wrap"
                      style={{
                        backgroundColor: "hsl(var(--slate-900))",
                        color: "hsl(var(--slate-100))",
                        lineHeight: "1.5",
                      }}
                    >
                      {databaseSchema.final_migration_sql}
                    </pre>
                  </div>
                ) : (
                  <div
                    className="text-center py-8"
                    style={{ color: "hsl(var(--slate-500))" }}
                  >
                    <FileText className="h-12 w-12 mx-auto mb-4 opacity-50" />
                    <p>Final migration SQL not available</p>
                    <p className="text-sm mt-2">
                      This might happen if no migrations were processed
                      successfully
                    </p>
                  </div>
                )}
              </div>
            </div>
          ) : (
            <div className="space-y-4">
              <div className="text-center py-4">
                <div
                  className="text-sm"
                  style={{ color: "hsl(var(--slate-600))" }}
                >
                  LLM-Enhanced Relationship Analysis
                </div>
                <div
                  className="text-xs mt-2"
                  style={{ color: "hsl(var(--slate-500))" }}
                >
                  AI-detected relationships including implicit connections
                  between tables
                </div>
              </div>
              <div
                className="border rounded-lg"
                style={{
                  backgroundColor: "hsl(var(--slate-50))",
                  borderColor: "hsl(var(--slate-200))",
                }}
              >
                {databaseSchema.llm_relationships ? (
                  <div className="p-4">
                    <div className="flex items-center justify-between mb-4">
                      <div
                        className="text-sm font-medium"
                        style={{ color: "hsl(var(--slate-700))" }}
                      >
                        <Brain className="h-4 w-4 inline mr-2" />
                        AI-Generated Relationships (
                        {databaseSchema.llm_relationships.length} characters)
                      </div>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => {
                          navigator.clipboard.writeText(
                            databaseSchema.llm_relationships
                          );
                        }}
                      >
                        <Download className="h-4 w-4 mr-2" />
                        Copy Mermaid
                      </Button>
                    </div>
                    <ZoomableMermaid
                      mermaidCode={databaseSchema.llm_relationships}
                      title="AI-Generated Database Relationships"
                      className="min-h-96"
                      containerClassName=""
                      initialZoom={0.7}
                      maxZoom={4.0}
                      minZoom={0.1}
                    />
                    <details className="mt-4">
                      <summary
                        className="cursor-pointer text-sm font-medium mb-2"
                        style={{ color: "hsl(var(--slate-700))" }}
                      >
                        View Raw Mermaid Code
                      </summary>
                      <pre
                        className="text-sm overflow-x-auto p-4 rounded-lg font-mono whitespace-pre-wrap"
                        style={{
                          backgroundColor: "hsl(var(--slate-900))",
                          color: "hsl(var(--slate-100))",
                          lineHeight: "1.5",
                        }}
                      >
                        {databaseSchema.llm_relationships}
                      </pre>
                    </details>
                  </div>
                ) : (
                  <div
                    className="text-center py-8"
                    style={{ color: "hsl(var(--slate-500))" }}
                  >
                    <Brain className="h-12 w-12 mx-auto mb-4 opacity-50" />
                    <p>LLM relationship analysis not available</p>
                    <p className="text-sm mt-2">
                      This could be due to missing OpenAI configuration or
                      analysis failure
                    </p>
                  </div>
                )}
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

// Questions Tab Component
function QuestionsTab({ getAnalysisData }) {
  const data = getAnalysisData();
  const questions = data?.helpfulQuestions || [];

  if (!data) {
    return (
      <div className="text-center py-12">
        <AlertCircle className="h-12 w-12 mx-auto mb-4 opacity-50" />
        <p style={{ color: "hsl(var(--slate-500))" }}>No data available</p>
      </div>
    );
  }

  if (!questions || questions.length === 0) {
    return (
      <div className="space-y-6">
        <Card
          className="border-0 shadow-sm"
          style={{ backgroundColor: "white" }}
        >
          <CardContent className="pt-6">
            <div className="text-center py-12">
              <Eye
                className="h-16 w-16 mx-auto mb-4"
                style={{ color: "hsl(var(--slate-400))" }}
              />
              <CardTitle
                className="text-xl mb-2"
                style={{ color: "hsl(var(--slate-700))" }}
              >
                No Helpful Questions Available
              </CardTitle>
              <p
                className="max-w-md mx-auto text-sm"
                style={{ color: "hsl(var(--slate-500))" }}
              >
                Questions specific to this project could not be generated. This
                might happen if the analysis is incomplete or the project
                structure is too minimal.
              </p>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <Card className="border-0 shadow-sm" style={{ backgroundColor: "white" }}>
        <CardHeader style={{ backgroundColor: "hsl(var(--slate-50))" }}>
          <CardTitle
            className="flex items-center gap-3"
            style={{ color: "hsl(var(--slate-800))" }}
          >
            <div
              className="p-2 rounded-lg"
              style={{ backgroundColor: "hsl(var(--slate-100))" }}
            >
              <Eye
                className="h-6 w-6"
                style={{ color: "hsl(var(--slate-700))" }}
              />
            </div>
            Helpful Questions & Answers
          </CardTitle>
          <CardDescription>
            Project-specific questions to help you understand and develop this
            application faster
          </CardDescription>
        </CardHeader>
        <CardContent className="pt-6">
          <div className="text-center mb-6">
            <div
              className="text-2xl font-bold"
              style={{ color: "hsl(var(--slate-800))" }}
            >
              {questions.length} Questions
            </div>
            <div className="text-sm" style={{ color: "hsl(var(--slate-500))" }}>
              Generated based on your project analysis
            </div>
          </div>
        </CardContent>
      </Card>

      {/* FAQ Cards */}
      <div className="space-y-4">
        {questions.map((qa, index) => (
          <details
            key={index}
            className="group border rounded-lg"
            style={{ borderColor: "hsl(var(--slate-200))" }}
          >
            <summary className="flex items-center justify-between p-6 cursor-pointer hover:bg-slate-50 transition-colors">
              <div className="flex items-start gap-4 flex-1">
                <div
                  className="p-2 rounded-lg mt-1"
                  style={{ backgroundColor: "hsl(var(--slate-100))" }}
                >
                  <span
                    className="text-sm font-bold"
                    style={{ color: "hsl(var(--slate-700))" }}
                  >
                    Q{index + 1}
                  </span>
                </div>
                <div className="flex-1">
                  <h3
                    className="text-lg font-semibold mb-2"
                    style={{ color: "hsl(var(--slate-800))" }}
                  >
                    {qa.question}
                  </h3>
                  <div
                    className="text-sm"
                    style={{ color: "hsl(var(--slate-500))" }}
                  >
                    Click to see the answer
                  </div>
                </div>
              </div>
              <ChevronDown
                className="h-5 w-5 transition-transform group-open:rotate-180"
                style={{ color: "hsl(var(--slate-500))" }}
              />
            </summary>
            <div className="px-6 pb-6">
              <Separator className="mb-4" />
              <div className="pl-12">
                <div
                  className="p-4 rounded-lg"
                  style={{ backgroundColor: "hsl(var(--slate-50))" }}
                >
                  <div
                    className="text-sm font-medium mb-2"
                    style={{ color: "hsl(var(--slate-700))" }}
                  >
                    Answer:
                  </div>
                  <div
                    className="text-sm leading-relaxed whitespace-pre-wrap"
                    style={{ color: "hsl(var(--slate-600))" }}
                  >
                    {qa.answer}
                  </div>
                </div>
              </div>
            </div>
          </details>
        ))}
      </div>

      {/* Footer */}
      <Card
        className="border-0 shadow-sm"
        style={{ backgroundColor: "hsl(var(--slate-50))" }}
      >
        <CardContent className="pt-6 pb-6">
          <div className="text-center">
            <div className="text-sm" style={{ color: "hsl(var(--slate-600))" }}>
              💡 These questions are AI-generated based on your project's
              specific structure, dependencies, and patterns to help accelerate
              your development process.
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

// Relationships Tab Component
function RelationshipsTab({ getAnalysisData }) {
  const data = getAnalysisData();
  const relationships = data?.relationships || [];
  const services = data?.services || [];

  // Generate Mermaid diagram for service relationships
  const generateServiceMermaid = () => {
    if (!relationships.length && !services.length) return "";

    let mermaid = "graph TD\n";

    // Create nodes for all services
    const serviceNodes = new Set();

    // Add services from the services array
    services.forEach((service) => {
      const nodeId = service.name.replace(/[^a-zA-Z0-9]/g, "_");
      const serviceType = service.api_type || "http";
      serviceNodes.add(nodeId);
      mermaid += `    ${nodeId}[${
        service.name
      } - ${serviceType.toUpperCase()}]\n`;
    });

    // Add services from relationships if not already added
    relationships.forEach((rel) => {
      const fromId = rel.from.replace(/[^a-zA-Z0-9]/g, "_");
      const toId = rel.to.replace(/[^a-zA-Z0-9]/g, "_");

      if (!serviceNodes.has(fromId)) {
        serviceNodes.add(fromId);
        mermaid += `    ${fromId}[${rel.from}]\n`;
      }
      if (!serviceNodes.has(toId)) {
        serviceNodes.add(toId);
        mermaid += `    ${toId}[${rel.to}]\n`;
      }
    });

    mermaid += "\n";

    // Add relationships
    relationships.forEach((rel) => {
      const fromId = rel.from.replace(/[^a-zA-Z0-9]/g, "_");
      const toId = rel.to.replace(/[^a-zA-Z0-9]/g, "_");
      const linkType = rel.type || "depends on";
      mermaid += `    ${fromId} --> ${toId}\n`;
    });

    // Add styling
    mermaid += "\n";
    mermaid +=
      "    classDef httpService fill:#e1f5fe,stroke:#01579b,stroke-width:2px\n";
    mermaid +=
      "    classDef grpcService fill:#f3e5f5,stroke:#4a148c,stroke-width:2px\n";
    mermaid +=
      "    classDef graphqlService fill:#e8f5e8,stroke:#1b5e20,stroke-width:2px\n";

    return mermaid;
  };

  if (!data) {
    return (
      <div className="text-center py-12">
        <AlertCircle className="h-12 w-12 mx-auto mb-4 opacity-50" />
        <p style={{ color: "hsl(var(--slate-500))" }}>No data available</p>
      </div>
    );
  }

  const hasDiagramData = relationships.length > 0 || services.length > 0;

  return (
    <div className="space-y-6">
      {/* Service Relationship Diagram */}
      {hasDiagramData && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Link className="h-5 w-5" />
              Service Architecture Diagram
            </CardTitle>
            <CardDescription>
              Visual representation of service dependencies and relationships
            </CardDescription>
          </CardHeader>
          <CardContent>
            <ZoomableMermaid
              mermaidCode={generateServiceMermaid()}
              title="Service Relationship Diagram"
              className="min-h-80"
              containerClassName="bg-white"
              initialZoom={0.9}
              maxZoom={3.0}
              minZoom={0.3}
            />
          </CardContent>
        </Card>
      )}

      {/* Detailed Relationships List */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Link className="h-5 w-5" />
            Service Relationships
          </CardTitle>
          <CardDescription>
            Detailed connections and dependencies between services
          </CardDescription>
        </CardHeader>
        <CardContent>
          {relationships.length > 0 ? (
            <div className="space-y-4">
              {relationships.map((rel, index) => (
                <div
                  key={index}
                  className="flex items-center justify-between p-4 rounded-lg border"
                >
                  <div>
                    <div className="font-medium">{rel.from}</div>
                    <div className="text-sm text-muted-foreground">
                      {rel.type || "dependency"}
                    </div>
                  </div>
                  <div className="flex items-center text-sm text-muted-foreground">
                    <div className="w-8 border-t border-gray-300"></div>
                    <div className="mx-2">→</div>
                  </div>
                  <div className="text-right">
                    <div className="font-medium">{rel.to}</div>
                    <div className="text-sm text-muted-foreground">
                      {rel.description || "Connected service"}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          ) : services.length > 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              <Link className="h-12 w-12 mx-auto mb-4 opacity-50" />
              <p>Services detected but no explicit relationships found</p>
              <p className="text-sm mt-2">
                See the diagram above for service architecture
              </p>
            </div>
          ) : (
            <div className="text-center py-8 text-muted-foreground">
              <Link className="h-12 w-12 mx-auto mb-4 opacity-50" />
              <p>No service relationships detected</p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

// Analysis Tab Component
function AnalysisTab({ getAnalysisData }) {
  const data = getAnalysisData();
  const projectSummary = data?.projectSummary || {};
  const folderSummaries = data?.folderSummaries || {};
  const folderEntries = Object.entries(folderSummaries);

  if (!data) {
    return (
      <div className="text-center py-12">
        <AlertCircle className="h-12 w-12 mx-auto mb-4 opacity-50" />
        <p style={{ color: "hsl(var(--slate-500))" }}>No data available</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <BarChart3 className="h-5 w-5" />
            Project Analysis
          </CardTitle>
          <CardDescription>Comprehensive analysis and insights</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-6">
            {projectSummary.purpose && (
              <div>
                <h3 className="font-medium mb-2">Purpose</h3>
                <p className="text-sm text-muted-foreground">
                  {projectSummary.purpose}
                </p>
              </div>
            )}

            {projectSummary.architecture && (
              <div>
                <h3 className="font-medium mb-2">Architecture</h3>
                <Badge variant="secondary">{projectSummary.architecture}</Badge>
              </div>
            )}

            {projectSummary.languages &&
              Object.keys(projectSummary.languages).length > 0 && (
                <div>
                  <h3 className="font-medium mb-2">Languages</h3>
                  <div className="flex flex-wrap gap-2">
                    {Object.entries(projectSummary.languages).map(
                      ([lang, count]) => (
                        <Badge key={lang} variant="outline">
                          {lang} ({count})
                        </Badge>
                      )
                    )}
                  </div>
                </div>
              )}
          </div>
        </CardContent>
      </Card>

      {folderEntries.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Folder Analysis</CardTitle>
            <CardDescription>
              Analysis of {folderEntries.length} folders
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {folderEntries.map(([folderPath, folderInfo]) => (
                <Card key={folderPath}>
                  <CardHeader className="pb-3">
                    <CardTitle className="text-sm font-medium">
                      {folderPath}
                    </CardTitle>
                    <CardDescription className="text-sm">
                      {folderInfo.purpose}
                    </CardDescription>
                  </CardHeader>
                  <CardContent className="pt-0">
                    <div className="space-y-2">
                      {folderInfo.languages && (
                        <div className="flex flex-wrap gap-1">
                          {Object.entries(folderInfo.languages).map(
                            ([lang, count]) => (
                              <Badge
                                key={lang}
                                variant="outline"
                                className="text-xs"
                              >
                                {lang} ({count})
                              </Badge>
                            )
                          )}
                        </div>
                      )}
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

// Secrets Tab Component
function SecretsTab({ getAnalysisData }) {
  const data = getAnalysisData();
  const projectSecrets = data?.projectSecrets;

  if (!projectSecrets) {
    return (
      <div className="space-y-6">
        <Card
          className="border-0 shadow-sm"
          style={{ backgroundColor: "white" }}
        >
          <CardContent className="pt-6">
            <div className="text-center py-12">
              <Shield
                className="h-16 w-16 mx-auto mb-4"
                style={{ color: "hsl(var(--slate-400))" }}
              />
              <CardTitle
                className="text-xl mb-2"
                style={{ color: "hsl(var(--slate-700))" }}
              >
                No Secret Variables Detected
              </CardTitle>
              <p
                className="max-w-md mx-auto text-sm mb-4"
                style={{ color: "hsl(var(--slate-500))" }}
              >
                No environment variables or configuration secrets found. This
                project may not require additional configuration.
              </p>
              <div
                className="text-xs bg-slate-50 border rounded-lg p-4 max-w-md mx-auto"
                style={{ color: "hsl(var(--slate-600))" }}
              >
                <div className="font-medium mb-2">
                  💡 Detection Requirements:
                </div>
                <div className="text-left space-y-1">
                  <div>
                    • Secrets must be defined in{" "}
                    <code className="bg-white px-1 rounded border">.env</code>{" "}
                    files
                  </div>
                  <div>• Or in YAML configuration files</div>
                  <div>
                    • Variables like API keys, database URLs, tokens, etc.
                  </div>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  const totalRequired = projectSecrets.required_count || 0;
  const totalVariables = projectSecrets.total_variables || 0;

  return (
    <div className="space-y-8">
      {/* Secrets Overview Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <Card
          className="border-0 shadow-sm"
          style={{ backgroundColor: "white" }}
        >
          <CardContent className="pt-6 pb-6">
            <div className="flex items-center gap-4">
              <div
                className="p-3 rounded-xl"
                style={{ backgroundColor: "hsl(var(--slate-100))" }}
              >
                <Shield
                  className="h-6 w-6"
                  style={{ color: "hsl(var(--slate-700))" }}
                />
              </div>
              <div>
                <div
                  className="text-sm font-medium"
                  style={{ color: "hsl(var(--slate-500))" }}
                >
                  Total Variables
                </div>
                <div
                  className="text-2xl font-bold"
                  style={{ color: "hsl(var(--slate-800))" }}
                >
                  {totalVariables}
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card
          className="border-0 shadow-sm"
          style={{ backgroundColor: "white" }}
        >
          <CardContent className="pt-6 pb-6">
            <div className="flex items-center gap-4">
              <div
                className="p-3 rounded-xl"
                style={{ backgroundColor: "hsl(var(--red-100))" }}
              >
                <AlertCircle
                  className="h-6 w-6"
                  style={{ color: "hsl(var(--red-600))" }}
                />
              </div>
              <div>
                <div
                  className="text-sm font-medium"
                  style={{ color: "hsl(var(--slate-500))" }}
                >
                  Required
                </div>
                <div
                  className="text-2xl font-bold"
                  style={{ color: "hsl(var(--red-600))" }}
                >
                  {totalRequired}
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card
          className="border-0 shadow-sm"
          style={{ backgroundColor: "white" }}
        >
          <CardContent className="pt-6 pb-6">
            <div className="flex items-center gap-4">
              <div
                className="p-3 rounded-xl"
                style={{ backgroundColor: "hsl(var(--slate-100))" }}
              >
                <Server
                  className="h-6 w-6"
                  style={{ color: "hsl(var(--slate-700))" }}
                />
              </div>
              <div>
                <div
                  className="text-sm font-medium"
                  style={{ color: "hsl(var(--slate-500))" }}
                >
                  Project Type
                </div>
                <div
                  className="text-lg font-semibold"
                  style={{ color: "hsl(var(--slate-800))" }}
                >
                  {projectSecrets.project_type === "monorepo"
                    ? "Monorepo"
                    : "Single Service"}
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Summary */}
      {projectSecrets.summary && (
        <Card
          className="border-0 shadow-sm"
          style={{ backgroundColor: "white" }}
        >
          <CardHeader style={{ backgroundColor: "hsl(var(--slate-50))" }}>
            <CardTitle style={{ color: "hsl(var(--slate-800))" }}>
              Configuration Summary
            </CardTitle>
          </CardHeader>
          <CardContent className="pt-6">
            <p style={{ color: "hsl(var(--slate-600))" }}>
              {projectSecrets.summary}
            </p>
          </CardContent>
        </Card>
      )}

      {/* Global Secrets */}
      {projectSecrets.global_secrets &&
        projectSecrets.global_secrets.length > 0 && (
          <Card
            className="border-0 shadow-sm"
            style={{ backgroundColor: "white" }}
          >
            <CardHeader style={{ backgroundColor: "hsl(var(--slate-50))" }}>
              <CardTitle
                className="flex items-center gap-3"
                style={{ color: "hsl(var(--slate-800))" }}
              >
                <div
                  className="p-2 rounded-lg"
                  style={{ backgroundColor: "hsl(var(--slate-100))" }}
                >
                  <Shield
                    className="h-6 w-6"
                    style={{ color: "hsl(var(--slate-700))" }}
                  />
                </div>
                Global Environment Variables
              </CardTitle>
              <CardDescription>
                Project-wide configuration variables that need to be set
              </CardDescription>
            </CardHeader>
            <CardContent className="pt-6">
              <div className="space-y-4">
                {projectSecrets.global_secrets.map((secret, index) => (
                  <SecretVariableCard key={index} secret={secret} />
                ))}
              </div>
            </CardContent>
          </Card>
        )}

      {/* Service-specific Secrets */}
      {projectSecrets.services && projectSecrets.services.length > 0 && (
        <div className="space-y-6">
          <h2
            className="text-2xl font-bold"
            style={{ color: "hsl(var(--slate-800))" }}
          >
            Service Configuration
          </h2>
          {projectSecrets.services.map((service, index) => (
            <Card
              key={index}
              className="border-0 shadow-sm"
              style={{ backgroundColor: "white" }}
            >
              <CardHeader style={{ backgroundColor: "hsl(var(--slate-50))" }}>
                <CardTitle
                  className="flex items-center gap-3"
                  style={{ color: "hsl(var(--slate-800))" }}
                >
                  <div
                    className="p-2 rounded-lg"
                    style={{ backgroundColor: "hsl(var(--slate-100))" }}
                  >
                    <Server
                      className="h-6 w-6"
                      style={{ color: "hsl(var(--slate-700))" }}
                    />
                  </div>
                  {service.service_name}
                </CardTitle>
                <CardDescription>
                  Path: {service.service_path} • Config files:{" "}
                  {service.config_files?.join(", ") || "None"}
                </CardDescription>
              </CardHeader>
              <CardContent className="pt-6">
                {service.variables && service.variables.length > 0 ? (
                  <div className="space-y-4">
                    {service.variables.map((secret, secretIndex) => (
                      <SecretVariableCard key={secretIndex} secret={secret} />
                    ))}
                  </div>
                ) : (
                  <div
                    className="text-center py-8"
                    style={{ color: "hsl(var(--slate-500))" }}
                  >
                    <Shield className="h-12 w-12 mx-auto mb-4 opacity-50" />
                    <p>No configuration variables found for this service</p>
                  </div>
                )}
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Setup Instructions */}
      {totalRequired > 0 && (
        <Card
          className="border-0 shadow-sm"
          style={{
            backgroundColor: "hsl(var(--blue-50))",
            borderColor: "hsl(var(--blue-200))",
          }}
        >
          <CardHeader>
            <CardTitle
              className="flex items-center gap-3"
              style={{ color: "hsl(var(--blue-800))" }}
            >
              <div
                className="p-2 rounded-lg"
                style={{ backgroundColor: "hsl(var(--blue-100))" }}
              >
                <Key
                  className="h-6 w-6"
                  style={{ color: "hsl(var(--blue-700))" }}
                />
              </div>
              Setup Instructions
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div
              className="space-y-4"
              style={{ color: "hsl(var(--blue-800))" }}
            >
              <p>
                <strong>
                  To run this project, you'll need to configure the following:
                </strong>
              </p>
              <ol className="list-decimal list-inside space-y-2 ml-4">
                <li>
                  Copy{" "}
                  <code className="bg-blue-100 px-2 py-1 rounded">
                    .env.example
                  </code>{" "}
                  to <code className="bg-blue-100 px-2 py-1 rounded">.env</code>{" "}
                  (if available)
                </li>
                <li>
                  Set values for the {totalRequired} required environment
                  variables shown above
                </li>
                <li>
                  Update any configuration files (config.yaml,
                  application.properties, etc.) with your values
                </li>
                <li>
                  For API keys and secrets, refer to the respective service
                  documentation
                </li>
                <li>
                  Ensure all services have access to their required environment
                  variables
                </li>
              </ol>
              <p className="text-sm mt-4">
                💡 <strong>Tip:</strong> Check each service's README or
                documentation for specific setup instructions and where to
                obtain API keys.
              </p>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

// Secret Variable Card Component
function SecretVariableCard({ secret }) {
  const [isExpanded, setIsExpanded] = useState(false);

  const getTypeColor = (type) => {
    const colors = {
      api_key: "hsl(var(--green-600))",
      database_url: "hsl(var(--blue-600))",
      secret: "hsl(var(--red-600))",
      credential: "hsl(var(--orange-600))",
      config: "hsl(var(--slate-600))",
    };
    return colors[type] || colors.config;
  };

  const getTypeBackground = (type) => {
    const backgrounds = {
      api_key: "hsl(var(--green-100))",
      database_url: "hsl(var(--blue-100))",
      secret: "hsl(var(--red-100))",
      credential: "hsl(var(--orange-100))",
      config: "hsl(var(--slate-100))",
    };
    return backgrounds[type] || backgrounds.config;
  };

  return (
    <div
      className="border rounded-lg p-4"
      style={{
        borderColor: "hsl(var(--slate-200))",
        backgroundColor: "hsl(var(--slate-50))",
      }}
    >
      <div className="flex items-start justify-between">
        <div className="flex items-start gap-3 flex-1">
          <div
            className="p-2 rounded-lg"
            style={{ backgroundColor: getTypeBackground(secret.type) }}
          >
            <Key
              className="h-4 w-4"
              style={{ color: getTypeColor(secret.type) }}
            />
          </div>
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-3 mb-2">
              <h4
                className="font-mono text-lg font-semibold"
                style={{ color: "hsl(var(--slate-800))" }}
              >
                {secret.name}
              </h4>
              <Badge
                variant="outline"
                style={{
                  backgroundColor: getTypeBackground(secret.type),
                  color: getTypeColor(secret.type),
                  borderColor: getTypeColor(secret.type),
                }}
              >
                {secret.type.replace("_", " ")}
              </Badge>
              {secret.required && <Badge variant="destructive">Required</Badge>}
            </div>
            <p
              className="text-sm mb-2"
              style={{ color: "hsl(var(--slate-600))" }}
            >
              {secret.description}
            </p>
            <div className="text-xs" style={{ color: "hsl(var(--slate-500))" }}>
              Source: {secret.source}
            </div>
          </div>
        </div>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => setIsExpanded(!isExpanded)}
          className="ml-2"
        >
          {isExpanded ? (
            <ChevronDown className="h-4 w-4" />
          ) : (
            <ChevronRight className="h-4 w-4" />
          )}
        </Button>
      </div>

      {isExpanded && secret.example && (
        <div
          className="mt-4 pt-4 border-t"
          style={{ borderColor: "hsl(var(--slate-200))" }}
        >
          <div className="space-y-2">
            <div
              className="text-sm font-medium"
              style={{ color: "hsl(var(--slate-700))" }}
            >
              Example:
            </div>
            <div
              className="font-mono text-sm p-3 rounded border"
              style={{
                backgroundColor: "hsl(var(--slate-100))",
                borderColor: "hsl(var(--slate-300))",
                color: "hsl(var(--slate-700))",
              }}
            >
              {secret.name}={secret.example}
            </div>
            <div className="text-xs" style={{ color: "hsl(var(--slate-500))" }}>
              Copy this to your .env file and replace with your actual value
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default App;
