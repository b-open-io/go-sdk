package wallet

import (
	"fmt"
	"regexp"
	"strings"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
)

// KeyDeriver is responsible for deriving various types of keys using a root private key.
// It supports deriving public and private keys, symmetric keys, and revealing key linkages.
type KeyDeriver struct {
	privateKey *ec.PrivateKey
	publicKey  *ec.PublicKey
}

// NewKeyDeriver creates a new KeyDeriver instance with a root private key.
// The root key can be either a specific private key or the special 'anyone' key.
func NewKeyDeriver(privateKey *ec.PrivateKey) *KeyDeriver {
	return &KeyDeriver{
		privateKey: privateKey,
		publicKey:  privateKey.PubKey(),
	}
}

// DeriveSymmetricKey creates a symmetric key based on protocol ID, key ID, and counterparty.
// Note: Symmetric keys should not be derivable by everyone due to security risks.
func (kd *KeyDeriver) DeriveSymmetricKey(protocol WalletProtocol, keyID string, counterparty WalletCounterparty) ([]byte, error) {
	// Prevent deriving symmetric key for self
	if counterparty.Type == CounterpartyTypeSelf {
		return nil, fmt.Errorf("cannot derive symmetric key for self")
	}

	// If counterparty is 'anyone', use a fixed public key
	if counterparty.Type == CounterpartyTypeAnyone {
		_, fixedKey := ec.PrivateKeyFromBytes([]byte{1})
		counterparty = WalletCounterparty{
			Type:         CounterpartyTypeOther,
			Counterparty: fixedKey,
		}
	}

	// Derive both public and private keys
	derivedPublicKey, err := kd.DerivePublicKey(protocol, keyID, counterparty, false)
	if err != nil {
		return nil, fmt.Errorf("failed to derive public key: %w", err)
	}

	derivedPrivateKey, err := kd.DerivePrivateKey(protocol, keyID, counterparty)
	if err != nil {
		return nil, fmt.Errorf("failed to derive private key: %w", err)
	}

	// Create shared secret
	sharedSecret, err := derivedPrivateKey.DeriveSharedSecret(derivedPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create shared secret: %w", err)
	}
	if sharedSecret == nil {
		return nil, fmt.Errorf("failed to derive shared secret")
	}

	// Return the x coordinate of the shared secret point
	return sharedSecret.X.Bytes(), nil
}

// DerivePublicKey creates a public key based on protocol ID, key ID, and counterparty.
// The forSelf parameter determines whether the key is derived for the user's own identity.
func (kd *KeyDeriver) DerivePublicKey(protocol WalletProtocol, keyID string, counterparty WalletCounterparty, forSelf bool) (*ec.PublicKey, error) {
	counterpartyKey := kd.normalizeCounterparty(counterparty)
	invoiceNumber, err := kd.computeInvoiceNumber(protocol, keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to compute invoice number: %w", err)
	}

	if forSelf {
		privKey, err := kd.privateKey.DeriveChild(counterpartyKey, invoiceNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to derive child private key: %w", err)
		}
		return privKey.PubKey(), nil
	}

	pubKey, err := counterpartyKey.DeriveChild(kd.privateKey, invoiceNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to derive child public key: %w", err)
	}
	return pubKey, nil
}

// DerivePrivateKey creates a private key based on protocol ID, key ID, and counterparty.
// The derived key can be used for signing or other cryptographic operations.
func (kd *KeyDeriver) DerivePrivateKey(protocol WalletProtocol, keyID string, counterparty WalletCounterparty) (*ec.PrivateKey, error) {
	counterpartyKey := kd.normalizeCounterparty(counterparty)
	invoiceNumber, err := kd.computeInvoiceNumber(protocol, keyID)
	if err != nil {
		return nil, err
	}

	k, err := kd.privateKey.DeriveChild(counterpartyKey, invoiceNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to derive child key: %w", err)
	}
	return k, nil
}

// normalizeCounterparty converts the counterparty parameter into a standard public key format.
// It handles special cases like 'self' and 'anyone' by converting them to their corresponding public keys.
func (kd *KeyDeriver) normalizeCounterparty(counterparty WalletCounterparty) *ec.PublicKey {
	switch counterparty.Type {
	case CounterpartyTypeSelf:
		return kd.privateKey.PubKey()
	case CounterpartyTypeOther:
		return counterparty.Counterparty
	case CounterpartyTypeAnyone:
		_, pub := ec.PrivateKeyFromBytes([]byte{1})
		return pub
	default:
		return nil
	}
}

var regexOnlyLettersNumbersSpaces = regexp.MustCompile(`^[a-z0-9 ]+$`)

// computeInvoiceNumber generates a unique identifier string based on the protocol and key ID.
// This string is used as part of the key derivation process to ensure unique keys for different contexts.
func (kd *KeyDeriver) computeInvoiceNumber(protocol WalletProtocol, keyID string) (string, error) {
	// Validate protocol security level
	if protocol.SecurityLevel < 0 || protocol.SecurityLevel > 2 {
		return "", fmt.Errorf("protocol security level must be 0, 1, or 2")
	}

	// Validate key ID
	if len(keyID) > 800 {
		return "", fmt.Errorf("key IDs must be 800 characters or less")
	}
	if len(keyID) < 1 {
		return "", fmt.Errorf("key IDs must be 1 character or more")
	}

	// Validate protocol name
	protocolName := strings.ToLower(strings.TrimSpace(protocol.Protocol))
	if len(protocolName) > 400 {
		// Special handling for specific linkage revelation
		if strings.HasPrefix(protocolName, "specific linkage revelation ") {
			if len(protocolName) > 430 {
				return "", fmt.Errorf("specific linkage revelation protocol names must be 430 characters or less")
			}
		} else {
			return "", fmt.Errorf("protocol names must be 400 characters or less")
		}
	}
	if len(protocolName) < 5 {
		return "", fmt.Errorf("protocol names must be 5 characters or more")
	}
	if strings.Contains(protocolName, "  ") {
		return "", fmt.Errorf("protocol names cannot contain multiple consecutive spaces (\"  \")")
	}
	if !regexOnlyLettersNumbersSpaces.MatchString(protocolName) {
		return "", fmt.Errorf("protocol names can only contain letters, numbers and spaces")
	}
	if strings.HasSuffix(protocolName, " protocol") {
		return "", fmt.Errorf("no need to end your protocol name with \" protocol\"")
	}

	return fmt.Sprintf("%d-%s-%s", protocol.SecurityLevel, protocolName, keyID), nil
}
