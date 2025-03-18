package wallet

import (
	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeyDeriver(t *testing.T) {
	rootPrivateKey, _ := ec.NewPrivateKey()
	rootPublicKey := rootPrivateKey.PubKey()
	counterpartyPrivateKey, _ := ec.NewPrivateKey()
	counterpartyPublicKey := counterpartyPrivateKey.PubKey()
	anyonePrivateKey, _ := ec.PrivateKeyFromBytes([]byte{1})
	anyonePublicKey := anyonePrivateKey.PubKey()

	protocolID := WalletProtocol{
		SecurityLevel: SecurityLevelEveryAppAndCounterparty,
		Protocol:      "testprotocol",
	}
	keyID := "12345"

	var keyDeriver *KeyDeriver

	t.Run("should compute the correct invoice number", func(t *testing.T) {
		keyDeriver = NewKeyDeriver(rootPrivateKey)
		invoiceNumber, err := keyDeriver.computeInvoiceNumber(protocolID, keyID)
		assert.NoError(t, err)
		assert.Equal(t, "2-testprotocol-12345", invoiceNumber)
	})

	t.Run("should normalize counterparty correctly for self", func(t *testing.T) {
		normalized := keyDeriver.normalizeCounterparty(WalletCounterparty{
			Type: CounterpartyTypeSelf,
		})
		assert.Equal(t, rootPublicKey.ToDERHex(), normalized.ToDERHex())
	})

	t.Run("should normalize counterparty correctly for anyone", func(t *testing.T) {
		normalized := keyDeriver.normalizeCounterparty(WalletCounterparty{
			Type: CounterpartyTypeAnyone,
		})
		assert.Equal(t, anyonePublicKey.ToDERHex(), normalized.ToDERHex())
	})

	t.Run("should normalize counterparty correctly when given as a public key", func(t *testing.T) {
		normalized := keyDeriver.normalizeCounterparty(WalletCounterparty{
			Type:         CounterpartyTypeOther,
			Counterparty: counterpartyPublicKey,
		})
		assert.Equal(t, counterpartyPublicKey.ToDERHex(), normalized.ToDERHex())
	})

	t.Run("should allow public key derivation as anyone", func(t *testing.T) {
		anyoneDeriver := NewKeyDeriver(anyonePrivateKey)
		derivedPublicKey, err := anyoneDeriver.DerivePublicKey(
			protocolID,
			keyID,
			WalletCounterparty{
				Type:         CounterpartyTypeOther,
				Counterparty: counterpartyPublicKey,
			},
			false,
		)
		assert.NoError(t, err)
		assert.IsType(t, &ec.PublicKey{}, derivedPublicKey)
	})

	t.Run("should derive the correct public key for counterparty", func(t *testing.T) {
		derivedPublicKey, err := keyDeriver.DerivePublicKey(
			protocolID,
			keyID,
			WalletCounterparty{
				Type:         CounterpartyTypeOther,
				Counterparty: counterpartyPublicKey,
			},
			false,
		)
		assert.NoError(t, err)
		assert.IsType(t, &ec.PublicKey{}, derivedPublicKey)
	})

	t.Run("should derive the correct public key for self", func(t *testing.T) {
		derivedPublicKey, err := keyDeriver.DerivePublicKey(
			protocolID,
			keyID,
			WalletCounterparty{
				Type:         CounterpartyTypeOther,
				Counterparty: counterpartyPublicKey,
			},
			true,
		)
		assert.NoError(t, err)
		assert.IsType(t, &ec.PublicKey{}, derivedPublicKey)
	})

	t.Run("should derive the correct private key", func(t *testing.T) {
		derivedPrivateKey, err := keyDeriver.DerivePrivateKey(
			protocolID,
			keyID,
			WalletCounterparty{
				Type:         CounterpartyTypeOther,
				Counterparty: counterpartyPublicKey,
			},
		)
		assert.NoError(t, err)
		assert.IsType(t, &ec.PrivateKey{}, derivedPrivateKey)
	})

	t.Run("should derive the correct symmetric key", func(t *testing.T) {
		derivedSymmetricKey, err := keyDeriver.DeriveSymmetricKey(
			protocolID,
			keyID,
			WalletCounterparty{
				Type:         CounterpartyTypeOther,
				Counterparty: counterpartyPublicKey,
			},
		)
		assert.NoError(t, err)
		assert.NotEmpty(t, derivedSymmetricKey)
	})

	t.Run("should be able to derive symmetric key with anyone", func(t *testing.T) {
		_, err := keyDeriver.DeriveSymmetricKey(
			protocolID,
			keyID,
			WalletCounterparty{
				Type: CounterpartyTypeAnyone,
			},
		)
		assert.NoError(t, err)
	})

	t.Run("should reveal the correct counterparty shared secret", func(t *testing.T) {
		sharedSecret, err := keyDeriver.DeriveSymmetricKey(
			protocolID,
			keyID,
			WalletCounterparty{
				Type:         CounterpartyTypeOther,
				Counterparty: counterpartyPublicKey,
			},
		)
		assert.NoError(t, err)
		assert.NotEmpty(t, sharedSecret)
	})

	t.Run("should not reveal shared secret for self", func(t *testing.T) {
		_, err := keyDeriver.DeriveSymmetricKey(
			protocolID,
			keyID,
			WalletCounterparty{
				Type: CounterpartyTypeSelf,
			},
		)
		assert.Error(t, err)
		assert.EqualError(t, err, "cannot derive symmetric key for self")
	})

	t.Run("should reveal the specific key association", func(t *testing.T) {
		sharedSecret, err := keyDeriver.DeriveSymmetricKey(
			protocolID,
			keyID,
			WalletCounterparty{
				Type:         CounterpartyTypeOther,
				Counterparty: counterpartyPublicKey,
			},
		)
		assert.NoError(t, err)
		assert.NotEmpty(t, sharedSecret)
	})

	t.Run("should throw an error for invalid protocol names", func(t *testing.T) {
		testCases := []struct {
			name     string
			protocol WalletProtocol
			keyID    string
		}{
			{
				name: "long key ID",
				protocol: WalletProtocol{
					SecurityLevel: 2,
					Protocol:      "test",
				},
				keyID: "long" + string(make([]byte, 800)),
			},
			{
				name: "empty key ID",
				protocol: WalletProtocol{
					SecurityLevel: 2,
					Protocol:      "test",
				},
				keyID: "",
			},
			{
				name: "invalid security level",
				protocol: WalletProtocol{
					SecurityLevel: -3,
					Protocol:      "otherwise valid",
				},
				keyID: keyID,
			},
			{
				name: "double space in protocol name",
				protocol: WalletProtocol{
					SecurityLevel: 2,
					Protocol:      "double  space",
				},
				keyID: keyID,
			},
			{
				name: "empty protocol name",
				protocol: WalletProtocol{
					SecurityLevel: 0,
					Protocol:      "",
				},
				keyID: keyID,
			},
			{
				name: "long protocol name",
				protocol: WalletProtocol{
					SecurityLevel: 0,
					Protocol:      "long" + string(make([]byte, 400)),
				},
				keyID: keyID,
			},
			{
				name: "redundant protocol suffix",
				protocol: WalletProtocol{
					SecurityLevel: 2,
					Protocol:      "redundant protocol protocol",
				},
				keyID: keyID,
			},
			{
				name: "invalid characters in protocol name",
				protocol: WalletProtocol{
					SecurityLevel: 2,
					Protocol:      "üñî√é®sål ©0på",
				},
				keyID: keyID,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := keyDeriver.computeInvoiceNumber(tc.protocol, tc.keyID)
				assert.Error(t, err)
			})
		}
	})
}
