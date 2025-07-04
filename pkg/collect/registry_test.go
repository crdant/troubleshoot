package collect

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/containers/image/v5/transports/alltransports"
	"github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/rest"
)

func TestGetImageAuthConfigFromData(t *testing.T) {
	tests := []struct {
		name             string
		imageName        string
		dockerConfigJSON string
		expectedUsername string
		expectedPassword string
		expectedError    bool
	}{
		{
			name:             "docker.io auth",
			imageName:        "docker.io/myimage",
			dockerConfigJSON: `{"auths":{"docker.io":{"auth":"username:password"}}}`,
			expectedUsername: "username",
			expectedPassword: "password",
			expectedError:    false,
		},
		{
			name:             "docker.io auth multi colon",
			imageName:        "docker.io/myimage",
			dockerConfigJSON: `{"auths":{"docker.io":{"auth":"user:name:pass:word"}}}`,
			expectedError:    true,
		},
		{
			name:             "gcr.io auth",
			imageName:        "gcr.io/myimage",
			dockerConfigJSON: `{"auths":{"gcr.io":{"username":"_json_key","password":"sa-key"}}}`,
			expectedUsername: "_json_key",
			expectedPassword: "sa-key",
			expectedError:    false,
		},
		{
			name:             "proxy.replicated.com auth base64 encoded",
			imageName:        "proxy.replicated.com/app-slug/myimage",
			dockerConfigJSON: `{"auths":{"proxy.replicated.com":{"auth":"bGljZW5zZV9pZF8xOmxpY2Vuc2VfaWRfMQ=="}}}`,
			expectedUsername: "license_id_1",
			expectedPassword: "license_id_1",
			expectedError:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			imageRef, err := alltransports.ParseImageName(fmt.Sprintf("docker://%s", test.imageName))
			assert.NoError(t, err)

			pullSecrets := &v1beta2.ImagePullSecrets{
				SecretType: "kubernetes.io/dockerconfigjson",
				Data: map[string]string{
					".dockerconfigjson": base64.StdEncoding.EncodeToString([]byte(test.dockerConfigJSON)),
				},
			}

			authConfig, err := getImageAuthConfigFromData(imageRef, pullSecrets)
			if test.expectedError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, authConfig)
			assert.Equal(t, test.expectedUsername, authConfig.username)
			assert.Equal(t, test.expectedPassword, authConfig.password)
		})
	}
}

func TestGetImageAuthConfig_BackwardCompatibility(t *testing.T) {
	// Test that the wrapper function still works with RegistryImages after refactoring
	dockerConfigJSON := `{
		"auths": {
			"private-registry.io": {
				"username": "testuser",
				"password": "testpass"
			}
		}
	}`

	registryCollector := &v1beta2.RegistryImages{
		Images: []string{"private-registry.io/user/app:latest"},
		ImagePullSecrets: &v1beta2.ImagePullSecrets{
			Data: map[string]string{
				".dockerconfigjson": base64.StdEncoding.EncodeToString([]byte(dockerConfigJSON)),
			},
			SecretType: "kubernetes.io/dockerconfigjson",
		},
		Namespace: "default",
	}

	imageRef, err := alltransports.ParseImageName("docker://private-registry.io/user/app:latest")
	assert.NoError(t, err)

	// This calls the wrapper function which should use the generic function internally
	authConfig, err := getImageAuthConfig("default", &rest.Config{}, registryCollector, imageRef)
	assert.NoError(t, err)
	assert.NotNil(t, authConfig)
	assert.Equal(t, "testuser", authConfig.username)
	assert.Equal(t, "testpass", authConfig.password)
}
