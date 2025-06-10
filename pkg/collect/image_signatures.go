package collect

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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

		// TODO: Implement actual signature collection using Cosign
		// For now, we'll return a placeholder structure
		imageData.Signatures = []Signature{
			{
				Verified:  false,
				Signature: "",
				Error:     "signature collection not yet implemented",
			},
		}

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