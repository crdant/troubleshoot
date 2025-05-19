package collect

import (
	"testing"

	"github.com/sigstore/cosign/v2/pkg/cosign"
	"github.com/stretchr/testify/require"
)

func TestCosignDependency(t *testing.T) {
	// This test ensures that the Cosign dependency is properly included
	// and that basic imports work correctly
	
	t.Run("can import and use cosign", func(t *testing.T) {
		// Simply creating a cosign options struct verifies that we can use the package
		options := &cosign.CheckOpts{}
		require.NotNil(t, options, "should be able to create cosign options")
	})
}