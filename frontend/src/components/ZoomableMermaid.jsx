import React, { useRef, useEffect, useState, useCallback } from 'react';
import { ZoomIn, ZoomOut, RotateCcw, Maximize2, Download, Copy } from 'lucide-react';

const ZoomableMermaid = ({ 
  mermaidCode, 
  title = "Diagram",
  className = "",
  containerClassName = "",
  showTitle = true,
  showControls = true,
  minZoom = 0.1,
  maxZoom = 5.0,
  initialZoom = 1.0,
  onError = null,
  config = {}
}) => {
  const containerRef = useRef(null);
  const mermaidRef = useRef(null);
  const renderingRef = useRef(false); // Track rendering state to prevent concurrent renders
  const [zoom, setZoom] = useState(initialZoom);
  const [isRendering, setIsRendering] = useState(false);
  const [renderError, setRenderError] = useState(null);
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });
  const [transform, setTransform] = useState({ x: 0, y: 0 });

  // Default Mermaid configuration with accessibility and visual improvements
  const defaultConfig = {
    startOnLoad: false,
    theme: 'default',
    themeVariables: {
      fontSize: '16px',
      fontFamily: 'Inter, system-ui, sans-serif',
      primaryColor: '#3b82f6',
      primaryTextColor: '#1e293b',
      primaryBorderColor: '#2563eb',
      lineColor: '#64748b',
      secondaryColor: '#e2e8f0',
      tertiaryColor: '#f8fafc',
      background: '#ffffff',
      mainBkg: '#ffffff',
      secondBkg: '#f1f5f9',
      tertiaryBkg: '#f8fafc'
    },
    er: {
      fontSize: 14,
      useMaxWidth: false
    },
    flowchart: {
      useMaxWidth: false,
      nodeSpacing: 50,
      rankSpacing: 50
    },
    ...config
  };

  // Zoom functions
  const handleZoomIn = useCallback(() => {
    setZoom(prev => Math.min(prev * 1.25, maxZoom));
  }, [maxZoom]);

  const handleZoomOut = useCallback(() => {
    setZoom(prev => Math.max(prev / 1.25, minZoom));
  }, [minZoom]);

  const handleResetZoom = useCallback(() => {
    setZoom(initialZoom);
    setTransform({ x: 0, y: 0 });
  }, [initialZoom]);

  const handleFitToScreen = useCallback(() => {
    if (containerRef.current && mermaidRef.current) {
      const container = containerRef.current;
      const diagram = mermaidRef.current.firstChild;
      
      if (diagram) {
        try {
          const containerRect = container.getBoundingClientRect();
          const diagramRect = diagram.getBoundingClientRect();
          
          // Only calculate if both elements have valid dimensions
          if (containerRect.width > 0 && containerRect.height > 0 && 
              diagramRect.width > 0 && diagramRect.height > 0) {
            const scaleX = (containerRect.width - 40) / diagramRect.width;
            const scaleY = (containerRect.height - 40) / diagramRect.height;
            const optimalZoom = Math.min(scaleX, scaleY, maxZoom);
            
            setZoom(Math.max(optimalZoom, minZoom));
            setTransform({ x: 0, y: 0 });
          }
        } catch (error) {
          console.warn('Fit to screen calculation failed:', error);
          // Fallback to reset zoom
          setZoom(initialZoom);
          setTransform({ x: 0, y: 0 });
        }
      }
    }
  }, [minZoom, maxZoom, initialZoom]);

  // Copy functions
  const handleCopyMermaid = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(mermaidCode);
      // You could add a toast notification here
    } catch (err) {
      console.error('Failed to copy Mermaid code:', err);
    }
  }, [mermaidCode]);

  const handleDownloadSVG = useCallback(() => {
    const svgElement = mermaidRef.current?.querySelector('svg');
    if (svgElement) {
      try {
        const svgData = new XMLSerializer().serializeToString(svgElement);
        const blob = new Blob([svgData], { type: 'image/svg+xml' });
        const url = URL.createObjectURL(blob);
        
        const link = document.createElement('a');
        link.href = url;
        link.download = `${title.toLowerCase().replace(/\s+/g, '-')}.svg`;
        link.style.display = 'none';
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        
        URL.revokeObjectURL(url);
      } catch (error) {
        console.warn('SVG download failed:', error);
      }
    }
  }, [title]);

  // Drag functions
  const handleMouseDown = useCallback((e) => {
    if (e.target.closest('.zoom-controls')) return;
    setIsDragging(true);
    setDragStart({ x: e.clientX - transform.x, y: e.clientY - transform.y });
    e.preventDefault();
  }, [transform]);

  const handleMouseMove = useCallback((e) => {
    if (!isDragging) return;
    setTransform({
      x: e.clientX - dragStart.x,
      y: e.clientY - dragStart.y
    });
  }, [isDragging, dragStart]);

  const handleMouseUp = useCallback(() => {
    setIsDragging(false);
  }, []);

  // Wheel zoom
  const handleWheel = useCallback((e) => {
    if (e.ctrlKey || e.metaKey) {
      e.preventDefault();
      const delta = e.deltaY > 0 ? 0.9 : 1.1;
      setZoom(prev => Math.max(minZoom, Math.min(maxZoom, prev * delta)));
    }
  }, [minZoom, maxZoom]);

  // Render Mermaid diagram
  const renderMermaid = useCallback(async () => {
    if (!mermaidCode || !mermaidRef.current || renderingRef.current) return;

    renderingRef.current = true;
    setIsRendering(true);
    setRenderError(null);

    try {
      const mermaid = (await import('mermaid')).default;
      
      // Initialize Mermaid with our config
      mermaid.initialize(defaultConfig);

      // Generate unique ID for this diagram
      const diagramId = `mermaid-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
      
      // Clear the container and set up properly
      mermaidRef.current.innerHTML = '';
      
      // Create a div that will contain the Mermaid diagram
      const diagramContainer = document.createElement('div');
      diagramContainer.id = diagramId;
      diagramContainer.innerHTML = mermaidCode;
      
      // Append to our ref container BEFORE rendering (this ensures it's in DOM)
      mermaidRef.current.appendChild(diagramContainer);
      
      // Now render the diagram - the element is properly attached to DOM
      await mermaid.run({
        nodes: [diagramContainer]
      });
      
      // Find and configure the SVG
      const svg = diagramContainer.querySelector('svg');
      if (svg) {
        svg.style.maxWidth = 'none';
        svg.style.height = 'auto';
        svg.setAttribute('preserveAspectRatio', 'xMidYMid meet');
        
        // Ensure the SVG has proper dimensions
        if (!svg.getAttribute('width') || !svg.getAttribute('height')) {
          svg.setAttribute('width', '100%');
          svg.setAttribute('height', 'auto');
        }
      }

      // Add a small delay before hiding loading state to prevent flickering
      setTimeout(() => {
        setIsRendering(false);
        renderingRef.current = false;
      }, 10);

    } catch (error) {
      console.error('Error rendering Mermaid diagram:', error);
      setRenderError(error.message);
      if (onError) onError(error);
      
      // Show error in the container
      if (mermaidRef.current) {
        mermaidRef.current.innerHTML = `
          <div class="flex flex-col items-center justify-center p-8 text-center">
            <div class="text-red-500 mb-4">
              <svg class="h-12 w-12 mx-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z" />
              </svg>
            </div>
            <h3 class="text-lg font-medium text-gray-900 mb-2">Diagram Rendering Error</h3>
            <p class="text-sm text-gray-600 mb-4">Unable to render the Mermaid diagram</p>
            <details class="text-left">
              <summary class="cursor-pointer text-sm font-medium text-gray-700 mb-2">Error Details</summary>
              <pre class="text-xs bg-gray-100 p-2 rounded overflow-x-auto">${error.message}</pre>
            </details>
          </div>
        `;
      }
      setIsRendering(false);
      renderingRef.current = false;
    }
  }, [defaultConfig, onError]);

  // Effect to render diagram when code changes
  useEffect(() => {
    if (mermaidCode) {
      renderMermaid();
    }
  }, [mermaidCode]); // Remove renderMermaid dependency to prevent infinite re-renders

  // Effect for mouse event listeners
  useEffect(() => {
    if (isDragging) {
      document.addEventListener('mousemove', handleMouseMove);
      document.addEventListener('mouseup', handleMouseUp);
      return () => {
        document.removeEventListener('mousemove', handleMouseMove);
        document.removeEventListener('mouseup', handleMouseUp);
      };
    }
  }, [isDragging, handleMouseMove, handleMouseUp]);

  if (!mermaidCode) {
    return (
      <div className={`border rounded-lg p-6 ${containerClassName}`} style={{ backgroundColor: "hsl(var(--slate-50))", borderColor: "hsl(var(--slate-200))" }}>
        <div className="text-center py-8" style={{ color: "hsl(var(--slate-500))" }}>
          <div className="text-sm">No diagram data available</div>
        </div>
      </div>
    );
  }

  return (
    <div className={`relative border rounded-lg overflow-hidden ${containerClassName}`} style={{ backgroundColor: "hsl(var(--slate-50))", borderColor: "hsl(var(--slate-200))" }}>
      {/* Title */}
      {showTitle && (
        <div className="px-4 py-2 border-b" style={{ backgroundColor: "white", borderColor: "hsl(var(--slate-200))" }}>
          <div className="text-sm font-medium" style={{ color: "hsl(var(--slate-700))" }}>
            {title}
          </div>
        </div>
      )}

      {/* Zoom Controls */}
      {showControls && (
        <div className="absolute top-2 right-2 z-10 zoom-controls">
          <div className="flex items-center gap-1 p-1 rounded-lg shadow-sm" style={{ backgroundColor: "white", borderColor: "hsl(var(--slate-200))" }}>
            <button
              onClick={handleZoomOut}
              disabled={zoom <= minZoom}
              className="p-1.5 rounded hover:bg-gray-100 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              title="Zoom Out"
            >
              <ZoomOut className="h-4 w-4" />
            </button>
            
            <div className="px-2 py-1 text-xs font-mono min-w-12 text-center" style={{ color: "hsl(var(--slate-600))" }}>
              {Math.round(zoom * 100)}%
            </div>
            
            <button
              onClick={handleZoomIn}
              disabled={zoom >= maxZoom}
              className="p-1.5 rounded hover:bg-gray-100 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              title="Zoom In"
            >
              <ZoomIn className="h-4 w-4" />
            </button>
            
            <div className="w-px h-4 bg-gray-300 mx-1" />
            
            <button
              onClick={handleFitToScreen}
              className="p-1.5 rounded hover:bg-gray-100 transition-colors"
              title="Fit to Screen"
            >
              <Maximize2 className="h-4 w-4" />
            </button>
            
            <button
              onClick={handleResetZoom}
              className="p-1.5 rounded hover:bg-gray-100 transition-colors"
              title="Reset View"
            >
              <RotateCcw className="h-4 w-4" />
            </button>

            <div className="w-px h-4 bg-gray-300 mx-1" />

            <button
              onClick={handleCopyMermaid}
              className="p-1.5 rounded hover:bg-gray-100 transition-colors"
              title="Copy Mermaid Code"
            >
              <Copy className="h-4 w-4" />
            </button>

            <button
              onClick={handleDownloadSVG}
              className="p-1.5 rounded hover:bg-gray-100 transition-colors"
              title="Download SVG"
            >
              <Download className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}

      {/* Diagram Container */}
      <div
        ref={containerRef}
        className={`relative overflow-hidden ${className}`}
        style={{ 
          height: '400px',
          cursor: isDragging ? 'grabbing' : 'grab',
          userSelect: 'none'
        }}
        onMouseDown={handleMouseDown}
        onWheel={handleWheel}
      >
        {/* Loading State */}
        {isRendering && (
          <div className="absolute inset-0 flex items-center justify-center bg-white bg-opacity-75 z-20">
            <div className="text-center">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-2"></div>
              <div className="text-sm" style={{ color: "hsl(var(--slate-600))" }}>Rendering diagram...</div>
            </div>
          </div>
        )}

        {/* Diagram Content */}
        <div
          ref={mermaidRef}
          className="absolute top-1/2 left-1/2 transition-transform duration-200"
          style={{
            transform: `translate(-50%, -50%) translate(${transform.x}px, ${transform.y}px) scale(${zoom})`,
            transformOrigin: 'center center'
          }}
        />
      </div>

      {/* Usage Instructions */}
      {showControls && (
        <div className="px-4 py-2 border-t text-xs" style={{ backgroundColor: "hsl(var(--slate-50))", borderColor: "hsl(var(--slate-200))", color: "hsl(var(--slate-500))" }}>
          <div className="flex items-center justify-between">
            <span>💡 Drag to pan • Ctrl+Scroll to zoom • Use controls to navigate</span>
            <span>Zoom: {Math.round(zoom * 100)}%</span>
          </div>
        </div>
      )}
    </div>
  );
};

export default ZoomableMermaid;
