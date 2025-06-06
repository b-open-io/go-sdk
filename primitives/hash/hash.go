package primitives

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"

	"golang.org/x/crypto/ripemd160" // nolint:staticcheck // required
)

// Sha256 calculates hash(b) and returns the resulting bytes.
func Sha256(b []byte) []byte {
	data := sha256.Sum256(b)
	return data[:]
}

// Sha256d calculates hash(hash(b)) and returns the resulting bytes.
func Sha256d(b []byte) []byte {
	first := Sha256(b)
	return Sha256(first[:])
}

// Sha512 calculates hash(b) and returns the resulting 64 bytes.
func Sha512(b []byte) []byte {
	data := sha512.Sum512(b)
	return data[:]
}

// Sha256HMAC - HMAC with SHA256
func Sha256HMAC(b, key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(b)
	return mac.Sum(nil)
}

// Sha512HMAC - HMAC with SHA512
func Sha512HMAC(b, key []byte) []byte {
	mac := hmac.New(sha512.New, key)
	mac.Write(b)
	return mac.Sum(nil)
}

// Ripemd160 hashes with RIPEMD160
func Ripemd160(b []byte) []byte {
	ripe := ripemd160.New() // nolint:gosec // required
	_, _ = ripe.Write(b[:])
	return ripe.Sum(nil)
}

// Hash160 hashes with SHA256 and then hashes again with RIPEMD160.
func Hash160(b []byte) []byte {
	hash := Sha256(b)
	return Ripemd160(hash[:])
}
