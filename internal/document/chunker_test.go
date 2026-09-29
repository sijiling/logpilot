package document

import (
	"strings"
	"testing"
)

func TestSplitMarkdown(t *testing.T) {
	text := "# Panic\n\nfirst paragraph\n\n## Steps\n\n" + strings.Repeat("x", 30)
	chunks := SplitMarkdown(text, 24, 5)

	if len(chunks) < 3 {
		t.Fatalf("chunks length = %d, want at least 3", len(chunks))
	}
	for index, chunk := range chunks {
		if chunk.Index != index {
			t.Errorf("chunk index = %d, want %d", chunk.Index, index)
		}
		if len([]rune(chunk.Content)) > 24 {
			t.Errorf("chunk %d length = %d, want <= 24", index, len([]rune(chunk.Content)))
		}
	}
	for index := 1; index < len(chunks); index++ {
		previous := []rune(chunks[index-1].Content)
		current := []rune(chunks[index].Content)
		if len(previous) >= 5 && len(current) >= 5 && string(previous[len(previous)-5:]) != string(current[:5]) {
			t.Errorf("chunks %d and %d do not overlap by 5 runes", index-1, index)
		}
	}
}

func TestSplitMarkdownEmpty(t *testing.T) {
	if chunks := SplitMarkdown("  \n", 800, 120); len(chunks) != 0 {
		t.Fatalf("chunks length = %d, want 0", len(chunks))
	}
}
