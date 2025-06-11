package collect

import (
	"encoding/base64"
	"testing"

	"github.com/containers/image/v5/transports/alltransports"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

func TestGetImageAuthConfig_ImageSignatures(t *testing.T) {
	tests := []struct {
		name            string
		imageSignatures *troubleshootv1beta2.ImageSignatures
		imageName       string
		expectAuth      bool
		expectError     bool
	}{
		{
			name: "no image pull secrets",
			imageSignatures: &troubleshootv1beta2.ImageSignatures{
				Images:    []string{"nginx:latest"},
				Namespace: "default",
			},
			imageName:   "nginx:latest",
			expectAuth:  false,
			expectError: false,
		},
		{
			name: "with image pull secrets data - docker.io auth",
			imageSignatures: &troubleshootv1beta2.ImageSignatures{
				Images:    []string{"nginx:latest"},
				Namespace: "default",
				ImagePullSecrets: &troubleshootv1beta2.ImagePullSecrets{
					Data: map[string]string{
						".dockerconfigjson": base64.StdEncoding.EncodeToString([]byte(`{
							"auths": {
								"docker.io": {
									"auth": "dGVzdDp0ZXN0"
								}
							}
						}`)),
					},
					SecretType: "kubernetes.io/dockerconfigjson",
				},
			},
			imageName:   "nginx:latest",
			expectAuth:  true,
			expectError: false,
		},
		{
			name: "with image pull secrets data - gcr.io auth",
			imageSignatures: &troubleshootv1beta2.ImageSignatures{
				Images:    []string{"gcr.io/test/nginx:latest"},
				Namespace: "default",
				ImagePullSecrets: &troubleshootv1beta2.ImagePullSecrets{
					Data: map[string]string{
						".dockerconfigjson": base64.StdEncoding.EncodeToString([]byte(`{
							"auths": {
								"gcr.io": {
									"username": "_json_key",
									"password": "test-service-account-key"
								}
							}
						}`)),
					},
					SecretType: "kubernetes.io/dockerconfigjson",
				},
			},
			imageName:   "gcr.io/test/nginx:latest",
			expectAuth:  true,
			expectError: false,
		},
		{
			name: "invalid secret type",
			imageSignatures: &troubleshootv1beta2.ImageSignatures{
				Images:    []string{"nginx:latest"},
				Namespace: "default",
				ImagePullSecrets: &troubleshootv1beta2.ImagePullSecrets{
					Data: map[string]string{
						".dockerconfigjson": base64.StdEncoding.EncodeToString([]byte(`{}`)),
					},
					SecretType: "invalid",
				},
			},
			imageName:   "nginx:latest",
			expectAuth:  false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imageRef, err := alltransports.ParseImageName("docker://" + tt.imageName)
			if err != nil {
				t.Fatalf("Failed to parse image name: %v", err)
			}

			authConfig, err := getImageAuthConfigForSignatures("default", &rest.Config{}, tt.imageSignatures, imageRef)

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

			if tt.expectAuth {
				if authConfig == nil {
					t.Error("Expected auth config but got nil")
					return
				}
				if authConfig.username == "" || authConfig.password == "" {
					t.Error("Expected username and password in auth config")
				}
			} else {
				if authConfig != nil {
					t.Error("Expected no auth config but got one")
				}
			}
		})
	}
}

func TestCreateSystemContextForSignatures(t *testing.T) {
	tests := []struct {
		name       string
		authConfig *registryAuthConfig
		expectAuth bool
	}{
		{
			name:       "no auth config",
			authConfig: nil,
			expectAuth: false,
		},
		{
			name: "with auth config",
			authConfig: &registryAuthConfig{
				username: "testuser",
				password: "testpass",
			},
			expectAuth: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sysCtx := createSystemContextForSignatures(tt.authConfig)

			if tt.expectAuth {
				if sysCtx.DockerAuthConfig == nil {
					t.Error("Expected Docker auth config but got nil")
					return
				}
				if sysCtx.DockerAuthConfig.Username != tt.authConfig.username {
					t.Errorf("Expected username %s but got %s", tt.authConfig.username, sysCtx.DockerAuthConfig.Username)
				}
				if sysCtx.DockerAuthConfig.Password != tt.authConfig.password {
					t.Errorf("Expected password %s but got %s", tt.authConfig.password, sysCtx.DockerAuthConfig.Password)
				}
			} else {
				if sysCtx.DockerAuthConfig != nil {
					t.Error("Expected no Docker auth config but got one")
				}
			}

			// Check default system context settings
			if !sysCtx.DockerDisableV1Ping {
				t.Error("Expected DockerDisableV1Ping to be true")
			}
			if sysCtx.DockerInsecureSkipTLSVerify != 1 { // types.OptionalBoolTrue = 1
				t.Error("Expected DockerInsecureSkipTLSVerify to be true")
			}
		})
	}
}

func TestCollectImageSignatures_WithAuthentication(t *testing.T) {
	collector := &CollectImageSignatures{
		Collector: &troubleshootv1beta2.ImageSignatures{
			Images:    []string{"nginx:latest"},
			Namespace: "default",
			ImagePullSecrets: &troubleshootv1beta2.ImagePullSecrets{
				Data: map[string]string{
					".dockerconfigjson": base64.StdEncoding.EncodeToString([]byte(`{
						"auths": {
							"docker.io": {
								"auth": "dGVzdDp0ZXN0"
							}
						}
					}`)),
				},
				SecretType: "kubernetes.io/dockerconfigjson",
			},
		},
		BundlePath:   "/tmp/test",
		Namespace:    "default",
		ClientConfig: &rest.Config{},
		Client:       fake.NewSimpleClientset(),
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
	}
}
