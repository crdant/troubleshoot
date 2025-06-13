package collect

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"github.com/replicatedhq/troubleshoot/pkg/multitype"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
				BundlePath:   "",
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
				BundlePath:   "",
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

func TestCollectImageSignatures_ErrorHandling(t *testing.T) {
	tests := []struct {
		name                string
		collector           *CollectImageSignatures
		expectImageErrors   map[string]bool // map[image]hasError
		expectRegistryError bool
		expectAuthError     bool
	}{
		{
			name: "invalid image name formats",
			collector: &CollectImageSignatures{
				Collector: &troubleshootv1beta2.ImageSignatures{
					Images: []string{
						"",
						"registry.io/user/image:tag:invalid",
						"valid-image:latest",
					},
					Namespace: "default",
				},
				BundlePath:   "",
				Namespace:    "default",
				ClientConfig: &rest.Config{},
				Client:       fake.NewSimpleClientset(),
				Context:      context.Background(),
			},
			expectImageErrors: map[string]bool{
				"":                               true,
				"registry.io/user/image:tag:invalid": true,
				"valid-image:latest":                 false,
			},
		},
		{
			name: "missing image pull secret",
			collector: &CollectImageSignatures{
				Collector: &troubleshootv1beta2.ImageSignatures{
					Images: []string{"private-registry.io/user/image:latest"},
					ImagePullSecrets: &troubleshootv1beta2.ImagePullSecrets{
						Name: "nonexistent-secret",
					},
					Namespace: "default",
				},
				BundlePath:   "",
				Namespace:    "default",
				ClientConfig: &rest.Config{},
				Client:       fake.NewSimpleClientset(),
				Context:      context.Background(),
			},
			expectImageErrors: map[string]bool{
				"private-registry.io/user/image:latest": true,
			},
			expectAuthError: true,
		},
		{
			name: "invalid image pull secret data",
			collector: &CollectImageSignatures{
				Collector: &troubleshootv1beta2.ImageSignatures{
					Images: []string{"private-registry.io/user/image:latest"},
					ImagePullSecrets: &troubleshootv1beta2.ImagePullSecrets{
						Data: map[string]string{
							"invalid-key": "invalid-value",
						},
					},
					Namespace: "default",
				},
				BundlePath:   "",
				Namespace:    "default",
				ClientConfig: &rest.Config{},
				Client:       fake.NewSimpleClientset(),
				Context:      context.Background(),
			},
			expectImageErrors: map[string]bool{
				"private-registry.io/user/image:latest": true,
			},
			expectAuthError: true,
		},
		{
			name: "valid secret with authentication",
			collector: &CollectImageSignatures{
				Collector: &troubleshootv1beta2.ImageSignatures{
					Images: []string{"private-registry.io/user/image:latest"},
					ImagePullSecrets: &troubleshootv1beta2.ImagePullSecrets{
						Name: "valid-secret",
					},
					Namespace: "test-namespace",
				},
				BundlePath:   "",
				Namespace:    "test-namespace",
				ClientConfig: &rest.Config{},
				Client: fake.NewSimpleClientset(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "valid-secret",
						Namespace: "test-namespace",
					},
					Type: corev1.SecretTypeDockerConfigJson,
					Data: map[string][]byte{
						corev1.DockerConfigJsonKey: []byte(`{
							"auths": {
								"private-registry.io": {
									"username": "testuser",
									"password": "testpass",
									"auth": "dGVzdHVzZXI6dGVzdHBhc3M="
								}
							}
						}`),
					},
				}),
				Context: context.Background(),
			},
			expectImageErrors: map[string]bool{
				"private-registry.io/user/image:latest": true, // Will fail due to fake client
			},
		},
		{
			name: "air-gapped environment simulation",
			collector: &CollectImageSignatures{
				Collector: &troubleshootv1beta2.ImageSignatures{
					Images: []string{
						"localhost:5000/internal/app:latest",
						"internal-registry.company.com/app:v1.0",
					},
					Namespace: "default",
				},
				BundlePath:   "",
				Namespace:    "default",
				ClientConfig: &rest.Config{},
				Client:       fake.NewSimpleClientset(),
				Context:      context.Background(),
			},
			expectImageErrors: map[string]bool{
				"localhost:5000/internal/app:latest":         false,
				"internal-registry.company.com/app:v1.0": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			progressChan := make(chan interface{}, 1)
			defer close(progressChan)

			result, err := tt.collector.Collect(progressChan)
			if err != nil {
				t.Errorf("Collect() unexpected error = %v", err)
				return
			}

			if result == nil {
				t.Fatal("Collect() returned nil result")
			}

			// Parse the result to check for specific errors
			var signatureInfo ImageSignaturesInfo
			if len(result) == 0 {
				t.Fatal("No files in result")
			}

			// Find the signatures file data
			var signaturesData []byte
			for relativePath, data := range result {
				if strings.Contains(relativePath, "image-signatures/") && data != nil {
					signaturesData = data
					break
				}
			}

			if signaturesData == nil {
				t.Fatal("No signatures file found in result")
			}

			err = json.Unmarshal(signaturesData, &signatureInfo)
			if err != nil {
				t.Fatalf("Failed to unmarshal signature info: %v", err)
			}

			// Verify error expectations
			for _, imageData := range signatureInfo.Images {
				expectError, exists := tt.expectImageErrors[imageData.Image]
				if !exists {
					continue
				}

				hasError := imageData.Error != ""
				if hasError != expectError {
					t.Errorf("Image %s: expected error=%v, got error=%v (error: %s)",
						imageData.Image, expectError, hasError, imageData.Error)
				}

				// Check for specific error types
				if tt.expectAuthError && hasError {
					if !strings.Contains(imageData.Error, "auth") {
						t.Errorf("Expected auth error for image %s, got: %s",
							imageData.Image, imageData.Error)
					}
				}
			}
		})
	}
}

func TestCollectImageSignatures_RegistryTimeoutHandling(t *testing.T) {
	// Test that registry timeouts are handled gracefully
	collector := &CollectImageSignatures{
		Collector: &troubleshootv1beta2.ImageSignatures{
			Images: []string{
				"timeout-registry.example.com/app:latest",
			},
			Namespace: "default",
		},
		BundlePath:   "",
		Namespace:    "default",
		ClientConfig: &rest.Config{},
		Client:       fake.NewSimpleClientset(),
		Context:      context.Background(),
	}

	progressChan := make(chan interface{}, 1)
	defer close(progressChan)

	result, err := collector.Collect(progressChan)
	if err != nil {
		t.Errorf("Collect() should handle registry errors gracefully, got error: %v", err)
		return
	}

	if result == nil {
		t.Fatal("Collect() returned nil result")
	}
}

func TestCollectImageSignatures_PartialFailureRecovery(t *testing.T) {
	// Test that partial failures don't prevent processing other images
	collector := &CollectImageSignatures{
		Collector: &troubleshootv1beta2.ImageSignatures{
			Images: []string{
				"nginx:latest",
				"",
				"alpine:3.14",
				"malformed:image:name:too:many:colons",
			},
			Namespace: "default",
		},
		BundlePath:   "",
		Namespace:    "default",
		ClientConfig: &rest.Config{},
		Client:       fake.NewSimpleClientset(),
		Context:      context.Background(),
	}

	progressChan := make(chan interface{}, 1)
	defer close(progressChan)

	result, err := collector.Collect(progressChan)
	if err != nil {
		t.Errorf("Collect() should continue processing valid images despite errors, got: %v", err)
		return
	}

	if result == nil {
		t.Fatal("Collect() returned nil result")
	}

	// Parse the result
	var signatureInfo ImageSignaturesInfo
	if len(result) == 0 {
		t.Fatal("No files in result")
	}

	// Find the signatures file data
	var signaturesData []byte
	for relativePath, data := range result {
		if strings.Contains(relativePath, "image-signatures/") && data != nil {
			signaturesData = data
			break
		}
	}

	if signaturesData == nil {
		t.Fatal("No signatures file found in result")
	}

	err = json.Unmarshal(signaturesData, &signatureInfo)
	if err != nil {
		t.Fatalf("Failed to unmarshal signature info: %v", err)
	}

	// Verify all images are processed, some with errors
	if len(signatureInfo.Images) != 4 {
		t.Errorf("Expected 4 images processed, got %d", len(signatureInfo.Images))
	}

	validImages := 0
	errorImages := 0
	for _, imageData := range signatureInfo.Images {
		if imageData.Error != "" {
			errorImages++
		} else {
			validImages++
		}
	}

	// We expect at least some valid images and some error images
	if validImages == 0 {
		t.Error("Expected at least some valid images to be processed")
	}
	if errorImages == 0 {
		t.Error("Expected at least some images to have errors")
	}
}