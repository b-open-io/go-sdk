package serializer

import (
	"github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetVersionResult(t *testing.T) {
	tests := []struct {
		name   string
		result *wallet.GetVersionResult
	}{
		{
			name: "standard version",
			result: &wallet.GetVersionResult{
				Version: "1.2.3",
			},
		},
		{
			name: "long version",
			result: &wallet.GetVersionResult{
				Version: "v2.5.1-beta+12345",
			},
		},
		{
			name: "empty version",
			result: &wallet.GetVersionResult{
				Version: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test serialization
			data, err := SerializeGetVersionResult(tt.result)
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(data), 1) // At least error byte

			// Test deserialization
			got, err := DeserializeGetVersionResult(data)
			require.NoError(t, err)
			require.Equal(t, tt.result, got)
		})
	}
}
