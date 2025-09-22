// Mock data that matches the structure from our Go backend analyzer
export const mockAnalysisData = {
  projectInfo: {
    name: "E-commerce Platform",
    path: "/Users/dev/ecommerce-platform",
    type: "Backend",
    confidence: "Very High",
    architecture: "microservices",
    layout: "monorepo",
    primaryType: "Backend",
    secondaryType: "DevOps/Infrastructure",
    purpose: "An e-commerce platform for managing product catalogs, user accounts, and order processing with microservices architecture.",
    techStacks: ["Go", "React", "PostgreSQL", "Docker", "Kubernetes"],
    generatedAt: "2024-09-22T00:00:00Z",
    analysisTime: "15.42 seconds"
  },

  statistics: {
    filesAnalyzed: 156,
    totalSizeMB: 2.4,
    foldersAnalyzed: 23,
    linesOfCode: 12847,
    fileTypes: {
      ".go": 45,
      ".jsx": 32,
      ".sql": 12,
      ".yaml": 8,
      ".md": 6,
      ".json": 5
    }
  },

  services: [
    {
      name: "api-gateway",
      type: "HTTP",
      port: "8080",
      path: "cmd/api-gateway",
      apiType: "HTTP",
      entryPoint: "main.go",
      description: "Main API gateway handling external requests and routing",
      dependencies: ["user-service", "order-service", "product-service"],
      endpoints: [
        "GET /api/v1/health",
        "POST /api/v1/auth/login",
        "GET /api/v1/users/:id",
        "POST /api/v1/orders",
        "GET /api/v1/products"
      ]
    },
    {
      name: "user-service",
      type: "gRPC",
      port: "50051",
      path: "cmd/user-service",
      apiType: "gRPC",
      entryPoint: "main.go",
      description: "User management service handling authentication and profiles",
      dependencies: [],
      endpoints: [
        "rpc GetUser",
        "rpc CreateUser",
        "rpc UpdateUser",
        "rpc DeleteUser",
        "rpc AuthenticateUser"
      ]
    },
    {
      name: "order-service",
      type: "HTTP",
      port: "8082",
      path: "cmd/order-service",
      apiType: "HTTP",
      entryPoint: "main.go",
      description: "Order processing and management service",
      dependencies: ["user-service", "product-service", "payment-service"],
      endpoints: [
        "POST /orders",
        "GET /orders/:id",
        "PUT /orders/:id/status",
        "GET /orders/user/:userId"
      ]
    },
    {
      name: "product-service",
      type: "HTTP",
      port: "8083",
      path: "cmd/product-service",
      apiType: "HTTP",
      entryPoint: "main.go",
      description: "Product catalog and inventory management",
      dependencies: [],
      endpoints: [
        "GET /products",
        "GET /products/:id",
        "POST /products",
        "PUT /products/:id",
        "DELETE /products/:id"
      ]
    },
    {
      name: "payment-service",
      type: "HTTP",
      port: "8084",
      path: "cmd/payment-service",
      apiType: "HTTP",
      entryPoint: "main.go",
      description: "Payment processing and transaction management",
      dependencies: ["user-service"],
      endpoints: [
        "POST /payments/charge",
        "GET /payments/:id",
        "POST /payments/refund",
        "GET /payments/user/:userId"
      ]
    }
  ],

  relationships: [
    {
      from: "api-gateway",
      to: "user-service",
      evidenceType: "config",
      evidence: "Docker env: USER_SERVICE_URL=http://user-service:50051",
      filePath: "docker-compose.yml",
      confidence: 0.95
    },
    {
      from: "api-gateway", 
      to: "order-service",
      evidenceType: "config",
      evidence: "Docker env: ORDER_SERVICE_URL=http://order-service:8082",
      filePath: "docker-compose.yml",
      confidence: 0.95
    },
    {
      from: "order-service",
      to: "user-service",
      evidenceType: "network",
      evidence: "gRPC call: grpc.Dial('user-service:50051')",
      filePath: "cmd/order-service/internal/client/user.go",
      confidence: 0.85
    },
    {
      from: "order-service",
      to: "product-service",
      evidenceType: "network",
      evidence: "HTTP call: http://product-service:8083/products",
      filePath: "cmd/order-service/internal/handlers/order.go",
      confidence: 0.80
    },
    {
      from: "payment-service",
      to: "user-service",
      evidenceType: "import",
      evidence: "Import: 'github.com/company/platform/services/user/pkg/client'",
      filePath: "cmd/payment-service/internal/handlers/payment.go",
      confidence: 0.90
    }
  ],

  mermaidGraph: {
    mermaid: "graph TD\\n  api_gateway[API Gateway - HTTP]\\n  user_service{User Service - gRPC}\\n  order_service[Order Service - HTTP]\\n  product_service[Product Service - HTTP]\\n  payment_service[Payment Service - HTTP]\\n\\n  api_gateway -->|config| user_service\\n  api_gateway -->|config| order_service\\n  order_service -->|grpc| user_service\\n  order_service -->|http| product_service\\n  payment_service -->|import| user_service"
  },

  database: {
    migrationPath: "migrations",
    generatedAt: "2024-09-22T00:00:00Z",
    tables: {
      users: {
        name: "users",
        columns: {
          id: { name: "id", type: "bigint", constraints: ["PK"], primaryKey: true },
          email: { name: "email", type: "varchar(255)", constraints: ["not null", "unique"] },
          name: { name: "name", type: "varchar(255)", constraints: ["not null"] },
          password_hash: { name: "password_hash", type: "varchar(255)", constraints: ["not null"] },
          phone: { name: "phone", type: "varchar(20)", constraints: [] },
          email_verified: { name: "email_verified", type: "boolean", constraints: ["default FALSE"] },
          created_at: { name: "created_at", type: "timestamp", constraints: ["default CURRENT_TIMESTAMP"] },
          updated_at: { name: "updated_at", type: "timestamp", constraints: ["default CURRENT_TIMESTAMP"] }
        },
        primaryKeys: ["id"],
        indexes: { idx_users_email: { name: "idx_users_email", columns: ["email"], unique: true } }
      },
      orders: {
        name: "orders",
        columns: {
          id: { name: "id", type: "bigint", constraints: ["PK"], primaryKey: true },
          user_id: { name: "user_id", type: "bigint", constraints: ["not null", "FK"], references: { table: "users", column: "id" } },
          total: { name: "total", type: "decimal(10,2)", constraints: ["not null"] },
          status: { name: "status", type: "varchar(50)", constraints: ["default 'pending'"] },
          created_at: { name: "created_at", type: "timestamp", constraints: ["default CURRENT_TIMESTAMP"] },
          updated_at: { name: "updated_at", type: "timestamp", constraints: ["default CURRENT_TIMESTAMP"] }
        },
        primaryKeys: ["id"],
        indexes: { 
          idx_orders_user_id: { name: "idx_orders_user_id", columns: ["user_id"], unique: false },
          idx_orders_status: { name: "idx_orders_status", columns: ["status"], unique: false }
        }
      },
      products: {
        name: "products",
        columns: {
          id: { name: "id", type: "bigint", constraints: ["PK"], primaryKey: true },
          name: { name: "name", type: "varchar(255)", constraints: ["not null"] },
          description: { name: "description", type: "text", constraints: [] },
          price: { name: "price", type: "decimal(10,2)", constraints: ["not null"] },
          inventory_count: { name: "inventory_count", type: "integer", constraints: ["default 0"] },
          category_id: { name: "category_id", type: "bigint", constraints: ["FK"], references: { table: "categories", column: "id" } },
          created_at: { name: "created_at", type: "timestamp", constraints: ["default CURRENT_TIMESTAMP"] },
          updated_at: { name: "updated_at", type: "timestamp", constraints: ["default CURRENT_TIMESTAMP"] }
        },
        primaryKeys: ["id"],
        indexes: {}
      },
      categories: {
        name: "categories",
        columns: {
          id: { name: "id", type: "bigint", constraints: ["PK"], primaryKey: true },
          name: { name: "name", type: "varchar(255)", constraints: ["not null", "unique"] },
          description: { name: "description", type: "text", constraints: [] },
          created_at: { name: "created_at", type: "timestamp", constraints: ["default CURRENT_TIMESTAMP"] }
        },
        primaryKeys: ["id"],
        indexes: {}
      },
      order_items: {
        name: "order_items", 
        columns: {
          id: { name: "id", type: "bigint", constraints: ["PK"], primaryKey: true },
          order_id: { name: "order_id", type: "bigint", constraints: ["not null", "FK"], references: { table: "orders", column: "id" } },
          product_id: { name: "product_id", type: "bigint", constraints: ["not null", "FK"], references: { table: "products", column: "id" } },
          quantity: { name: "quantity", type: "integer", constraints: ["not null", "default 1"] },
          price: { name: "price", type: "decimal(10,2)", constraints: ["not null"] },
          created_at: { name: "created_at", type: "timestamp", constraints: ["default CURRENT_TIMESTAMP"] }
        },
        primaryKeys: ["id"],
        indexes: {
          idx_order_items_order_id: { name: "idx_order_items_order_id", columns: ["order_id"], unique: false },
          idx_order_items_product_id: { name: "idx_order_items_product_id", columns: ["product_id"], unique: false },
          unique_order_product: { name: "unique_order_product", columns: ["order_id", "product_id"], unique: true }
        }
      }
    },
    relationships: [
      { from: "orders", fromColumn: "user_id", to: "users", toColumn: "id" },
      { from: "products", fromColumn: "category_id", to: "categories", toColumn: "id" },
      { from: "order_items", fromColumn: "order_id", to: "orders", toColumn: "id" },
      { from: "order_items", fromColumn: "product_id", to: "products", toColumn: "id" }
    ]
  },

  files: [
    { path: "cmd/api-gateway/main.go", type: "go", size: "2.1 KB", lines: 87 },
    { path: "cmd/user-service/main.go", type: "go", size: "1.8 KB", lines: 72 },
    { path: "internal/database/user.go", type: "go", size: "3.2 KB", lines: 134 },
    { path: "migrations/001_create_users.sql", type: "sql", size: "0.4 KB", lines: 12 },
    { path: "docker-compose.yml", type: "yaml", size: "1.2 KB", lines: 45 },
    { path: "README.md", type: "markdown", size: "2.8 KB", lines: 89 }
  ]
}

// Helper functions for data transformation
export const getServiceTypeColor = (type) => {
  switch (type?.toLowerCase()) {
    case 'grpc': return 'warning'
    case 'http': return 'primary'
    case 'graphql': return 'success'
    default: return 'secondary'
  }
}

export const getConfidenceColor = (confidence) => {
  if (confidence >= 0.9) return 'success'
  if (confidence >= 0.7) return 'warning'
  return 'error'
}

export const getEvidenceTypeIcon = (evidenceType) => {
  switch (evidenceType) {
    case 'config': return 'Settings'
    case 'import': return 'Package'
    case 'network': return 'Network'
    default: return 'Link'
  }
}
