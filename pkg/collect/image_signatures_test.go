package collect

import (
	"context"
	"testing"

	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"github.com/replicatedhq/troubleshoot/pkg/multitype"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

func TestCollectImageSignatures_Title(t *testing.T) {
	tests := []struct {
		name      string
		collector *CollectImageSignatures
		expected  string
	}{
		{
			name: "with collector name",
			collector: &CollectImageSignatures{
				Collector: &troubleshootv1beta2.ImageSignatures{
					CollectorMeta: troubleshootv1beta2.CollectorMeta{
						CollectorName: "test-signatures",
					},
				},
			},
			expected: "image-signatures/test-signatures",
		},
		{
			name: "without collector name",
			collector: &CollectImageSignatures{
				Collector: &troubleshootv1beta2.ImageSignatures{},
			},
			expected: "image-signatures",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.collector.Title()
			if result != tt.expected {
				t.Errorf("Title() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCollectImageSignatures_IsExcluded(t *testing.T) {
	tests := []struct {
		name      string
		collector *CollectImageSignatures
		expected  bool
	}{
		{
			name: "exclude set to true",
			collector: &CollectImageSignatures{
				Collector: &troubleshootv1beta2.ImageSignatures{
					CollectorMeta: troubleshootv1beta2.CollectorMeta{
						Exclude: &multitype.BoolOrString{Type: multitype.Bool, BoolVal: true},
					},
				},
			},
			expected: true,
		},
		{
			name: "exclude set to false",
			collector: &CollectImageSignatures{
				Collector: &troubleshootv1beta2.ImageSignatures{
					CollectorMeta: troubleshootv1beta2.CollectorMeta{
						Exclude: &multitype.BoolOrString{Type: multitype.Bool, BoolVal: false},
					},
				},
			},
			expected: false,
		},
		{
			name: "exclude not set",
			collector: &CollectImageSignatures{
				Collector: &troubleshootv1beta2.ImageSignatures{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.collector.IsExcluded()
			if err != nil {
				t.Errorf("IsExcluded() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("IsExcluded() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCollectImageSignatures_Collect(t *testing.T) {
	tests := []struct {
		name      string
		collector *CollectImageSignatures
		wantErr   bool
	}{
		{
			name: "basic collection",
			collector: &CollectImageSignatures{
				Collector: &troubleshootv1beta2.ImageSignatures{
					Images:    []string{"nginx:latest", "alpine:3.14"},
					Namespace: "default",
				},
				BundlePath:   "/tmp/test",
				Namespace:    "default",
				ClientConfig: &rest.Config{},
				Client:       fake.NewSimpleClientset(),
				Context:      context.Background(),
			},
			wantErr: false,
		},
		{
			name: "empty images list",
			collector: &CollectImageSignatures{
				Collector: &troubleshootv1beta2.ImageSignatures{
					Images:    []string{},
					Namespace: "default",
				},
				BundlePath:   "/tmp/test",
				Namespace:    "default",
				ClientConfig: &rest.Config{},
				Client:       fake.NewSimpleClientset(),
				Context:      context.Background(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			progressChan := make(chan interface{}, 1)
			defer close(progressChan)

			result, err := tt.collector.Collect(progressChan)
			if (err != nil) != tt.wantErr {
				t.Errorf("Collect() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("Collect() returned nil result when no error expected")
			}
		})
	}
}

func TestCollectImageSignatures_GetRBACErrors(t *testing.T) {
	collector := &CollectImageSignatures{
		RBACErrors: []error{},
	}

	errors := collector.GetRBACErrors()
	if len(errors) != 0 {
		t.Errorf("GetRBACErrors() = %v, want empty slice", errors)
	}

	if collector.HasRBACErrors() {
		t.Error("HasRBACErrors() = true, want false")
	}
}