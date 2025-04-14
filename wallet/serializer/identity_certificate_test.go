package serializer

import (
	"encoding/base64"
	"encoding/hex"
	"github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestIdentityCertificate(t *testing.T) {
	cert := &wallet.IdentityCertificate{
		Certificate: wallet.Certificate{
			Type:               base64.StdEncoding.EncodeToString(padOrTrim([]byte("test-type"), SizeType)),
			Subject:            hex.EncodeToString(make([]byte, SizeSubject)),
			SerialNumber:       base64.StdEncoding.EncodeToString(padOrTrim([]byte("test-serial"), SizeSerial)),
			Certifier:          hex.EncodeToString(make([]byte, SizeCertifier)),
			RevocationOutpoint: "0000000000000000000000000000000000000000000000000000000000000000.0",
			Signature:          hex.EncodeToString(make([]byte, 64)),
			Fields: map[string]string{
				"field1": "value1",
				"field2": "value2",
			},
		},
		CertifierInfo: wallet.IdentityCertifier{
			Name:        "Test Certifier",
			IconUrl:     "https://example.com/icon.png",
			Description: "Test description",
			Trust:       5,
		},
		PubliclyRevealedKeyring: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		DecryptedFields: map[string]string{
			"field1": "decrypted1",
			"field2": "decrypted2",
		},
	}

	// Test serialization
	data, err := SerializeIdentityCertificate(cert)
	require.NoError(t, err)

	// Test deserialization
	got, err := DeserializeIdentityCertificate(data)
	require.NoError(t, err)

	// Compare results
	require.Equal(t, cert, got)
}
