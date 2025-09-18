package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type REPL struct {
	scanner     *bufio.Scanner
	running     bool
	pathSet     bool
	targetPath  string
}

func NewREPL() *REPL {
	return &REPL{
		scanner: bufio.NewScanner(os.Stdin),
		running: true,
		pathSet: false,
	}
}

func (r *REPL) Start() {
	fmt.Println("🚀 Repo Explanation CLI Started")
	
	// First, prompt for folder path
	if !r.promptForPath() {
		return
	}

	// Then start command loop
	fmt.Println("Type 'try me' to test, '/end' to exit")
	fmt.Print("> ")

	for r.running && r.scanner.Scan() {
		input := strings.TrimSpace(r.scanner.Text())
		r.processCommand(input)
		
		if r.running {
			fmt.Print("> ")
		}
	}

	if err := r.scanner.Err(); err != nil {
		fmt.Printf("Error reading input: %v\n", err)
	}
}

func (r *REPL) promptForPath() bool {
	fmt.Print("Please enter the relative path to a folder: ")
	
	if !r.scanner.Scan() {
		return false
	}
	
	input := strings.TrimSpace(r.scanner.Text())
	if input == "" {
		fmt.Println("Path cannot be empty")
		return false
	}
	
	// Convert relative path to absolute
	absPath, err := filepath.Abs(input)
	if err != nil {
		fmt.Printf("Invalid path: %v\n", err)
		return false
	}
	
	// Check if path exists and is a directory
	info, err := os.Stat(absPath)
	if err != nil {
		fmt.Printf("Path does not exist: %v\n", err)
		return false
	}
	
	if !info.IsDir() {
		fmt.Printf("Path is not a directory: %s\n", absPath)
		return false
	}
	
	r.targetPath = absPath
	r.pathSet = true
	
	// Count folders and report
	folderCount, err := r.countFolders(absPath)
	if err != nil {
		fmt.Printf("Error counting folders: %v\n", err)
		return false
	}
	
	fmt.Printf("Total number of folders in '%s': %d\n", input, folderCount)
	fmt.Println()
	
	return true
}

func (r *REPL) countFolders(rootPath string) (int, error) {
	count := 0
	
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Count directories, but skip the root directory itself
		if info.IsDir() && path != rootPath {
			count++
		}
		
		return nil
	})
	
	return count, err
}

func (r *REPL) processCommand(input string) {
	switch input {
	case "try me":
		fmt.Println("i am here")
	case "/end":
		fmt.Println("Goodbye! 👋")
		r.running = false
	case "":
		// Do nothing for empty input
	default:
		fmt.Println("unsupported function")
	}
}
