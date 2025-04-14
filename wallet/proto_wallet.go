package wallet

import (
	"errors"
	"fmt"

	"crypto/hmac"
	"crypto/sha256"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	hash "github.com/bsv-blockchain/go-sdk/primitives/hash"
)

// ProtoWallet is a precursor to a full wallet, capable of performing foundational cryptographic operations.
// It can derive keys, create signatures, facilitate encryption and HMAC operations.
// Unlike a full wallet, it doesn't create transactions, manage outputs, interact with the blockchain,
// or store any data.
type ProtoWallet struct {
	// The underlying key deriver
	keyDeriver *KeyDeriver
}
type ProtoWalletArgsType string

const (
	ProtoWalletArgsTypePrivateKey ProtoWalletArgsType = "privateKey"
	ProtoWalletArgsTypeKeyDeriver ProtoWalletArgsType = "keyDeriver"
	ProtoWalletArgsTypeAnyone     ProtoWalletArgsType = "anyone"
)

type ProtoWalletArgs struct {
	Type       ProtoWalletArgsType
	PrivateKey *ec.PrivateKey
	KeyDeriver *KeyDeriver
}

// NewProtoWallet creates a new ProtoWallet from a private key or KeyDeriver
func NewProtoWallet(rootKeyOrKeyDeriver ProtoWalletArgs) (*ProtoWallet, error) {
	switch rootKeyOrKeyDeriver.Type {
	case ProtoWalletArgsTypeKeyDeriver:
		return &ProtoWallet{
			keyDeriver: NewKeyDeriver(rootKeyOrKeyDeriver.PrivateKey),
		}, nil
	case ProtoWalletArgsTypePrivateKey:
		return &ProtoWallet{
			keyDeriver: NewKeyDeriver(rootKeyOrKeyDeriver.PrivateKey),
		}, nil
	case ProtoWalletArgsTypeAnyone:
		// Create an "anyone" key deriver as default
		kd := NewKeyDeriver(nil)
		return &ProtoWallet{
			keyDeriver: kd,
		}, nil
	}
	return nil, errors.New("invalid rootKeyOrKeyDeriver")
}

// GetPublicKey returns the public key for the wallet
func (p *ProtoWallet) GetPublicKey(args GetPublicKeyArgs, _originator string) (*GetPublicKeyResult, error) {
	if args.IdentityKey {
		if p.keyDeriver == nil {
			return nil, errors.New("keyDeriver is undefined")
		}
		return &GetPublicKeyResult{
			PublicKey: p.keyDeriver.rootKey.PubKey(),
		}, nil
	} else {
		if args.ProtocolID.Protocol == "" || args.KeyID == "" {
			return nil, errors.New("protocolID and keyID are required if identityKey is false")
		}

		if p.keyDeriver == nil {
			return nil, errors.New("keyDeriver is undefined")
		}

		// Handle default counterparty (self)
		counterparty := args.Counterparty
		if counterparty.Type == CounterpartyUninitialized {
			counterparty = Counterparty{
				Type: CounterpartyTypeSelf,
			}
		}

		pubKey, err := p.keyDeriver.DerivePublicKey(
			args.ProtocolID,
			args.KeyID,
			counterparty,
			args.ForSelf,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to derive public key: %v", err)
		}
		return &GetPublicKeyResult{
			PublicKey: pubKey,
		}, nil
	}
}

// Encrypt encrypts data using the provided protocol ID and key ID
func (p *ProtoWallet) Encrypt(
	args EncryptArgs,
	originator string,
) (*EncryptResult, error) {

	if args.Counterparty.Type == CounterpartyUninitialized {
		args.Counterparty = Counterparty{
			Type: CounterpartyTypeSelf,
		}
	}

	if p.keyDeriver == nil {
		return nil, errors.New("keyDeriver is undefined")
	}

	// Create protocol struct from the protocol ID array
	protocol := args.ProtocolID

	// Handle counterparty
	counterpartyObj := args.Counterparty

	// Derive a symmetric key for encryption
	key, err := p.keyDeriver.DeriveSymmetricKey(protocol, args.KeyID, counterpartyObj)
	if err != nil {
		return nil, fmt.Errorf("failed to derive symmetric key: %v", err)
	}

	encrypted, err := key.Encrypt(args.Plaintext)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt: %v", err)
	}

	return &EncryptResult{
		Ciphertext: encrypted,
	}, nil
}

// Decrypt decrypts data using the provided protocol ID and key ID
func (p *ProtoWallet) Decrypt(
	args DecryptArgs,
	originator string,
) (*DecryptResult, error) {

	if p.keyDeriver == nil {
		return nil, errors.New("keyDeriver is undefined")
	}

	// Handle uninitialized counterparty - default to self
	counterpartyObj := args.Counterparty
	if counterpartyObj.Type == CounterpartyUninitialized {
		counterpartyObj = Counterparty{
			Type: CounterpartyTypeSelf,
		}
	}

	// Derive a symmetric key for decryption
	key, err := p.keyDeriver.DeriveSymmetricKey(args.ProtocolID, args.KeyID, args.Counterparty)
	if err != nil {
		return nil, fmt.Errorf("failed to derive symmetric key: %v", err)
	}

	plaintext, err := key.Decrypt(args.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %v", err)
	}

	return &DecryptResult{
		Plaintext: plaintext,
	}, nil
}

// CreateSignature creates a signature for the provided data
func (p *ProtoWallet) CreateSignature(
	args CreateSignatureArgs,
	originator string,
) (*CreateSignatureResult, error) {
	if p.keyDeriver == nil {
		return nil, errors.New("keyDeriver is undefined")
	}

	if len(args.Data) == 0 && len(args.HashToDirectlySign) == 0 {
		return nil, fmt.Errorf("args.data or args.hashToDirectlySign must be valid")
	}

	// Get hash to sign
	var dataHash []byte
	if len(args.HashToDirectlySign) > 0 {
		dataHash = args.HashToDirectlySign
	} else {
		dataHash = hash.Sha256(args.Data)
	}

	// Handle counterparty
	counterpartyObj := args.Counterparty
	if counterpartyObj.Type == CounterpartyUninitialized {
		counterpartyObj = Counterparty{
			Type: CounterpartyTypeAnyone,
		}
	}

	// Derive private key for signing
	privKey, err := p.keyDeriver.DerivePrivateKey(
		args.ProtocolID,
		args.KeyID,
		counterpartyObj,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to derive private key: %v", err)
	}

	// Create signature
	signature, err := privKey.Sign(dataHash)
	if err != nil {
		return nil, fmt.Errorf("failed to create signature: %v", err)
	}

	return &CreateSignatureResult{
		Signature: *signature,
	}, nil
}

// VerifySignature verifies a signature for the provided data
func (p *ProtoWallet) VerifySignature(
	args VerifySignatureArgs,
	originator string,
) (*VerifySignatureResult, error) {
	if p.keyDeriver == nil {
		return nil, errors.New("keyDeriver is undefined")
	}

	if len(args.Data) == 0 && len(args.HashToDirectlyVerify) == 0 {
		return nil, fmt.Errorf("args.data or args.hashToDirectlyVerify must be valid")
	}

	// Get hash to verify
	var dataHash []byte
	if len(args.HashToDirectlyVerify) > 0 {
		dataHash = args.HashToDirectlyVerify
	} else {
		dataHash = hash.Sha256(args.Data)
	}

	// Handle counterparty
	counterparty := args.Counterparty
	if counterparty.Type == CounterpartyUninitialized {
		counterparty = Counterparty{
			Type: CounterpartyTypeSelf,
		}
	}

	// Derive public key for verification
	pubKey, err := p.keyDeriver.DerivePublicKey(
		args.ProtocolID,
		args.KeyID,
		counterparty,
		args.ForSelf,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to derive public key: %v", err)
	}

	// Verify signature
	valid := args.Signature.Verify(dataHash, pubKey)
	if !valid {
		return nil, fmt.Errorf("signature is not valid")
	}

	return &VerifySignatureResult{
		Valid: valid,
	}, nil
}

// CreateHmac generates an HMAC (Hash-based Message Authentication Code) for the provided data
// using a symmetric key derived from the protocol, key ID, and counterparty.
func (p *ProtoWallet) CreateHmac(
	args CreateHmacArgs,
	originator string,
) (*CreateHmacResult, error) {
	if p.keyDeriver == nil {
		return nil, errors.New("keyDeriver is undefined")
	}

	// Handle default counterparty (self for HMAC)
	counterpartyObj := args.Counterparty
	if counterpartyObj.Type == CounterpartyUninitialized {
		counterpartyObj = Counterparty{
			Type: CounterpartyTypeSelf,
		}
	}

	// Derive a symmetric key for HMAC
	key, err := p.keyDeriver.DeriveSymmetricKey(
		args.ProtocolID,
		args.KeyID,
		counterpartyObj,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to derive symmetric key: %v", err)
	}

	// Create HMAC using the derived key
	mac := hmac.New(sha256.New, key.ToBytes())
	mac.Write(args.Data)
	hmacValue := mac.Sum(nil)

	return &CreateHmacResult{Hmac: hmacValue}, nil
}

// VerifyHmac verifies that the provided HMAC matches the expected value for the given data.
// The verification uses the same protocol, key ID, and counterparty that were used to create the HMAC.
func (p *ProtoWallet) VerifyHmac(
	args VerifyHmacArgs,
	originator string,
) (*VerifyHmacResult, error) {
	if p.keyDeriver == nil {
		return nil, errors.New("keyDeriver is undefined")
	}

	// Handle default counterparty (self for HMAC)
	counterpartyObj := args.Counterparty
	if counterpartyObj.Type == CounterpartyUninitialized {
		counterpartyObj = Counterparty{
			Type: CounterpartyTypeSelf,
		}
	}

	// Derive a symmetric key for HMAC verification
	key, err := p.keyDeriver.DeriveSymmetricKey(
		args.ProtocolID,
		args.KeyID,
		counterpartyObj,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to derive symmetric key: %v", err)
	}

	// Create expected HMAC
	mac := hmac.New(sha256.New, key.ToBytes())
	mac.Write(args.Data)
	expectedHmac := mac.Sum(nil)

	// Verify HMAC
	if !hmac.Equal(expectedHmac, args.Hmac) {
		return &VerifyHmacResult{Valid: false}, nil
	}

	return &VerifyHmacResult{Valid: true}, nil
}
