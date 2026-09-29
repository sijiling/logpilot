// Package eval calculates deterministic retrieval and Agent behavior metrics.
package eval

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Case defines one expected evaluation outcome.
type Case struct {
	ID                 string   `json:"id"`
	ExpectedDocumentID string   `json:"expected_document_id"`
	ExpectedTool       string   `json:"expected_tool"`
	RequiredKeywords   []string `json:"required_keywords"`
}

// Result is the observed output for one case.
type Result struct {
	ID                   string   `json:"id"`
	RetrievedDocumentIDs []string `json:"retrieved_document_ids"`
	SelectedTools        []string `json:"selected_tools"`
	Answer               string   `json:"answer"`
}

// Metrics summarizes retrieval, tool selection, and answer keyword recall.
type Metrics struct {
	HitAt3                float64 `json:"hit_at_3"`
	ToolSelectionAccuracy float64 `json:"tool_selection_accuracy"`
	KeywordRecall         float64 `json:"keyword_recall"`
}

// LoadCases decodes JSONL evaluation cases.
func LoadCases(reader io.Reader) ([]Case, error) {
	var cases []Case
	if err := scanJSONL(reader, func(line []byte) error {
		var value Case
		if err := json.Unmarshal(line, &value); err != nil {
			return err
		}
		cases = append(cases, value)
		return nil
	}); err != nil {
		return nil, fmt.Errorf("load cases: %w", err)
	}
	return cases, nil
}

// LoadResults decodes JSONL evaluation results.
func LoadResults(reader io.Reader) ([]Result, error) {
	var results []Result
	if err := scanJSONL(reader, func(line []byte) error {
		var value Result
		if err := json.Unmarshal(line, &value); err != nil {
			return err
		}
		results = append(results, value)
		return nil
	}); err != nil {
		return nil, fmt.Errorf("load results: %w", err)
	}
	return results, nil
}

// Calculate returns exact aggregate metrics for matching case/result IDs.
func Calculate(cases []Case, results []Result) (Metrics, error) {
	if len(cases) == 0 {
		return Metrics{}, fmt.Errorf("eval: cases are empty")
	}
	byID := make(map[string]Result, len(results))
	for _, result := range results {
		byID[result.ID] = result
	}
	var hit, tools, keywords, required int
	for _, item := range cases {
		result, ok := byID[item.ID]
		if !ok {
			return Metrics{}, fmt.Errorf("eval: result %q is missing", item.ID)
		}
		if contains(result.RetrievedDocumentIDs[:min(3, len(result.RetrievedDocumentIDs))], item.ExpectedDocumentID) {
			hit++
		}
		if contains(result.SelectedTools, item.ExpectedTool) {
			tools++
		}
		answer := strings.ToLower(result.Answer)
		for _, keyword := range item.RequiredKeywords {
			required++
			if strings.Contains(answer, strings.ToLower(keyword)) {
				keywords++
			}
		}
	}
	count := float64(len(cases))
	keywordRecall := 1.0
	if required > 0 {
		keywordRecall = float64(keywords) / float64(required)
	}
	return Metrics{HitAt3: float64(hit) / count, ToolSelectionAccuracy: float64(tools) / count, KeywordRecall: keywordRecall}, nil
}

func scanJSONL(reader io.Reader, consume func([]byte) error) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64<<10), 1<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if err := consume([]byte(line)); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
