package serializer

import (
	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetPublicKeyArgs(t *testing.T) {
	testCounterpartyPrivateKey, err := ec.NewPrivateKey()
	require.NoError(t, err)
	tests := []struct {
		name string
		args *wallet.GetPublicKeyArgs
	}{
		{
			name: "full args with identity key",
			args: &wallet.GetPublicKeyArgs{
				IdentityKey: true,
				EncryptionArgs: wallet.EncryptionArgs{
					Privileged:       true,
					PrivilegedReason: "privileged reason",
					SeekPermission:   true,
				},
			},
		},
		{
			name: "full args without identity key",
			args: &wallet.GetPublicKeyArgs{
				ForSelf: true,
				EncryptionArgs: wallet.EncryptionArgs{
					ProtocolID: wallet.Protocol{
						SecurityLevel: wallet.SecurityLevelEveryApp,
						Protocol:      "test-protocol",
					},
					KeyID: "test-key-id",
					Counterparty: wallet.Counterparty{
						Type:         wallet.CounterpartyTypeOther,
						Counterparty: testCounterpartyPrivateKey.PubKey(),
					},
					Privileged:       true,
					PrivilegedReason: "privileged reason",
					SeekPermission:   true,
				},
			},
		},
		{
			name: "minimal args",
			args: &wallet.GetPublicKeyArgs{
				EncryptionArgs: wallet.EncryptionArgs{
					ProtocolID: wallet.Protocol{
						SecurityLevel: wallet.SecurityLevelSilent,
						Protocol:      "default",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test serialization
			data, err := SerializeGetPublicKeyArgs(tt.args)
			require.NoError(t, err)

			// Test deserialization
			got, err := DeserializeGetPublicKeyArgs(data)
			require.NoError(t, err)

			// Compare results
			require.Equal(t, tt.args, got)
		})
	}
}

func TestGetPublicKeyResult(t *testing.T) {
	testPrivKey, err := ec.NewPrivateKey()
	require.NoError(t, err)
	t.Run("serialize/deserialize", func(t *testing.T) {
		result := &wallet.GetPublicKeyResult{
			PublicKey: testPrivKey.PubKey(),
		}
		data, err := SerializeGetPublicKeyResult(result)
		require.NoError(t, err)

		got, err := DeserializeGetPublicKeyResult(data)
		require.NoError(t, err)
		require.Equal(t, result, got)
	})
}
