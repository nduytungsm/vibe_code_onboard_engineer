package chunker

import (
	"bufio"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Chunk represents a piece of file content
type Chunk struct {
	Content   string `json:"content"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Tokens    int    `json:"tokens"`
}

// ChunkFile splits file content into chunks based on token limits
func ChunkFile(content string, maxTokens int, filepath string) ([]Chunk, error) {
	if content == "" {
		return nil, nil
	}

	// Estimate tokens (rough approximation: 1 token ≈ 4 characters)
	estimatedTokens := estimateTokens(content)
	
	// If content is small enough, return as single chunk
	if estimatedTokens <= maxTokens {
		return []Chunk{
			{
				Content:   content,
				StartLine: 1,
				EndLine:   countLines(content),
				Tokens:    estimatedTokens,
			},
		}, nil
	}

	// Split into logical chunks
	return splitContent(content, maxTokens)
}

// splitContent intelligently splits content into chunks
func splitContent(content string, maxTokens int) ([]Chunk, error) {
	lines := strings.Split(content, "\n")
	chunks := make([]Chunk, 0)
	
	currentChunk := strings.Builder{}
	startLine := 1
	currentLine := 1
	
	for _, line := range lines {
		// Estimate tokens for current chunk + new line
		newContent := currentChunk.String() + line + "\n"
		estimatedTokens := estimateTokens(newContent)
		
		if estimatedTokens > maxTokens && currentChunk.Len() > 0 {
			// Current chunk is full, save it
			chunk := Chunk{
				Content:   currentChunk.String(),
				StartLine: startLine,
				EndLine:   currentLine - 1,
				Tokens:    estimateTokens(currentChunk.String()),
			}
			chunks = append(chunks, chunk)
			
			// Start new chunk
			currentChunk.Reset()
			currentChunk.WriteString(line + "\n")
			startLine = currentLine
		} else {
			// Add line to current chunk
			currentChunk.WriteString(line + "\n")
		}
		
		currentLine++
	}
	
	// Add final chunk if it has content
	if currentChunk.Len() > 0 {
		chunk := Chunk{
			Content:   currentChunk.String(),
			StartLine: startLine,
			EndLine:   len(lines),
			Tokens:    estimateTokens(currentChunk.String()),
		}
		chunks = append(chunks, chunk)
	}
	
	return chunks, nil
}

// estimateTokens provides a rough estimate of token count
// More accurate would be to use tiktoken library, but this is simpler
func estimateTokens(text string) int {
	// Rough approximation: 1 token ≈ 4 characters for English text
	// Code tends to have more tokens per character, so we'll be conservative
	charCount := utf8.RuneCountInString(text)
	
	// For code, assume 1 token ≈ 3 characters (more conservative)
	tokenEstimate := charCount / 3
	
	// Add some padding for special tokens
	return tokenEstimate + 10
}

// countLines counts the number of lines in text
func countLines(text string) int {
	if text == "" {
		return 0
	}
	
	scanner := bufio.NewScanner(strings.NewReader(text))
	lines := 0
	for scanner.Scan() {
		lines++
	}
	return lines
}

// SummarizeChunkInfo returns a summary of chunks
func SummarizeChunkInfo(chunks []Chunk) string {
	if len(chunks) == 0 {
		return "No chunks"
	}
	
	if len(chunks) == 1 {
		return fmt.Sprintf("Single chunk: %d tokens", chunks[0].Tokens)
	}
	
	totalTokens := 0
	for _, chunk := range chunks {
		totalTokens += chunk.Tokens
	}
	
	return fmt.Sprintf("%d chunks, %d total tokens", len(chunks), totalTokens)
}
