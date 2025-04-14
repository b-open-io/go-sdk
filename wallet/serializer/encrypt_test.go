package serializer

import (
	"github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestEncryptArgs(t *testing.T) {
	tests := []struct {
		name string
		args *wallet.EncryptArgs
	}{{
		name: "full args",
		args: &wallet.EncryptArgs{
			EncryptionArgs: wallet.EncryptionArgs{
				ProtocolID: wallet.Protocol{
					SecurityLevel: wallet.SecurityLevelEveryApp,
					Protocol:      "test-protocol",
				},
				KeyID:            "test-key",
				Privileged:       true,
				PrivilegedReason: "test reason",
				SeekPermission:   true,
				Counterparty:     wallet.Counterparty{Type: wallet.CounterpartyTypeSelf},
			},
			Plaintext: []byte{1, 2, 3, 4},
		},
	}, {
		name: "minimal args",
		args: &wallet.EncryptArgs{
			EncryptionArgs: wallet.EncryptionArgs{
				ProtocolID: wallet.Protocol{
					SecurityLevel: wallet.SecurityLevelSilent,
					Protocol:      "minimal",
				},
				KeyID: "min-key",
			},
			Plaintext: []byte{5, 6},
		},
	}, {
		name: "no seek permission",
		args: &wallet.EncryptArgs{
			EncryptionArgs: wallet.EncryptionArgs{
				ProtocolID: wallet.Protocol{
					SecurityLevel: wallet.SecurityLevelEveryAppAndCounterparty,
					Protocol:      "no-seek",
				},
				KeyID:          "no-seek-key",
				SeekPermission: false,
			},
			Plaintext: []byte{7, 8, 9},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test serialization
			data, err := SerializeEncryptArgs(tt.args)
			require.NoError(t, err)

			// Test deserialization
			got, err := DeserializeEncryptArgs(data)
			require.NoError(t, err)

			// Compare results
			require.Equal(t, tt.args, got)
		})
	}
}

func TestEncryptResult(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		result := &wallet.EncryptResult{Ciphertext: []byte{1, 2, 3}}
		data, err := SerializeEncryptResult(result)
		require.NoError(t, err)

		got, err := DeserializeEncryptResult(data)
		require.NoError(t, err)
		require.Equal(t, result, got)
	})

	t.Run("error byte", func(t *testing.T) {
		data := []byte{1} // error byte = 1 (failure)
		_, err := DeserializeEncryptResult(data)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed with error byte 1")
	})
}
