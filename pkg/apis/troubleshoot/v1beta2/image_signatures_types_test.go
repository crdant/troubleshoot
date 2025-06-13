package v1beta2

import (
	"testing"

	"github.com/replicatedhq/troubleshoot/pkg/multitype"
)

func TestImageSignatures_CollectorMeta(t *testing.T) {
	tests := []struct {
		name      string
		collector *ImageSignatures
		expected  string
	}{
		{
			name: "with collector name",
			collector: &ImageSignatures{
				CollectorMeta: CollectorMeta{
					CollectorName: "test-signatures",
				},
			},
			expected: "test-signatures",
		},
		{
			name: "without collector name",
			collector: &ImageSignatures{
				CollectorMeta: CollectorMeta{},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.collector.CollectorName != tt.expected {
				t.Errorf("CollectorName = %v, want %v", tt.collector.CollectorName, tt.expected)
			}
		})
	}
}

func TestImageSignatures_ExcludeFlag(t *testing.T) {
	tests := []struct {
		name      string
		collector *ImageSignatures
		expected  bool
	}{
		{
			name: "exclude set to true",
			collector: &ImageSignatures{
				CollectorMeta: CollectorMeta{
					Exclude: &multitype.BoolOrString{Type: multitype.Bool, BoolVal: true},
				},
			},
			expected: true,
		},
		{
			name: "exclude set to false",
			collector: &ImageSignatures{
				CollectorMeta: CollectorMeta{
					Exclude: &multitype.BoolOrString{Type: multitype.Bool, BoolVal: false},
				},
			},
			expected: false,
		},
		{
			name: "exclude not set",
			collector: &ImageSignatures{
				CollectorMeta: CollectorMeta{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.collector.Exclude != nil {
				excludeVal, err := tt.collector.Exclude.Bool()
				if err != nil {
					t.Errorf("Failed to get bool value: %v", err)
				} else if excludeVal != tt.expected {
					t.Errorf("Exclude = %v, want %v", excludeVal, tt.expected)
				}
			} else if tt.expected != false {
				t.Errorf("Exclude = nil, want %v", tt.expected)
			}
		})
	}
}

func TestImageSignaturesAnalyze_AnalyzeMeta(t *testing.T) {
	tests := []struct {
		name     string
		analyzer *ImageSignaturesAnalyze
		expected string
	}{
		{
			name: "with check name",
			analyzer: &ImageSignaturesAnalyze{
				AnalyzeMeta: AnalyzeMeta{
					CheckName: "image-signatures-check",
				},
			},
			expected: "image-signatures-check",
		},
		{
			name: "without check name",
			analyzer: &ImageSignaturesAnalyze{
				AnalyzeMeta: AnalyzeMeta{},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.analyzer.CheckName != tt.expected {
				t.Errorf("CheckName = %v, want %v", tt.analyzer.CheckName, tt.expected)
			}
		})
	}
}

func TestImageSignaturesAnalyze_RequiredFields(t *testing.T) {
	analyzer := &ImageSignaturesAnalyze{
		CollectorName: "test-collector",
		Outcomes: []*Outcome{
			{
				Pass: &SingleOutcome{
					Message: "All signatures valid",
				},
			},
		},
	}

	if analyzer.CollectorName == "" {
		t.Error("CollectorName should not be empty")
	}

	if len(analyzer.Outcomes) == 0 {
		t.Error("Outcomes should not be empty")
	}
}
