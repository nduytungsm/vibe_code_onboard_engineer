package cli

import (
	"fmt"
	"strings"
	"repo-explanation/internal/secrets"
)

// CommandHandler defines the interface for command handling
type CommandHandler interface {
	Handle(args []string) string
}

// TryMeCommand handles the "try me" command
type TryMeCommand struct{}

func (t *TryMeCommand) Handle(args []string) string {
	return "i am here"
}

// EndCommand handles the "/end" command
type EndCommand struct{}

func (e *EndCommand) Handle(args []string) string {
	return "Goodbye! 👋"
}

// SecretCommand handles secret extraction for a given folder path
type SecretCommand struct{}

func (s *SecretCommand) Handle(args []string) string {
	if len(args) == 0 {
		return "❌ Please provide a folder path. Usage: secrets /path/to/project"
	}
	
	folderPath := strings.Join(args, " ")
	
	fmt.Printf("🔍 Extracting secrets from: %s\n", folderPath)
	
	// Create secret extractor
	extractor := secrets.NewSecretExtractor(folderPath)
	
	// Extract secrets from configuration files
	projectSecrets, err := extractor.ExtractSecrets()
	if err != nil {
		return fmt.Sprintf("❌ Secret extraction failed: %v", err)
	}
	
	if projectSecrets == nil || projectSecrets.TotalVariables == 0 {
		return "✅ No configuration secrets found that need to be set."
	}
	
	// Format output
	var output strings.Builder
	output.WriteString("\n" + strings.Repeat("=", 60) + "\n")
	output.WriteString("🔐 SECRET EXTRACTION RESULTS\n")
	output.WriteString(strings.Repeat("=", 60) + "\n")
	
	output.WriteString(fmt.Sprintf("📂 Project Path: %s\n", folderPath))
	output.WriteString(fmt.Sprintf("📊 Project Type: %s\n", projectSecrets.ProjectType))
	output.WriteString(fmt.Sprintf("🔢 Total Variables: %d\n", projectSecrets.TotalVariables))
	output.WriteString(fmt.Sprintf("⚠️  Required Variables: %d\n", projectSecrets.RequiredCount))
	output.WriteString(fmt.Sprintf("📝 Summary: %s\n", projectSecrets.Summary))
	output.WriteString("\n")
	
	// Display Global Secrets
	if len(projectSecrets.GlobalSecrets) > 0 {
		output.WriteString("🌍 GLOBAL SECRETS\n")
		output.WriteString(strings.Repeat("-", 40) + "\n")
		for i, secret := range projectSecrets.GlobalSecrets {
			output.WriteString(fmt.Sprintf("%d. %s\n", i+1, secret.Name))
			output.WriteString(fmt.Sprintf("   Type: %s\n", strings.ToUpper(secret.Type)))
			output.WriteString(fmt.Sprintf("   Source: %s\n", secret.Source))
			output.WriteString(fmt.Sprintf("   Description: %s\n", secret.Description))
			if secret.Example != "" {
				output.WriteString(fmt.Sprintf("   Example: %s=%s\n", secret.Name, secret.Example))
			}
			output.WriteString("\n")
		}
	}
	
	// Display Service-Specific Secrets
	if len(projectSecrets.Services) > 0 {
		output.WriteString("⚙️  SERVICE SECRETS\n")
		output.WriteString(strings.Repeat("-", 40) + "\n")
		for _, service := range projectSecrets.Services {
			output.WriteString(fmt.Sprintf("📦 Service: %s\n", service.ServiceName))
			output.WriteString(fmt.Sprintf("📁 Path: %s\n", service.ServicePath))
			output.WriteString(fmt.Sprintf("📋 Config Files: %s\n", strings.Join(service.ConfigFiles, ", ")))
			output.WriteString("\n")
			
			if len(service.Variables) > 0 {
				for i, secret := range service.Variables {
					output.WriteString(fmt.Sprintf("  %d. %s\n", i+1, secret.Name))
					output.WriteString(fmt.Sprintf("     Type: %s\n", strings.ToUpper(secret.Type)))
					output.WriteString(fmt.Sprintf("     Source: %s\n", secret.Source))
					output.WriteString(fmt.Sprintf("     Description: %s\n", secret.Description))
					if secret.Example != "" {
						output.WriteString(fmt.Sprintf("     Example: %s=%s\n", secret.Name, secret.Example))
					}
					output.WriteString("\n")
				}
			} else {
				output.WriteString("  ✅ No configuration variables needed for this service\n\n")
			}
		}
	}
	
	// Setup Instructions
	if projectSecrets.RequiredCount > 0 {
		output.WriteString("🛠️  SETUP INSTRUCTIONS\n")
		output.WriteString(strings.Repeat("-", 40) + "\n")
		output.WriteString("To configure this project:\n")
		output.WriteString("1. Copy .env.example to .env (if available)\n")
		output.WriteString(fmt.Sprintf("2. Set values for the %d required environment variables shown above\n", projectSecrets.RequiredCount))
		output.WriteString("3. Update any configuration files (config.yaml, application.properties, etc.) with your values\n")
		output.WriteString("4. For API keys and secrets, refer to the respective service documentation\n")
		output.WriteString("5. Ensure all services have access to their required environment variables\n\n")
		output.WriteString("💡 Tip: Check each service's README or documentation for specific setup instructions.\n")
	}
	
	output.WriteString(strings.Repeat("=", 60) + "\n")
	
	return output.String()
}

// UnsupportedCommand handles unknown commands
type UnsupportedCommand struct{}

func (u *UnsupportedCommand) Handle(args []string) string {
	return "unsupported function"
}

// CommandRegistry manages available commands
type CommandRegistry struct {
	commands map[string]CommandHandler
}

func NewCommandRegistry() *CommandRegistry {
	registry := &CommandRegistry{
		commands: make(map[string]CommandHandler),
	}
	
	// Register available commands
	registry.commands["try me"] = &TryMeCommand{}
	registry.commands["/end"] = &EndCommand{}
	registry.commands["secrets"] = &SecretCommand{}
	
	return registry
}

func (cr *CommandRegistry) Execute(command string) (string, bool) {
	if handler, exists := cr.commands[command]; exists {
		return handler.Handle(nil), command == "/end"
	}
	
	// Return unsupported for unknown commands
	unsupported := &UnsupportedCommand{}
	return unsupported.Handle(nil), false
}
