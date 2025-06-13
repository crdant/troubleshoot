package analyzer

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
)

type AnalyzeImageSignatures struct {
	analyzer *troubleshootv1beta2.ImageSignaturesAnalyze
}

func (a *AnalyzeImageSignatures) Title() string {
	if a.analyzer.CheckName != "" {
		return a.analyzer.CheckName
	}
	return "Image Signatures"
}

func (a *AnalyzeImageSignatures) IsExcluded() (bool, error) {
	return isExcluded(a.analyzer.Exclude)
}

func (a *AnalyzeImageSignatures) Analyze(getFile getCollectedFileContents, findFiles getChildCollectedFileContents) ([]*AnalyzeResult, error) {
	result, err := a.analyzeImageSignatures(getFile, findFiles)
	if err != nil {
		return nil, err
	}
	result.Strict = a.analyzer.Strict.BoolOrDefaultFalse()
	return []*AnalyzeResult{result}, nil
}

// ImageSignatureData represents the collected signature data for a single image
type ImageSignatureData struct {
	Image      string      `json:"image"`
	Signatures []Signature `json:"signatures,omitempty"`
	Error      string      `json:"error,omitempty"`
}

// Signature represents a single signature for an image
type Signature struct {
	Verified  bool   `json:"verified"`
	Signature string `json:"signature,omitempty"`
	Error     string `json:"error,omitempty"`
}

// ImageSignaturesInfo represents the complete collected signature data
type ImageSignaturesInfo struct {
	Images []ImageSignatureData `json:"images"`
}

func (a *AnalyzeImageSignatures) analyzeImageSignatures(getFile getCollectedFileContents, findFiles getChildCollectedFileContents) (*AnalyzeResult, error) {
	// First try to get data from a specific collector name if provided
	var collectedData []byte
	var err error
	
	if a.analyzer.CollectorName != "" {
		collectedData, err = getFile(fmt.Sprintf("image-signatures/%s.json", a.analyzer.CollectorName))
	} else {
		// Fallback to looking for any image signatures file
		files, findErr := findFiles("image-signatures/*.json", nil)
		if findErr != nil {
			return nil, errors.Wrap(findErr, "failed to find image signatures files")
		}
		
		// Use the first file found
		for _, data := range files {
			collectedData = data
			break
		}
	}

	if err != nil || len(collectedData) == 0 {
		// No data found, return error result
		result := &AnalyzeResult{
			Title:   a.Title(),
			IconKey: "kubernetes_image_signatures",
		}
		
		for _, outcome := range a.analyzer.Outcomes {
			if outcome.Fail != nil {
				result.IsFail = true
				result.Message = outcome.Fail.Message
				result.URI = outcome.Fail.URI
				return result, nil
			}
		}
		
		// Default error message if no fail outcome is defined
		result.IsFail = true
		result.Message = "No image signature data was collected"
		return result, nil
	}

	// Parse the collected signature data
	signaturesInfo, err := a.processSignatureData(collectedData)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal image signatures result")
	}

	// Count signed, unsigned, and error states
	numSigned := 0
	numUnsigned := 0
	numErrors := 0

	for _, imageData := range signaturesInfo.Images {
		if imageData.Error != "" {
			numErrors++
			continue
		}
		
		hasValidSignature := false
		for _, sig := range imageData.Signatures {
			if sig.Verified && sig.Error == "" {
				hasValidSignature = true
				break
			}
		}
		
		if hasValidSignature {
			numSigned++
		} else {
			numUnsigned++
		}
	}

	// Evaluate outcomes based on the signature analysis
	return a.evaluateSignatureOutcomes(numSigned, numUnsigned, numErrors)
}

func (a *AnalyzeImageSignatures) processSignatureData(data []byte) (*ImageSignaturesInfo, error) {
	if len(data) == 0 {
		return nil, errors.New("empty signature data")
	}
	
	var signaturesInfo ImageSignaturesInfo
	if err := json.Unmarshal(data, &signaturesInfo); err != nil {
		return nil, errors.Wrap(err, "failed to parse signature JSON")
	}
	
	return &signaturesInfo, nil
}

func (a *AnalyzeImageSignatures) evaluateSignatureOutcomes(numSigned, numUnsigned, numErrors int) (*AnalyzeResult, error) {
	result := &AnalyzeResult{
		Title:   a.Title(),
		IconKey: "kubernetes_image_signatures",
	}

	// Check outcomes in order: fail, warn, pass
	for _, outcome := range a.analyzer.Outcomes {
		if outcome.Fail != nil {
			if match, err := a.compareSignatureConditional(outcome.Fail.When, numSigned, numUnsigned, numErrors); err != nil {
				return nil, errors.Wrap(err, "failed to compare signature conditional")
			} else if match {
				result.IsFail = true
				result.Message = outcome.Fail.Message
				result.URI = outcome.Fail.URI
				return result, nil
			}
		}
		
		if outcome.Warn != nil {
			if match, err := a.compareSignatureConditional(outcome.Warn.When, numSigned, numUnsigned, numErrors); err != nil {
				return nil, errors.Wrap(err, "failed to compare signature conditional")
			} else if match {
				result.IsWarn = true
				result.Message = outcome.Warn.Message
				result.URI = outcome.Warn.URI
				return result, nil
			}
		}
		
		if outcome.Pass != nil {
			if match, err := a.compareSignatureConditional(outcome.Pass.When, numSigned, numUnsigned, numErrors); err != nil {
				return nil, errors.Wrap(err, "failed to compare signature conditional")
			} else if match {
				result.IsPass = true
				result.Message = outcome.Pass.Message
				result.URI = outcome.Pass.URI
				return result, nil
			}
		}
	}

	// Default to a passing result if no outcomes matched
	result.IsPass = true
	result.Message = fmt.Sprintf("Analyzed %d images: %d signed, %d unsigned, %d errors", 
		numSigned+numUnsigned+numErrors, numSigned, numUnsigned, numErrors)
	return result, nil
}

func (a *AnalyzeImageSignatures) compareSignatureConditional(conditional string, numSigned, numUnsigned, numErrors int) (bool, error) {
	if conditional == "" {
		return true, nil
	}

	// Parse the conditional: field operator value
	parts := strings.Fields(conditional)
	if len(parts) != 3 {
		return false, errors.Errorf("unable to parse conditional: %s", conditional)
	}

	field := parts[0]
	operator := parts[1]
	valueStr := parts[2]

	// Get the actual value for the field
	var actualValue int
	switch field {
	case "signed":
		actualValue = numSigned
	case "unsigned":
		actualValue = numUnsigned
	case "errors":
		actualValue = numErrors
	default:
		return false, errors.Errorf("unknown field in conditional: %s", field)
	}

	// Parse the expected value
	expectedValue, err := strconv.Atoi(valueStr)
	if err != nil {
		return false, errors.Wrapf(err, "unable to parse expected value: %s", valueStr)
	}

	// Compare based on operator
	switch operator {
	case "=", "==", "===":
		return actualValue == expectedValue, nil
	case "!=", "!==":
		return actualValue != expectedValue, nil
	case "<":
		return actualValue < expectedValue, nil
	case "<=":
		return actualValue <= expectedValue, nil
	case ">":
		return actualValue > expectedValue, nil
	case ">=":
		return actualValue >= expectedValue, nil
	default:
		return false, errors.Errorf("unknown operator in conditional: %s", operator)
	}
}