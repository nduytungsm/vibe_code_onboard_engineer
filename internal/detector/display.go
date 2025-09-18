package detector

import (
	"fmt"
	"strings"
)

// DisplayResult formats and prints the project type detection results
func (dr *DetectionResult) DisplayResult() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("🔍 PROJECT TYPE DETECTION RESULTS")
	fmt.Println(strings.Repeat("=", 60))

	// Primary type with confidence
	confidenceBar := generateConfidenceBar(dr.Confidence)
	fmt.Printf("\n🎯 PRIMARY TYPE: %s\n", string(dr.PrimaryType))
	fmt.Printf("📊 CONFIDENCE: %.1f/10 %s\n", dr.Confidence, confidenceBar)

	// Secondary type if present
	if dr.SecondaryType != "" {
		fmt.Printf("🔄 SECONDARY: %s\n", string(dr.SecondaryType))
	}

	// Evidence section
	if len(dr.Evidence) > 0 {
		fmt.Println("\n🔍 DETECTION EVIDENCE:")
		for projectType, evidenceList := range dr.Evidence {
			if len(evidenceList) > 0 {
				fmt.Printf("  %s:\n", projectType)
				for _, evidence := range evidenceList {
					fmt.Printf("    • %s\n", evidence)
				}
			}
		}
	}

	// Detailed scores
	fmt.Println("\n📈 DETAILED SCORES:")
	scores := []struct {
		Type  ProjectType
		Score float64
	}{
		{Frontend, dr.Scores[Frontend]},
		{Backend, dr.Scores[Backend]},
		{Mobile, dr.Scores[Mobile]},
		{Desktop, dr.Scores[Desktop]},
		{Library, dr.Scores[Library]},
		{DevOps, dr.Scores[DevOps]},
		{DataScience, dr.Scores[DataScience]},
	}

	for _, score := range scores {
		if score.Score > 0 {
			bar := generateScoreBar(score.Score)
			fmt.Printf("  %-20s %.1f %s\n", string(score.Type), score.Score, bar)
		}
	}

	// Interpretation
	fmt.Println("\n💡 INTERPRETATION:")
	interpretation := dr.GetInterpretation()
	fmt.Printf("   %s\n", interpretation)

	fmt.Println("\n" + strings.Repeat("=", 60))
}

// GetInterpretation provides a human-readable interpretation of the results
func (dr *DetectionResult) GetInterpretation() string {
	switch dr.PrimaryType {
	case Frontend:
		if dr.Confidence >= 7.0 {
			return "This is clearly a frontend project with strong indicators like UI frameworks, styling files, and client-side code."
		} else if dr.Confidence >= 4.0 {
			return "This appears to be primarily a frontend project, though it may have some mixed characteristics."
		} else {
			return "This has frontend characteristics but the evidence is limited. May be a mixed or minimal project."
		}

	case Backend:
		if dr.Confidence >= 7.0 {
			return "This is clearly a backend/server-side project with strong indicators like APIs, databases, and server frameworks."
		} else if dr.Confidence >= 4.0 {
			return "This appears to be primarily a backend project, though it may have some mixed characteristics."
		} else {
			return "This has backend characteristics but the evidence is limited. May be a mixed or minimal project."
		}

	case Fullstack:
		return "This is a fullstack project containing both frontend and backend components, providing a complete application solution."

	case Mobile:
		if dr.Confidence >= 7.0 {
			return "This is clearly a mobile application project with mobile-specific frameworks and tools."
		} else {
			return "This appears to be a mobile project, possibly cross-platform or in early development."
		}

	case Desktop:
		return "This is a desktop application project designed to run on desktop operating systems."

	case Library:
		if dr.Confidence >= 5.0 {
			return "This is a library or package designed to be used by other applications, with proper packaging and documentation."
		} else {
			return "This appears to be a library or utility project, though it may be in early development."
		}

	case DevOps:
		return "This is a DevOps/Infrastructure project focused on deployment, orchestration, or infrastructure management."

	case DataScience:
		return "This is a data science or analytics project, likely involving data processing, analysis, or machine learning."

	case Unknown:
		if len(dr.Evidence) > 0 {
			return "Project type could not be clearly determined. It may be a mixed project, configuration files, or documentation."
		} else {
			return "Insufficient evidence to determine project type. The project may be empty or contain unsupported file types."
		}

	default:
		return "Project classification completed with available evidence."
	}
}

// generateConfidenceBar creates a visual confidence indicator
func generateConfidenceBar(confidence float64) string {
	maxBars := 10
	filledBars := int(confidence)
	if filledBars > maxBars {
		filledBars = maxBars
	}

	bar := "["
	for i := 0; i < filledBars; i++ {
		bar += "█"
	}
	for i := filledBars; i < maxBars; i++ {
		bar += "░"
	}
	bar += "]"

	// Add confidence level indicator
	if confidence >= 8.0 {
		bar += " (Very High)"
	} else if confidence >= 6.0 {
		bar += " (High)"
	} else if confidence >= 4.0 {
		bar += " (Medium)"
	} else if confidence >= 2.0 {
		bar += " (Low)"
	} else {
		bar += " (Very Low)"
	}

	return bar
}

// generateScoreBar creates a visual score indicator
func generateScoreBar(score float64) string {
	maxBars := 15
	filledBars := int(score)
	if filledBars > maxBars {
		filledBars = maxBars
	}

	bar := "["
	for i := 0; i < filledBars; i++ {
		if i < 5 {
			bar += "▓" // Different intensity for lower scores
		} else if i < 10 {
			bar += "█" // Solid for medium scores
		} else {
			bar += "█" // Solid for high scores
		}
	}
	for i := filledBars; i < maxBars; i++ {
		bar += "░"
	}
	bar += "]"

	return bar
}

// PrintSummary prints a concise summary of the detection result
func (dr *DetectionResult) PrintSummary() {
	typeEmoji := getTypeEmoji(dr.PrimaryType)
	confidenceLevel := getConfidenceLevel(dr.Confidence)
	
	fmt.Printf("\n%s PROJECT TYPE: %s (%s confidence)\n", 
		typeEmoji, string(dr.PrimaryType), confidenceLevel)
	
	if dr.SecondaryType != "" {
		secondaryEmoji := getTypeEmoji(dr.SecondaryType)
		fmt.Printf("%s Secondary: %s\n", secondaryEmoji, string(dr.SecondaryType))
	}
}

// getTypeEmoji returns appropriate emoji for project type
func getTypeEmoji(projectType ProjectType) string {
	switch projectType {
	case Frontend:
		return "🎨"
	case Backend:
		return "⚙️"
	case Fullstack:
		return "🌐"
	case Mobile:
		return "📱"
	case Desktop:
		return "🖥️"
	case Library:
		return "📚"
	case DevOps:
		return "🚀"
	case DataScience:
		return "📊"
	default:
		return "❓"
	}
}

// getConfidenceLevel returns human-readable confidence level
func getConfidenceLevel(confidence float64) string {
	if confidence >= 8.0 {
		return "Very High"
	} else if confidence >= 6.0 {
		return "High"
	} else if confidence >= 4.0 {
		return "Medium"
	} else if confidence >= 2.0 {
		return "Low"
	} else {
		return "Very Low"
	}
}
