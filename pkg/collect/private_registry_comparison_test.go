package collect

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

func TestPrivateRegistryParity(t *testing.T) {
	// Test data for private registry authentication
	dockerConfigJSON := `{
		"auths": {
			"private-registry.io": {
				"username": "testuser",
				"password": "testpass",
				"auth": "dGVzdHVzZXI6dGVzdHBhc3M="
			},
			"gcr.io": {
				"username": "_json_key",
				"password": "{\"type\":\"service_account\"}",
				"auth": ""
			}
		}
	}`

	tests := []struct {
		name          string
		images        []string
		secretName    string
		secretData    map[string][]byte
		namespace     string
		expectSimilar bool // Whether both collectors should behave similarly
	}{
		{
			name:   "private registry with kubernetes secret",
			images: []string{"private-registry.io/user/app:latest"},
			secretName: "private-registry-secret",
			secretData: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(dockerConfigJSON),
			},
			namespace:     "default",
			expectSimilar: true,
		},
		{
			name:   "GCR private registry",
			images: []string{"gcr.io/project/app:latest"},
			secretName: "gcr-secret",
			secretData: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(dockerConfigJSON),
			},
			namespace:     "default",
			expectSimilar: true,
		},
		{
			name:   "mixed public and private images",
			images: []string{"nginx:latest", "private-registry.io/user/app:latest"},
			secretName: "private-registry-secret",
			secretData: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(dockerConfigJSON),
			},
			namespace:     "default",
			expectSimilar: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake Kubernetes client with secret
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tt.secretName,
					Namespace: tt.namespace,
				},
				Type: corev1.SecretTypeDockerConfigJson,
				Data: tt.secretData,
			}
			client := fake.NewSimpleClientset(secret)
			clientConfig := &rest.Config{}

			// Test registry collector
			registryCollector := &CollectRegistry{
				Collector: &troubleshootv1beta2.RegistryImages{
					Images: tt.images,
					ImagePullSecrets: &troubleshootv1beta2.ImagePullSecrets{
						Name: tt.secretName,
					},
					Namespace: tt.namespace,
				},
				BundlePath:   "",
				Namespace:    tt.namespace,
				ClientConfig: clientConfig,
				Client:       client,
				Context:      context.Background(),
			}

			// Test image signatures collector
			signaturesCollector := &CollectImageSignatures{
				Collector: &troubleshootv1beta2.ImageSignatures{
					Images: tt.images,
					ImagePullSecrets: &troubleshootv1beta2.ImagePullSecrets{
						Name: tt.secretName,
					},
					Namespace: tt.namespace,
				},
				BundlePath:   "",
				Namespace:    tt.namespace,
				ClientConfig: clientConfig,
				Client:       client,
				Context:      context.Background(),
			}

			// Collect results from both collectors
			progressChan := make(chan interface{}, 1)
			defer close(progressChan)

			registryResult, registryErr := registryCollector.Collect(progressChan)
			signaturesResult, signaturesErr := signaturesCollector.Collect(progressChan)

			// Both should handle secrets similarly (both succeed or both fail)
			if tt.expectSimilar {
				if (registryErr == nil) != (signaturesErr == nil) {
					t.Errorf("Registry collector error: %v, Signatures collector error: %v - expected similar behavior", registryErr, signaturesErr)
				}

				if registryResult == nil && signaturesResult == nil {
					t.Error("Both collectors returned nil results")
				}
			}

			// Verify that both collectors can process the same authentication scenarios
			if registryResult != nil && signaturesResult != nil {
				t.Logf("Registry collector succeeded, Signatures collector succeeded - good parity")
			}
		})
	}
}

func TestPrivateRegistryAuthConfigParity(t *testing.T) {
	// Test that both collectors use the same authentication configuration logic
	dockerConfigJSON := `{
		"auths": {
			"private-registry.io": {
				"username": "testuser",
				"password": "testpass"
			}
		}
	}`

	imagePullSecrets := &troubleshootv1beta2.ImagePullSecrets{
		Data: map[string]string{
			".dockerconfigjson": base64.StdEncoding.EncodeToString([]byte(dockerConfigJSON)),
		},
		SecretType: "kubernetes.io/dockerconfigjson",
	}

	// Test both auth config extraction functions with the same data
	imageRef, err := parseImageReference("private-registry.io/user/app:latest")
	if err != nil {
		t.Fatalf("Failed to parse image reference: %v", err)
	}

	// Test registry collector auth config
	registryCollector := &troubleshootv1beta2.RegistryImages{
		ImagePullSecrets: imagePullSecrets,
	}
	
	registryAuthConfig, err := getImageAuthConfig("default", &rest.Config{}, registryCollector, imageRef)
	if err != nil {
		t.Errorf("Registry collector auth config failed: %v", err)
	}

	// Test image signatures collector auth config  
	signaturesCollector := &troubleshootv1beta2.ImageSignatures{
		ImagePullSecrets: imagePullSecrets,
	}
	
	signaturesAuthConfig, err := getImageAuthConfigGeneric("default", &rest.Config{}, signaturesCollector, imageRef)
	if err != nil {
		t.Errorf("Signatures collector auth config failed: %v", err)
	}

	// Both should produce equivalent results
	if registryAuthConfig == nil && signaturesAuthConfig == nil {
		t.Log("Both auth configs are nil - consistent")
		return
	}

	if registryAuthConfig == nil || signaturesAuthConfig == nil {
		t.Errorf("Auth config mismatch: registry=%v, signatures=%v", registryAuthConfig, signaturesAuthConfig)
		return
	}

	if registryAuthConfig.username != signaturesAuthConfig.username || 
	   registryAuthConfig.password != signaturesAuthConfig.password {
		t.Errorf("Auth configs differ: registry={%s:%s}, signatures={%s:%s}", 
			registryAuthConfig.username, registryAuthConfig.password,
			signaturesAuthConfig.username, signaturesAuthConfig.password)
	}
}

// Helper function to parse image reference consistently
func parseImageReference(image string) (types.ImageReference, error) {
	return alltransports.ParseImageName(fmt.Sprintf("docker://%s", image))
}