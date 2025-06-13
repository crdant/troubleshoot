package collect

import (
	"context"
	"testing"

	"github.com/containers/image/v5/transports/alltransports"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
)

func TestFetchImageSignatures(t *testing.T) {
	tests := []struct {
		name        string
		imageName   string
		authConfig  *registryAuthConfig
		expectSigs  bool
		expectError bool
	}{
		{
			name:        "public image without signatures",
			imageName:   "alpine:3.14",
			authConfig:  nil,
			expectSigs:  false,
			expectError: false,
		},
		{
			name:        "invalid image reference",
			imageName:   "invalid::reference",
			authConfig:  nil,
			expectSigs:  false,
			expectError: true,
		},
		{
			name:        "public image with auth config",
			imageName:   "nginx:latest",
			authConfig: &registryAuthConfig{
				username: "testuser",
				password: "testpass",
			},
			expectSigs:  false,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imageRef, err := alltransports.ParseImageName("docker://" + tt.imageName)
			if err != nil {
				if !tt.expectError {
					t.Fatalf("Failed to parse image name: %v", err)
				}
				return
			}

			sysCtx := createSystemContextForSignatures(tt.authConfig)
			
			signatures, err := fetchImageSignatures(context.Background(), imageRef, sysCtx)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.expectSigs {
				if len(signatures) == 0 {
					t.Error("Expected signatures but got none")
				}
			} else {
				// Note: We don't fail if we get signatures when not expecting them,
				// as this could change based on what's actually signed in the registry
				t.Logf("Found %d signatures for %s", len(signatures), tt.imageName)
			}

			// Validate signature structure
			for i, sig := range signatures {
				if sig.Signature == "" && sig.Error == "" {
					t.Errorf("Signature %d has no signature data or error", i)
				}
			}
		})
	}
}

func TestFormatSignatureData(t *testing.T) {
	tests := []struct {
		name       string
		signatures []SignatureInfo
		expected   []Signature
	}{
		{
			name:       "no signatures",
			signatures: []SignatureInfo{},
			expected:   []Signature{},
		},
		{
			name: "single signature",
			signatures: []SignatureInfo{
				{
					Signature: "test-signature-data",
					Verified:  true,
				},
			},
			expected: []Signature{
				{
					Signature: "test-signature-data",
					Verified:  true,
				},
			},
		},
		{
			name: "signature with error",
			signatures: []SignatureInfo{
				{
					Error: "verification failed",
				},
			},
			expected: []Signature{
				{
					Error: "verification failed",
				},
			},
		},
		{
			name: "multiple signatures",
			signatures: []SignatureInfo{
				{
					Signature: "signature-1",
					Verified:  true,
				},
				{
					Signature: "signature-2",
					Verified:  false,
				},
			},
			expected: []Signature{
				{
					Signature: "signature-1",
					Verified:  true,
				},
				{
					Signature: "signature-2",
					Verified:  false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSignatureData(tt.signatures)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d signatures, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i].Signature != expected.Signature {
					t.Errorf("Signature %d: expected signature %q, got %q", i, expected.Signature, result[i].Signature)
				}
				if result[i].Verified != expected.Verified {
					t.Errorf("Signature %d: expected verified %v, got %v", i, expected.Verified, result[i].Verified)
				}
				if result[i].Error != expected.Error {
					t.Errorf("Signature %d: expected error %q, got %q", i, expected.Error, result[i].Error)
				}
			}
		})
	}
}

func TestCollectImageSignatures_WithSignatureRetrieval(t *testing.T) {
	collector := &CollectImageSignatures{
		Collector: &troubleshootv1beta2.ImageSignatures{
			Images:    []string{"alpine:3.14"},
			Namespace: "default",
		},
		BundlePath:   "/tmp/test",
		Namespace:    "default",
		ClientConfig: nil, // Using nil since we're not connecting to a real cluster
		Client:       nil,
		Context:      context.Background(),
	}

	progressChan := make(chan interface{}, 1)
	defer close(progressChan)

	result, err := collector.Collect(progressChan)
	if err != nil {
		t.Errorf("Collect() error = %v", err)
		return
	}
	if result == nil {
		t.Error("Collect() returned nil result")
		return
	}

	// The test should pass regardless of whether signatures are found,
	// as long as the collection process completes without errors
	t.Log("Signature retrieval integration test completed successfully")
}