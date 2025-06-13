package analyzer

import (
	"testing"

	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"github.com/replicatedhq/troubleshoot/pkg/multitype"
)

func TestAnalyzeImageSignatures_Title(t *testing.T) {
	tests := []struct {
		name     string
		analyzer *troubleshootv1beta2.ImageSignaturesAnalyze
		want     string
	}{
		{
			name: "with_analyzer_name",
			analyzer: &troubleshootv1beta2.ImageSignaturesAnalyze{
				AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{
					CheckName: "image-security-check",
				},
			},
			want: "image-security-check",
		},
		{
			name: "without_analyzer_name",
			analyzer: &troubleshootv1beta2.ImageSignaturesAnalyze{
				AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{},
			},
			want: "Image Signatures",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AnalyzeImageSignatures{
				analyzer: tt.analyzer,
			}
			if got := a.Title(); got != tt.want {
				t.Errorf("AnalyzeImageSignatures.Title() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAnalyzeImageSignatures_IsExcluded(t *testing.T) {
	tests := []struct {
		name     string
		analyzer *troubleshootv1beta2.ImageSignaturesAnalyze
		want     bool
		wantErr  bool
	}{
		{
			name: "exclude_set_to_true",
			analyzer: &troubleshootv1beta2.ImageSignaturesAnalyze{
				AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{
					Exclude: &multitype.BoolOrString{
						Type:    multitype.Bool,
						BoolVal: true,
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "exclude_set_to_false",
			analyzer: &troubleshootv1beta2.ImageSignaturesAnalyze{
				AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{
					Exclude: &multitype.BoolOrString{
						Type:    multitype.Bool,
						BoolVal: false,
					},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "exclude_not_set",
			analyzer: &troubleshootv1beta2.ImageSignaturesAnalyze{
				AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{},
			},
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AnalyzeImageSignatures{
				analyzer: tt.analyzer,
			}
			got, err := a.IsExcluded()
			if (err != nil) != tt.wantErr {
				t.Errorf("AnalyzeImageSignatures.IsExcluded() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("AnalyzeImageSignatures.IsExcluded() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAnalyzeImageSignatures_Analyze(t *testing.T) {
	tests := []struct {
		name         string
		analyzer     *troubleshootv1beta2.ImageSignaturesAnalyze
		collectedData string
		want         int // number of results expected
		wantPass     bool
		wantFail     bool
		wantWarn     bool
		wantMessage  string
	}{
		{
			name: "basic_analysis_with_signatures",
			analyzer: &troubleshootv1beta2.ImageSignaturesAnalyze{
				AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{
					CheckName: "signature-check",
				},
				Outcomes: []*troubleshootv1beta2.Outcome{
					{
						Pass: &troubleshootv1beta2.SingleOutcome{
							When:    "signatures-found",
							Message: "Image signatures found and verified",
						},
					},
					{
						Fail: &troubleshootv1beta2.SingleOutcome{
							When:    "no-signatures",
							Message: "No signatures found for images",
						},
					},
				},
			},
			collectedData: `{
				"images": [
					{
						"image": "nginx:latest",
						"signatures": [
							{
								"verified": true,
								"signature": "signature-data-here"
							}
						]
					}
				]
			}`,
			want:        1,
			wantPass:    true,
			wantMessage: "Image signatures found and verified",
		},
		{
			name: "basic_analysis_no_signatures",
			analyzer: &troubleshootv1beta2.ImageSignaturesAnalyze{
				AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{
					CheckName: "signature-check",
				},
				Outcomes: []*troubleshootv1beta2.Outcome{
					{
						Pass: &troubleshootv1beta2.SingleOutcome{
							When:    "signatures-found",
							Message: "Image signatures found and verified",
						},
					},
					{
						Fail: &troubleshootv1beta2.SingleOutcome{
							When:    "no-signatures",
							Message: "No signatures found for images",
						},
					},
				},
			},
			collectedData: `{
				"images": [
					{
						"image": "nginx:latest",
						"signatures": [
							{
								"verified": false,
								"signature": "",
								"error": "no signatures found for this image"
							}
						]
					}
				]
			}`,
			want:        1,
			wantFail:    true,
			wantMessage: "No signatures found for images",
		},
		{
			name: "missing_collected_data",
			analyzer: &troubleshootv1beta2.ImageSignaturesAnalyze{
				AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{
					CheckName: "signature-check",
				},
				Outcomes: []*troubleshootv1beta2.Outcome{
					{
						Fail: &troubleshootv1beta2.SingleOutcome{
							When:    "error",
							Message: "Failed to analyze image signatures",
						},
					},
				},
			},
			collectedData: "",
			want:          1,
			wantFail:      true,
			wantMessage:   "Failed to analyze image signatures",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AnalyzeImageSignatures{
				analyzer: tt.analyzer,
			}

			getFile := func(filename string) ([]byte, error) {
				if filename == "image-signatures/signature-check.json" || filename == "image-signatures/signatures.json" {
					return []byte(tt.collectedData), nil
				}
				return nil, nil
			}

			findFiles := func(pattern string, excludePatterns []string) (map[string][]byte, error) {
				files := make(map[string][]byte)
				if pattern == "image-signatures/*.json" && tt.collectedData != "" {
					files["image-signatures/signatures.json"] = []byte(tt.collectedData)
				}
				return files, nil
			}

			results, err := a.Analyze(getFile, findFiles)
			if err != nil {
				t.Errorf("AnalyzeImageSignatures.Analyze() error = %v", err)
				return
			}

			if len(results) != tt.want {
				t.Errorf("AnalyzeImageSignatures.Analyze() got %d results, want %d", len(results), tt.want)
				return
			}

			if len(results) > 0 {
				result := results[0]
				if tt.wantPass && !result.IsPass {
					t.Errorf("Expected pass result, got IsPass=%v", result.IsPass)
				}
				if tt.wantFail && !result.IsFail {
					t.Errorf("Expected fail result, got IsFail=%v", result.IsFail)
				}
				if tt.wantWarn && !result.IsWarn {
					t.Errorf("Expected warn result, got IsWarn=%v", result.IsWarn)
				}
				if tt.wantMessage != "" && result.Message != tt.wantMessage {
					t.Errorf("Expected message %q, got %q", tt.wantMessage, result.Message)
				}
			}
		})
	}
}

