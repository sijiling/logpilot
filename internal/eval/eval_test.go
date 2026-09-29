package eval

import "testing"

func TestCalculate(t *testing.T) {
	cases := []Case{
		{ID: "one", ExpectedDocumentID: "doc-1", ExpectedTool: "search_knowledge", RequiredKeywords: []string{"panic", "nil"}},
		{ID: "two", ExpectedDocumentID: "doc-2", ExpectedTool: "analyze_log", RequiredKeywords: []string{"timeout"}},
	}
	results := []Result{
		{ID: "one", RetrievedDocumentIDs: []string{"doc-x", "doc-1"}, SelectedTools: []string{"search_knowledge"}, Answer: "panic caused by nil pointer"},
		{ID: "two", RetrievedDocumentIDs: []string{"doc-x"}, SelectedTools: []string{"search_knowledge"}, Answer: "unknown"},
	}
	metrics, err := Calculate(cases, results)
	if err != nil {
		t.Fatalf("Calculate() error: %v", err)
	}
	if metrics.HitAt3 != 0.5 || metrics.ToolSelectionAccuracy != 0.5 || metrics.KeywordRecall != 2.0/3.0 {
		t.Fatalf("metrics = %#v", metrics)
	}
}
