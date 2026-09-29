package document

import "strings"

// Chunk is an ordered section of a knowledge document.
type Chunk struct {
	Index   int
	Content string
}

// SplitMarkdown divides Markdown on paragraph boundaries when possible while
// retaining a fixed overlap between adjacent chunks.
func SplitMarkdown(text string, target, overlap int) []Chunk {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n"))
	if text == "" || target <= 0 {
		return nil
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= target {
		overlap = target - 1
	}

	runes := []rune(text)
	chunks := make([]Chunk, 0, (len(runes)+target-1)/target)
	for start := 0; start < len(runes); {
		end := min(start+target, len(runes))
		if end < len(runes) {
			if boundary := paragraphBoundary(runes, start, end, overlap); boundary > start {
				end = boundary
			}
		}
		chunks = append(chunks, Chunk{Index: len(chunks), Content: string(runes[start:end])})
		if end == len(runes) {
			break
		}
		start = end - overlap
	}
	return chunks
}

func paragraphBoundary(runes []rune, start, end, overlap int) int {
	minimum := start + overlap + 1
	for index := end - 1; index > minimum; index-- {
		if runes[index-1] == '\n' && runes[index] == '\n' {
			return index + 1
		}
	}
	return end
}
