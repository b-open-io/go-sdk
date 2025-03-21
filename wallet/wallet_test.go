package wallet_test

import (
	"crypto/sha256"
	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWallet(t *testing.T) {
	// Create test data
	sampleData := []byte{3, 1, 4, 1, 5, 9}

	// Generate keys
	userKey, err := ec.NewPrivateKey()
	assert.NoError(t, err)
	counterpartyKey, err := ec.NewPrivateKey()
	assert.NoError(t, err)

	// Create wallets with proper initialization
	userWallet := wallet.NewWallet(userKey)
	counterpartyWallet := wallet.NewWallet(counterpartyKey)

	// Define protocol and key ID
	protocol := wallet.WalletProtocol{
		SecurityLevel: wallet.SecurityLevelEveryAppAndCounterparty,
		Protocol:      "tests",
	}

	keyID := "4"

	t.Run("encrypts and decrypts messages", func(t *testing.T) {
		// Encrypt message
		encryptArgs := &wallet.WalletEncryptArgs{
			WalletEncryptionArgs: wallet.WalletEncryptionArgs{
				ProtocolID: protocol,
				KeyID:      keyID,
				Counterparty: wallet.WalletCounterparty{
					Type:         wallet.CounterpartyTypeOther,
					Counterparty: counterpartyKey.PubKey(),
				},
			},
			Plaintext: sampleData,
		}

		encryptResult, err := userWallet.Encrypt(encryptArgs)

		t.Run("successfully encrypts message", func(t *testing.T) {
			assert.NoError(t, err)
			assert.NotEqual(t, sampleData, encryptResult.Ciphertext)
		})

		// Decrypt message
		decryptArgs := &wallet.WalletDecryptArgs{
			WalletEncryptionArgs: wallet.WalletEncryptionArgs{
				ProtocolID: protocol,
				KeyID:      "4",
				Counterparty: wallet.WalletCounterparty{
					Type:         wallet.CounterpartyTypeOther,
					Counterparty: userKey.PubKey(),
				},
			},
			Ciphertext: encryptResult.Ciphertext,
		}

		t.Run("successfully decrypts message", func(t *testing.T) {
			decryptResult, err := counterpartyWallet.Decrypt(decryptArgs)
			assert.NoError(t, err)
			assert.Equal(t, sampleData, decryptResult.Plaintext)
		})

		// Test error cases
		t.Run("wrong protocol", func(t *testing.T) {
			wrongProtocolArgs := *decryptArgs
			wrongProtocolArgs.ProtocolID.Protocol = "wrong"
			_, err := counterpartyWallet.Decrypt(&wrongProtocolArgs)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "cipher: message authentication failed")
		})

		t.Run("wrong key ID", func(t *testing.T) {
			wrongKeyArgs := *decryptArgs
			wrongKeyArgs.KeyID = "5"
			_, err := counterpartyWallet.Decrypt(&wrongKeyArgs)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "cipher: message authentication failed")
		})

		t.Run("wrong counterparty", func(t *testing.T) {
			wrongCounterpartyArgs := *decryptArgs
			wrongCounterpartyArgs.Counterparty.Counterparty = counterpartyKey.PubKey()
			_, err := counterpartyWallet.Decrypt(&wrongCounterpartyArgs)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "cipher: message authentication failed")
		})

		t.Run("invalid protocol name", func(t *testing.T) {
			invalidProtocolArgs := *decryptArgs
			invalidProtocolArgs.ProtocolID.Protocol = "x"
			_, err := counterpartyWallet.Decrypt(&invalidProtocolArgs)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "protocol names must be 5 characters or more")
		})

		t.Run("invalid key ID", func(t *testing.T) {
			invalidKeyArgs := *decryptArgs
			invalidKeyArgs.KeyID = ""
			_, err := counterpartyWallet.Decrypt(&invalidKeyArgs)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "key IDs must be 1 character or more")
		})

		t.Run("invalid security level", func(t *testing.T) {
			invalidSecurityArgs := *decryptArgs
			invalidSecurityArgs.ProtocolID.SecurityLevel = -1
			_, err := counterpartyWallet.Decrypt(&invalidSecurityArgs)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "protocol security level must be 0, 1, or 2")
		})

		t.Run("validates BRC-2 encryption compliance vector", func(t *testing.T) {
			privKey, err := ec.PrivateKeyFromHex("6a2991c9de20e38b31d7ea147bf55f5039e4bbc073160f5e0d541d1f17e321b8")
			assert.NoError(t, err)

			counterparty, err := ec.PublicKeyFromString("0294c479f762f6baa97fbcd4393564c1d7bd8336ebd15928135bbcf575cd1a71a1")
			assert.NoError(t, err)

			result, err := wallet.NewWallet(privKey).Decrypt(&wallet.WalletDecryptArgs{
				WalletEncryptionArgs: wallet.WalletEncryptionArgs{
					ProtocolID: wallet.WalletProtocol{
						SecurityLevel: wallet.SecurityLevelEveryAppAndCounterparty,
						Protocol:      "BRC2 Test",
					},
					KeyID: "42",
					Counterparty: wallet.WalletCounterparty{
						Type:         wallet.CounterpartyTypeOther,
						Counterparty: counterparty,
					},
				},
				Ciphertext: []byte{
					252, 203, 216, 184, 29, 161, 223, 212, 16, 193, 94, 99, 31, 140, 99, 43,
					61, 236, 184, 67, 54, 105, 199, 47, 11, 19, 184, 127, 2, 165, 125, 9,
					188, 195, 196, 39, 120, 130, 213, 95, 186, 89, 64, 28, 1, 80, 20, 213,
					159, 133, 98, 253, 128, 105, 113, 247, 197, 152, 236, 64, 166, 207, 113,
					134, 65, 38, 58, 24, 127, 145, 140, 206, 47, 70, 146, 84, 186, 72, 95,
					35, 154, 112, 178, 55, 72, 124,
				},
			})
			assert.NoError(t, err)
			assert.Equal(t, []byte("BRC-2 Encryption Compliance Validated!"), result.Plaintext)
		})
	})

	t.Run("signs messages verifiable by counterparty", func(t *testing.T) {
		// Create base args
		baseArgs := wallet.WalletEncryptionArgs{
			ProtocolID: protocol,
			KeyID:      keyID,
		}

		// Create signature
		signArgs := &wallet.CreateSignatureArgs{
			WalletEncryptionArgs: baseArgs,
			Data:                 sampleData,
		}
		signArgs.WalletEncryptionArgs.Counterparty = wallet.WalletCounterparty{
			Type:         wallet.CounterpartyTypeOther,
			Counterparty: counterpartyKey.PubKey(),
		}

		signResult, err := userWallet.CreateSignature(signArgs, "")
		assert.NoError(t, err)
		assert.NotEmpty(t, signResult.Signature)

		// Verify signature
		verifyArgs := &wallet.VerifySignatureArgs{
			WalletEncryptionArgs: baseArgs,
			Signature:            signResult.Signature,
			Data:                 sampleData,
		}
		verifyArgs.WalletEncryptionArgs.Counterparty = wallet.WalletCounterparty{
			Type:         wallet.CounterpartyTypeOther,
			Counterparty: userKey.PubKey(),
		}

		verifyResult, err := counterpartyWallet.VerifySignature(verifyArgs, "")
		assert.NoError(t, err)
		assert.True(t, verifyResult.Valid)

		t.Run("directly signs hash of message", func(t *testing.T) {
			// Hash the sample data
			hash := sha256.Sum256(sampleData)

			// Create signature with hash
			signArgs.DashToDirectlySign = hash[:]
			signArgs.Data = nil

			signResult, err := userWallet.CreateSignature(signArgs, "")
			assert.NoError(t, err)
			assert.NotEmpty(t, signResult.Signature)

			// Verify signature with data
			verifyArgsWithData := *verifyArgs
			verifyArgsWithData.Data = sampleData
			verifyArgsWithData.HashToDirectlyVerify = nil

			verifyResult, err := counterpartyWallet.VerifySignature(&verifyArgsWithData, "")
			assert.NoError(t, err)
			assert.True(t, verifyResult.Valid)

			// Verify signature with hash directly
			verifyArgsWithData.Data = nil
			verifyArgsWithData.HashToDirectlyVerify = hash[:]

			verifyHashResult, err := counterpartyWallet.VerifySignature(&verifyArgsWithData, "")
			assert.NoError(t, err)
			assert.True(t, verifyHashResult.Valid)
		})

		t.Run("fails to verify signature with wrong data", func(t *testing.T) {
			// Verify with wrong data
			invalidVerifySignatureArgs := *verifyArgs
			invalidVerifySignatureArgs.Data = append([]byte{0}, sampleData...)
			_, err = counterpartyWallet.VerifySignature(&invalidVerifySignatureArgs, "")
			assert.Error(t, err)
		})

		t.Run("fails to verify signature with wrong protocol", func(t *testing.T) {
			invalidVerifySignatureArgs := *verifyArgs
			invalidVerifySignatureArgs.ProtocolID.Protocol = "wrong"
			_, err = counterpartyWallet.VerifySignature(&invalidVerifySignatureArgs, "")
			assert.Error(t, err)
		})

		t.Run("fails to verify signature with wrong key ID", func(t *testing.T) {
			invalidVerifySignatureArgs := *verifyArgs
			invalidVerifySignatureArgs.KeyID = "wrong"
			_, err = counterpartyWallet.VerifySignature(&invalidVerifySignatureArgs, "")
			assert.Error(t, err)
		})

		t.Run("fails to verify signature with wrong counterparty", func(t *testing.T) {
			invalidVerifySignatureArgs := *verifyArgs
			wrongKey, _ := ec.NewPrivateKey()
			invalidVerifySignatureArgs.Counterparty.Counterparty = wrongKey.PubKey()
			_, err = counterpartyWallet.VerifySignature(&invalidVerifySignatureArgs, "")
			assert.Error(t, err)
		})
		t.Run("validates the BRC-3 compliance vector", func(t *testing.T) {
			anyoneKey, _ := ec.PrivateKeyFromBytes([]byte{1})
			anyoneWallet := wallet.NewWallet(anyoneKey)

			counterparty, err := ec.PublicKeyFromString("0294c479f762f6baa97fbcd4393564c1d7bd8336ebd15928135bbcf575cd1a71a1")
			assert.NoError(t, err)

			signature, err := ec.FromDER([]byte{
				48, 68, 2, 32, 43, 34, 58, 156, 219, 32, 50, 70, 29, 240, 155, 137, 88,
				60, 200, 95, 243, 198, 201, 21, 56, 82, 141, 112, 69, 196, 170, 73, 156,
				6, 44, 48, 2, 32, 118, 125, 254, 201, 44, 87, 177, 170, 93, 11, 193,
				134, 18, 70, 9, 31, 234, 27, 170, 177, 54, 96, 181, 140, 166, 196, 144,
				14, 230, 118, 106, 105,
			})
			assert.NoError(t, err)

			verifyResult, err := anyoneWallet.VerifySignature(&wallet.VerifySignatureArgs{
				WalletEncryptionArgs: wallet.WalletEncryptionArgs{
					ProtocolID: wallet.WalletProtocol{
						SecurityLevel: wallet.SecurityLevelEveryAppAndCounterparty,
						Protocol:      "BRC3 Test",
					},
					KeyID: "42",
					Counterparty: wallet.WalletCounterparty{
						Type:         wallet.CounterpartyTypeOther,
						Counterparty: counterparty,
					},
				},
				Signature: *signature,
				Data:      []byte("BRC-3 Compliance Validated!"),
			}, "")
			assert.NoError(t, err)
			assert.True(t, verifyResult.Valid)
		})
	})
}
