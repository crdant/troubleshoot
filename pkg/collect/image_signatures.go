package collect

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/pkg/errors"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

type ImageSignatureData struct {
	Image      string      `json:"image"`
	Signatures []Signature `json:"signatures,omitempty"`
	Error      string      `json:"error,omitempty"`
}

type Signature struct {
	Verified  bool   `json:"verified"`
	Signature string `json:"signature,omitempty"`
	Error     string `json:"error,omitempty"`
}

type ImageSignaturesInfo struct {
	Images []ImageSignatureData `json:"images"`
}

type CollectImageSignatures struct {
	Collector    *troubleshootv1beta2.ImageSignatures
	BundlePath   string
	Namespace    string
	ClientConfig *rest.Config
	Client       kubernetes.Interface
	Context      context.Context
	RBACErrors
}

func (c *CollectImageSignatures) Title() string {
	return getCollectorName(c)
}

func (c *CollectImageSignatures) IsExcluded() (bool, error) {
	return isExcluded(c.Collector.Exclude)
}

func (c *CollectImageSignatures) Collect(progressChan chan<- interface{}) (CollectorResult, error) {
	signatureInfo := ImageSignaturesInfo{
		Images: []ImageSignatureData{},
	}

	for _, image := range c.Collector.Images {
		imageData := ImageSignatureData{
			Image: image,
		}

		// Set up authentication for the image
		imageRef, err := alltransports.ParseImageName(fmt.Sprintf("docker://%s", image))
		if err != nil {
			klog.Errorf("failed to parse image name %s: %v", image, err)
			imageData.Error = fmt.Sprintf("failed to parse image name: %v", err)
			signatureInfo.Images = append(signatureInfo.Images, imageData)
			continue
		}

		authConfig, err := getImageAuthConfigForSignatures(c.Namespace, c.ClientConfig, c.Collector, imageRef)
		if err != nil {
			klog.Errorf("failed to get auth config for %s: %v", image, err)
			imageData.Error = fmt.Sprintf("failed to get auth config: %v", err)
			signatureInfo.Images = append(signatureInfo.Images, imageData)
			continue
		}

		// Create system context with authentication
		sysCtx := createSystemContextForSignatures(authConfig)

		// Log authentication status for debugging
		if authConfig != nil {
			klog.V(4).Infof("Using authentication for image %s", image)
		} else {
			klog.V(4).Infof("No authentication configured for image %s", image)
		}

		// TODO: Implement actual signature collection using Cosign
		// For now, we'll return a placeholder structure that indicates auth is working
		authStatus := "no authentication"
		if authConfig != nil {
			authStatus = "authenticated"
		}
		imageData.Signatures = []Signature{
			{
				Verified:  false,
				Signature: "",
				Error:     fmt.Sprintf("signature collection not yet implemented (auth status: %s)", authStatus),
			},
		}

		// Keep reference to sysCtx for future Cosign integration
		_ = sysCtx

		signatureInfo.Images = append(signatureInfo.Images, imageData)
	}

	b, err := json.MarshalIndent(signatureInfo, "", "  ")
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal image signatures info")
	}

	collectorName := c.Collector.CollectorName
	if collectorName == "" {
		collectorName = "signatures"
	}

	output := NewResult()
	output.SaveResult(c.BundlePath, fmt.Sprintf("image-signatures/%s.json", collectorName), bytes.NewBuffer(b))

	return output, nil
}

// getImageAuthConfigForSignatures extracts authentication configuration for image signatures
// This is adapted from the getImageAuthConfig function in registry.go
func getImageAuthConfigForSignatures(namespace string, clientConfig *rest.Config, signaturesCollector *troubleshootv1beta2.ImageSignatures, imageRef types.ImageReference) (*registryAuthConfig, error) {
	if signaturesCollector.ImagePullSecrets == nil {
		return nil, nil
	}

	if signaturesCollector.ImagePullSecrets.Data != nil {
		config, err := getImageAuthConfigFromData(imageRef, signaturesCollector.ImagePullSecrets)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get auth from data")
		}
		return config, nil
	}

	if signaturesCollector.ImagePullSecrets.Name != "" {
		collectorNamespace := signaturesCollector.Namespace
		if collectorNamespace == "" {
			collectorNamespace = namespace
		}
		if collectorNamespace == "" {
			collectorNamespace = "default"
		}
		config, err := getImageAuthConfigFromSecret(clientConfig, imageRef, signaturesCollector.ImagePullSecrets, collectorNamespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get auth from secret")
		}
		return config, nil
	}

	return nil, errors.New("image pull secret spec is not valid")
}

// createSystemContextForSignatures creates a system context with authentication for signature operations
func createSystemContextForSignatures(authConfig *registryAuthConfig) *types.SystemContext {
	sysCtx := &types.SystemContext{
		DockerDisableV1Ping:         true,
		DockerInsecureSkipTLSVerify: types.OptionalBoolTrue,
	}

	if authConfig != nil {
		sysCtx.DockerAuthConfig = &types.DockerAuthConfig{
			Username: authConfig.username,
			Password: authConfig.password,
		}
	}

	return sysCtx
}
