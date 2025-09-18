package cli

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
