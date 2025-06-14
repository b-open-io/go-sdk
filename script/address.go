// Package script comment
package script

import (
	"encoding/hex"
	"fmt"

	base58 "github.com/bsv-blockchain/go-sdk/v2/compat/base58"
	ec "github.com/bsv-blockchain/go-sdk/v2/primitives/ec"
	crypto "github.com/bsv-blockchain/go-sdk/v2/primitives/hash"
	"github.com/bsv-blockchain/go-sdk/v2/util"
)

const (
	hashP2PKH        = 0x00
	hashTestNetP2PKH = 0x6f
)

// An Address struct contains the address string as well as the hash160 hex string of the public key.
// The address string will be human-readable and specific to the network type, but the public key hash
// is useful because it stays the same regardless of the network type (mainnet, testnet).
type Address struct {
	AddressString string
	PublicKeyHash util.ByteString
}

// NewAddressFromString takes a string address (P2PKH) and returns a pointer to an Address
// which contains the address string as well as the public key hash string.
func NewAddressFromString(addr string) (*Address, error) {
	pkh, err := addressToPubKeyHash(addr)
	if err != nil {
		return nil, err
	}
	return &Address{
		AddressString: addr,
		PublicKeyHash: pkh,
	}, nil
}

func addressToPubKeyHash(address string) ([]byte, error) {
	decoded, err := base58.Decode(address)
	if err != nil {
		return []byte{}, fmt.Errorf("%w for '%s'", ErrEncodingBadChar, address)
	}

	if len(decoded) != 25 {
		return []byte{}, fmt.Errorf("%w for '%s'", ErrInvalidAddressLength, address)
	}

	switch decoded[0] {
	case hashP2PKH: // Pubkey hash (P2PKH address)
		return decoded[1 : len(decoded)-4], nil

	case hashTestNetP2PKH: // Testnet pubkey hash (P2PKH address)
		return decoded[1 : len(decoded)-4], nil

	default:
		return []byte{}, fmt.Errorf("%w %s", ErrUnsupportedAddress, address)
	}
}

// NewAddressFromPublicKeyString takes a public key string and returns an Address struct pointer.
// If mainnet parameter is true it will return a mainnet address (starting with a 1).
// Otherwise, (mainnet is false) it will return a testnet address (starting with an m or n).
func NewAddressFromPublicKeyString(pubKey string, mainnet bool) (*Address, error) {
	pubKeyBytes, err := hex.DecodeString(pubKey)
	if err != nil {
		return nil, err
	}
	return NewAddressFromPublicKeyHash(crypto.Hash160(pubKeyBytes), mainnet)
}

// NewAddressFromPublicKeyHash takes a public key hash in bytes and returns an Address struct pointer.
// If mainnet parameter is true it will return a mainnet address (starting with a 1).
// Otherwise, (mainnet is false) it will return a testnet address (starting with an m or n).
func NewAddressFromPublicKeyHash(hash []byte, mainnet bool) (*Address, error) {

	// regtest := 111
	// mainnet: 0

	bb := make([]byte, 1)
	if !mainnet {
		bb[0] = 111
	}
	//nolint: makezero // we need to set up the array with 1
	bb = append(bb, hash...)

	return &Address{
		AddressString: Base58EncodeMissingChecksum(bb),
		PublicKeyHash: hash,
	}, nil
}

// NewAddressFromPublicKey takes a bec public key and returns an Address struct pointer.
// If mainnet parameter is true it will return a mainnet address (starting with a 1).
// Otherwise, (mainnet is false) it will return a testnet address (starting with an m or n).
func NewAddressFromPublicKey(pubKey *ec.PublicKey, mainnet bool) (*Address, error) {
	return NewAddressFromPublicKeyWithCompression(pubKey, mainnet, true)
}

func NewAddressFromPublicKeyWithCompression(pubKey *ec.PublicKey, mainnet bool, isCompressed bool) (*Address, error) {
	var hash []byte
	if isCompressed {
		hash = pubKey.Hash()
	} else {
		hash = crypto.Hash160(pubKey.Uncompressed())
	}

	// regtest := 111
	// mainnet: 0

	bb := make([]byte, 1)
	if !mainnet {
		bb[0] = 111
	}
	//nolint: makezero // we need to set up the array with 1
	bb = append(bb, hash...)

	return &Address{
		AddressString: Base58EncodeMissingChecksum(bb),
		PublicKeyHash: hash,
	}, nil
}

// Base58EncodeMissingChecksum appends a checksum to a byte sequence
// then encodes into base58 encoding.
func Base58EncodeMissingChecksum(input []byte) string {
	b := make([]byte, 0, len(input)+4)
	b = append(b, input[:]...)
	ckSum := checksum(b)
	b = append(b, ckSum[:]...)
	return base58.Encode(b)
}

func checksum(input []byte) (ckSum [4]byte) {
	h := crypto.Sha256d(input)
	copy(ckSum[:], h[:4])
	return
}
