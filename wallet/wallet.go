// Package wallet provides a comprehensive interface for wallet operations in the BSV blockchain.
// It defines the core Interface with 29 methods covering transaction management, certificate
// operations, cryptographic functions, and blockchain queries. The package includes ProtoWallet
// for basic operations, key derivation utilities, and a complete serializer framework for the
// wallet wire protocol. This design maintains compatibility with the TypeScript SDK while
// following Go idioms and best practices.
package wallet

import (
	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	sighash "github.com/bsv-blockchain/go-sdk/transaction/sighash"
)

// SecurityLevel defines the access control level for wallet operations.
// It determines how strictly the wallet enforces user confirmation for operations.
type SecurityLevel int

var (
	SecurityLevelSilent                  SecurityLevel = 0
	SecurityLevelEveryApp                SecurityLevel = 1
	SecurityLevelEveryAppAndCounterparty SecurityLevel = 2
)

// Protocol defines a protocol with its security level and name.
// The security level determines how strictly the wallet enforces user confirmation.
type Protocol struct {
	SecurityLevel SecurityLevel
	Protocol      string
}

// CounterpartyType represents the type of counterparty in a cryptographic operation.
type CounterpartyType int

const (
	CounterpartyUninitialized CounterpartyType = 0
	CounterpartyTypeAnyone    CounterpartyType = 1
	CounterpartyTypeSelf      CounterpartyType = 2
	CounterpartyTypeOther     CounterpartyType = 3
)

// Counterparty represents the other party in a cryptographic operation.
// It can be a specific public key, or one of the special values 'self' or 'anyone'.
type Counterparty struct {
	Type         CounterpartyType
	Counterparty *ec.PublicKey
}

// Wallet provides cryptographic operations for a specific identity.
// It can encrypt/decrypt data, create/verify signatures, and manage keys.
type Wallet struct {
	ProtoWallet
}

// NewWallet creates a new wallet instance using the provided private key.
// The private key serves as the root of trust for all cryptographic operations.
func NewWallet(privateKey *ec.PrivateKey) (*Wallet, error) {
	if privateKey == nil {
		// Anyone wallet
		return &Wallet{ProtoWallet: ProtoWallet{keyDeriver: NewKeyDeriver(nil)}}, nil
	}
	w, err := NewProtoWallet(ProtoWalletArgs{
		Type:       ProtoWalletArgsTypePrivateKey,
		PrivateKey: privateKey,
	})
	if err != nil {
		return nil, err
	}
	return &Wallet{
		ProtoWallet: *w,
	}, nil
}

// EncryptionArgs contains common parameters for cryptographic operations.
// These parameters specify the protocol, key identity, counterparty, and access control settings.
type EncryptionArgs struct {
	ProtocolID       Protocol     `json:"protocolID,omitempty"`
	KeyID            string       `json:"keyID,omitempty"`
	Counterparty     Counterparty `json:"counterparty,omitempty"`
	Privileged       bool         `json:"privileged,omitempty"`
	PrivilegedReason string       `json:"privilegedReason,omitempty"`
	SeekPermission   bool         `json:"seekPermission,omitempty"`
}

// EncryptArgs contains parameters for encrypting data.
// It extends EncryptionArgs with the plaintext data to be encrypted.
type EncryptArgs struct {
	EncryptionArgs
	Plaintext BytesList `json:"plaintext"`
}

// DecryptArgs contains parameters for decrypting data.
// It extends EncryptionArgs with the ciphertext data to be decrypted.
type DecryptArgs struct {
	EncryptionArgs
	Ciphertext BytesList `json:"ciphertext"`
}

// EncryptResult contains the result of an encryption operation.
type EncryptResult struct {
	Ciphertext BytesList `json:"ciphertext"`
}

// DecryptResult contains the result of a decryption operation.
type DecryptResult struct {
	Plaintext BytesList `json:"plaintext"`
}

// GetPublicKeyArgs contains parameters for retrieving a public key.
// It extends EncryptionArgs with flags to specify identity key or derived key behavior.
type GetPublicKeyArgs struct {
	EncryptionArgs
	IdentityKey bool  `json:"identityKey,omitempty"`
	ForSelf     *bool `json:"forSelf,omitempty"`
}

// GetPublicKeyResult contains the result of a public key retrieval operation.
type GetPublicKeyResult struct {
	PublicKey *ec.PublicKey `json:"publicKey"`
}

// CreateSignatureArgs contains parameters for creating a digital signature.
// It can sign either raw data (which will be hashed) or a pre-computed hash.
type CreateSignatureArgs struct {
	EncryptionArgs
	Data               BytesList `json:"data,omitempty"`
	HashToDirectlySign BytesList `json:"hashToDirectlySign,omitempty"`
}

// CreateSignatureResult contains the result of a signature creation operation.
type CreateSignatureResult struct {
	Signature *ec.Signature `json:"-"` // Ignore original field for JSON
}

// SignOutputs defines which transaction outputs should be signed using SIGHASH flags.
// It wraps the sighash.Flag type to provide specific signing behavior.
type SignOutputs sighash.Flag

var (
	SignOutputsAll    SignOutputs = SignOutputs(sighash.All)
	SignOutputsNone   SignOutputs = SignOutputs(sighash.None)
	SignOutputsSingle SignOutputs = SignOutputs(sighash.Single)
)

// VerifySignatureArgs contains parameters for verifying a digital signature.
// It can verify against either raw data (which will be hashed) or a pre-computed hash.
type VerifySignatureArgs struct {
	EncryptionArgs
	Data                 []byte        `json:"data,omitempty"`
	HashToDirectlyVerify []byte        `json:"hashToDirectlyVerify,omitempty"`
	Signature            *ec.Signature `json:"-"` // Ignore original field for JSON
	ForSelf              *bool         `json:"forSelf,omitempty"`
}

// CreateHMACArgs contains parameters for creating an HMAC.
// It extends EncryptionArgs with the data to be authenticated.
type CreateHMACArgs struct {
	EncryptionArgs
	Data BytesList `json:"data"`
}

// CreateHMACResult contains the result of an HMAC creation operation.
type CreateHMACResult struct {
	HMAC [32]byte `json:"hmac"`
}

// VerifyHMACArgs contains parameters for verifying an HMAC.
// It extends EncryptionArgs with the data and HMAC to be verified.
type VerifyHMACArgs struct {
	EncryptionArgs
	Data []byte   `json:"data"`
	HMAC [32]byte `json:"hmac"`
}

// VerifyHMACResult contains the result of an HMAC verification operation.
type VerifyHMACResult struct {
	Valid bool `json:"valid"`
}

// VerifySignatureResult contains the result of a signature verification operation.
type VerifySignatureResult struct {
	Valid bool `json:"valid"`
}

// AnyoneKey returns the special "anyone" private and public key pair.
// This key pair is used when no specific counterparty is specified,
// effectively making operations available to anyone.
func AnyoneKey() (*ec.PrivateKey, *ec.PublicKey) {
	return ec.PrivateKeyFromBytes([]byte{1})
}
