package serializer

import (
	"encoding/base64"
	"encoding/hex"
	"github.com/bsv-blockchain/go-sdk/util"
	"testing"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/stretchr/testify/require"
)

func TestListCertificatesArgs(t *testing.T) {
	tests := []struct {
		name string
		args *wallet.ListCertificatesArgs
	}{{
		name: "full args",
		args: &wallet.ListCertificatesArgs{
			Certifiers: []string{
				"0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
				"02c6047f9441ed7d6d3045406e95c07cd85c778e4b8cef3ca7abac09b95c709ee5",
			},
			Types: []string{
				base64.StdEncoding.EncodeToString(padOrTrim([]byte("type1"), SizeType)),
				base64.StdEncoding.EncodeToString(padOrTrim([]byte("type2"), SizeType)),
			},
			Limit:            10,
			Offset:           5,
			Privileged:       util.BoolPtr(true),
			PrivilegedReason: "test-reason",
		},
	}, {
		name: "minimal args",
		args: &wallet.ListCertificatesArgs{
			Certifiers: []string{"0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798"},
			Types:      []string{base64.StdEncoding.EncodeToString(padOrTrim([]byte("minimal"), SizeType))},
		},
	}, {
		name: "empty certifiers and types",
		args: &wallet.ListCertificatesArgs{
			Certifiers: []string{},
			Types:      []string{},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test serialization
			data, err := SerializeListCertificatesArgs(tt.args)
			require.NoError(t, err)

			// Test deserialization
			got, err := DeserializeListCertificatesArgs(data)
			require.NoError(t, err)

			// Compare results
			require.Equal(t, tt.args, got)
		})
	}
}

func TestListCertificatesResult(t *testing.T) {
	t.Run("full result", func(t *testing.T) {
		pk, err := ec.NewPrivateKey()
		result := &wallet.ListCertificatesResult{
			TotalCertificates: 2,
			Certificates: []wallet.CertificateResult{
				{
					Certificate: wallet.Certificate{
						Type:               base64.StdEncoding.EncodeToString(padOrTrim([]byte("cert1"), SizeType)),
						Subject:            pk.PubKey(),
						SerialNumber:       base64.StdEncoding.EncodeToString(padOrTrim([]byte("serial1"), SizeSerial)),
						Certifier:          pk.PubKey(),
						RevocationOutpoint: "0000000000000000000000000000000000000000000000000000000000000000.0",
						Signature:          hex.EncodeToString(make([]byte, 64)),
						Fields: map[string]string{
							"field1": "value1",
						},
					},
					Keyring: map[string]string{
						"key1": "value1",
					},
					Verifier: "verifier1",
				},
				{
					Certificate: wallet.Certificate{
						Type:               base64.StdEncoding.EncodeToString(padOrTrim([]byte("cert2"), SizeType)),
						Subject:            pk.PubKey(),
						SerialNumber:       base64.StdEncoding.EncodeToString(padOrTrim([]byte("serial2"), SizeSerial)),
						Certifier:          pk.PubKey(),
						RevocationOutpoint: "0000000000000000000000000000000000000000000000000000000000000000.0",
					},
				},
			},
		}

		data, err := SerializeListCertificatesResult(result)
		require.NoError(t, err)

		got, err := DeserializeListCertificatesResult(data)
		require.NoError(t, err)
		require.Equal(t, result, got)
	})

	t.Run("error byte", func(t *testing.T) {
		data := []byte{1} // error byte = 1 (failure)
		_, err := DeserializeListCertificatesResult(data)
		require.Error(t, err)
		require.Contains(t, err.Error(), "listCertificates failed with error byte 1")
	})
}
