# Repository Analyzer Frontend

A modern React dashboard for visualizing repository analysis data including project insights, service relationships, and database schemas.

## 🚀 Tech Stack

- **React 19** - Latest React with modern features
- **Vite** - Lightning fast build tool and dev server
- **Tailwind CSS v4** - Latest utility-first CSS framework with @theme configuration
- **@tailwindcss/vite** - Native Vite plugin for optimal performance
- **Lucide React** - Beautiful & consistent icon library
- **Axios** - HTTP client for API communication
- **React Router** - Client-side routing (ready for future use)
- **Headless UI** - Unstyled, accessible UI components

## 🎨 Features

- **Modern Dashboard UI** - Clean, professional interface
- **Responsive Design** - Works perfectly on desktop, tablet, and mobile
- **Interactive Navigation** - Tabbed interface for different data views
- **Component Library** - Pre-built cards, badges, buttons with consistent styling
- **Dark Mode Ready** - Built with Tailwind's theming system
- **Performance Optimized** - Fast loading and smooth interactions

## 📊 Dashboard Sections

- **Overview** - Project summary, confidence metrics, quick stats
- **Services** - Discovered microservices with API types and ports
- **Database** - Database tables and schema visualization
- **Dependencies** - Service relationship graphs (Coming Soon)
- **Files** - Project file structure analysis (Coming Soon)
- **Analysis** - Detailed architectural insights (Coming Soon)

## 🛠️ Development

### Prerequisites

- Node.js 18+ 
- npm or yarn

### Getting Started

```bash
# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview
```

The app will be available at `http://localhost:5173`

### Available Scripts

- `npm run dev` - Start development server with host binding
- `npm run build` - Build for production
- `npm run preview` - Preview production build
- `npm run lint` - Run ESLint
- `npm run format` - Format code with Prettier

## 🎯 Integration Points

The frontend is designed to integrate with the Go backend analyzer:

- **Project Analysis API** - Fetch repository analysis results
- **Service Discovery API** - Get microservice information
- **Database Schema API** - Retrieve ERD and schema data
- **Relationship API** - Get service dependency graphs
- **File Analysis API** - Access detailed file insights

## 🎨 Design System

### Colors
- **Primary**: Blue (#3b82f6) for main actions and highlights
- **Success**: Green for positive states and metrics
- **Warning**: Yellow for warnings and gRPC services
- **Error**: Red for errors and critical states
- **Gray Scale**: Consistent gray palette for text and backgrounds

### Components
- **Cards** - White containers with subtle shadows
- **Badges** - Colored indicators for types and statuses
- **Buttons** - Primary, secondary, and ghost variants
- **Navigation** - Clean tab-based interface

### Typography
- **Headings** - Inter font with semibold weight
- **Body** - Inter font for readability  
- **Code** - Fira Code for monospace content

### Tailwind CSS v4 Configuration
This project uses Tailwind CSS v4 with the modern `@theme` configuration approach:
- Custom theme defined in `src/index.css` using `@theme` directive
- Uses `@tailwindcss/vite` plugin for optimal build performance
- No separate `tailwind.config.js` file needed
- CSS-first configuration for better performance and developer experience

## 📁 Project Structure

```
frontend/
├── src/
│   ├── components/     # Reusable UI components (future)
│   ├── pages/         # Page components (future)
│   ├── hooks/         # Custom React hooks (future)
│   ├── utils/         # Utility functions (future)
│   ├── App.jsx        # Main application component
│   ├── main.jsx       # Application entry point
│   └── index.css      # Global styles with Tailwind
├── public/            # Static assets
├── package.json       # Dependencies and scripts
├── tailwind.config.js # Tailwind configuration
├── postcss.config.js  # PostCSS configuration
└── vite.config.js     # Vite configuration
```

## 🚀 Future Enhancements

- Real API integration with Go backend
- Interactive service dependency graphs
- Database ERD visualization
- File tree explorer
- Code syntax highlighting
- Real-time analysis updates
- Export functionality for reports
- Advanced filtering and search
