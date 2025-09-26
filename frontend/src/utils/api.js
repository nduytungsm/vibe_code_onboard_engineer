import axios from "axios";

// API base configuration
const API_BASE_URL =
  import.meta.env.VITE_API_URL || "http://13.239.135.39:8080/api";

const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: 35 * 60 * 1000, // 35 minutes (5 minutes buffer over backend timeout)
  headers: {
    "Content-Type": "application/json",
  },
});

// Request interceptor for logging
api.interceptors.request.use(
  (config) => {
    console.log(
      `🚀 API Request: ${config.method?.toUpperCase()} ${config.url}`
    );
    return config;
  },
  (error) => {
    console.error("❌ API Request Error:", error);
    return Promise.reject(error);
  }
);

// Response interceptor for error handling
api.interceptors.response.use(
  (response) => {
    console.log(`✅ API Response: ${response.status} ${response.config.url}`);
    return response;
  },
  (error) => {
    console.error(
      "❌ API Response Error:",
      error.response?.data || error.message
    );
    return Promise.reject(error);
  }
);

// Repository Analysis API
export const repositoryAPI = {
  // Analyze a new repository with streaming progress (recommended)
  analyzeRepositoryStream: (
    repositoryUrl,
    token = null,
    onProgress,
    onComplete,
    onError
  ) => {
    const payload = {
      url: repositoryUrl,
      type: "github_url",
    };

    // Add token if provided
    if (token) {
      payload.token = token;
    }

    let isCompleted = false;
    let lastProgressData = null;

    // Set up a timeout to handle cases where stream doesn't complete properly
    const timeoutId = setTimeout(() => {
      if (!isCompleted) {
        console.warn(
          "⚠️ Stream timeout reached, checking if we have usable data"
        );
        // If we received substantial progress data, treat it as a completion
        if (lastProgressData && lastProgressData.progress >= 75) {
          console.log("✅ Using last received data as completion");
          isCompleted = true;
          onComplete?.(lastProgressData.data || lastProgressData);
        } else {
          onError?.("Analysis timeout - please try again");
        }
      }
    }, 35 * 60 * 1000); // 35 minute timeout

    // Use fetch with streaming response for SSE
    fetch(`${API_BASE_URL}/analyze/stream`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(payload),
    })
      .then((response) => {
        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status}`);
        }

        // Check if response is SSE
        const contentType = response.headers.get("content-type");
        if (!contentType?.includes("text/event-stream")) {
          throw new Error("Server did not return an event stream");
        }

        const reader = response.body?.getReader();
        if (!reader) {
          throw new Error("Response body is not readable");
        }

        const decoder = new TextDecoder();
        let buffer = "";

        const readStream = () => {
          reader
            .read()
            .then(({ done, value }) => {
              if (done) {
                console.log("📡 Stream completed by server");
                clearTimeout(timeoutId);

                // If we haven't received a completion event but have good data, use it
                if (
                  !isCompleted &&
                  lastProgressData &&
                  lastProgressData.progress >= 90
                ) {
                  console.log(
                    "✅ Stream ended with substantial data, treating as completion"
                  );
                  isCompleted = true;
                  onComplete?.(lastProgressData.data || lastProgressData);
                } else if (!isCompleted) {
                  console.warn("⚠️ Stream ended without completion event");
                  onError?.("Analysis incomplete - stream ended unexpectedly");
                }
                return;
              }

              // Decode the chunk and add to buffer
              buffer += decoder.decode(value, { stream: true });

              // Process complete messages
              const messages = buffer.split("\n\n");
              buffer = messages.pop() || ""; // Keep incomplete message in buffer

              messages.forEach((message) => {
                if (message.trim() === "") return;

                // Parse SSE message
                const lines = message.split("\n");
                let eventData = "";

                lines.forEach((line) => {
                  if (line.startsWith("data: ")) {
                    eventData = line.substring(6);
                  }
                });

                if (eventData) {
                  try {
                    const data = JSON.parse(eventData);

                    // Store the latest progress data
                    lastProgressData = data;

                    switch (data.type) {
                      case "progress":
                        console.log(
                          `📊 Progress: ${data.progress}% - ${data.stage}`
                        );
                        onProgress?.(
                          data.stage,
                          data.message,
                          data.progress,
                          data.data
                        );
                        break;
                      case "data":
                        console.log(
                          `📊 Data event: ${data.progress}% - ${data.stage}`
                        );
                        onProgress?.(
                          data.stage,
                          data.message,
                          data.progress,
                          data.data
                        );
                        break;
                      case "complete":
                        console.log("🎉 Analysis completed successfully");
                        clearTimeout(timeoutId);
                        isCompleted = true;
                        onComplete?.(data.data);
                        return; // Stop reading
                      case "error":
                        console.error(
                          "❌ Analysis error:",
                          data.error || data.message
                        );
                        clearTimeout(timeoutId);
                        isCompleted = true;
                        onError?.(data.error || data.message);
                        return; // Stop reading
                      default:
                        console.log("Unknown event type:", data.type, data);
                    }
                  } catch (error) {
                    console.error("Error parsing SSE data:", error, eventData);
                    clearTimeout(timeoutId);
                    isCompleted = true;
                    onError?.("Failed to parse server response");
                    return;
                  }
                }
              });

              // Continue reading
              readStream();
            })
            .catch((error) => {
              console.error("Error reading stream:", error);
              clearTimeout(timeoutId);
              if (!isCompleted) {
                isCompleted = true;
                onError?.("Stream reading error: " + error.message);
              }
            });
        };

        // Start reading the stream
        readStream();

        // Return cleanup function
        return () => {
          reader.cancel();
        };
      })
      .catch((error) => {
        console.error("Failed to initiate streaming analysis:", error);
        clearTimeout(timeoutId);
        if (!isCompleted) {
          isCompleted = true;
          onError?.(error.message);
        }
      });
  },

  // Analyze a new repository (legacy method for backward compatibility)
  analyzeRepository: async (repositoryUrl, token = null) => {
    const payload = {
      url: repositoryUrl,
      type: "github_url",
    };

    // Add token if provided
    if (token) {
      payload.token = token;
    }

    const response = await api.post("/analyze", payload);
    return response.data;
  },

  // Get project analysis results
  getProjectAnalysis: async (analysisId) => {
    const response = await api.get(`/analysis/${analysisId}`);
    return response.data;
  },

  // Get discovered services
  getServices: async (analysisId) => {
    const response = await api.get(`/analysis/${analysisId}/services`);
    return response.data;
  },

  // Get service relationships
  getRelationships: async (analysisId) => {
    const response = await api.get(`/analysis/${analysisId}/relationships`);
    return response.data;
  },

  // Get database schema
  getDatabaseSchema: async (analysisId) => {
    const response = await api.get(`/analysis/${analysisId}/database`);
    return response.data;
  },

  // Get file analysis
  getFileAnalysis: async (analysisId) => {
    const response = await api.get(`/analysis/${analysisId}/files`);
    return response.data;
  },

  // Get Mermaid graph data
  getMermaidGraph: async (analysisId) => {
    const response = await api.get(`/analysis/${analysisId}/mermaid`);
    return response.data;
  },

  // Get PlantUML ERD
  getPlantUMLERD: async (analysisId) => {
    const response = await api.get(`/analysis/${analysisId}/plantuml`);
    return response.data;
  },
};

// Health check
export const healthAPI = {
  check: async () => {
    const response = await api.get("/health");
    return response.data;
  },
};

export default api;
