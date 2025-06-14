package serializer

import (
	"fmt"

	"github.com/bsv-blockchain/go-sdk/v2/util"
	"github.com/bsv-blockchain/go-sdk/v2/wallet"
)

func SerializeIdentityCertificate(cert *wallet.IdentityCertificate) ([]byte, error) {
	w := util.NewWriter()

	// Serialize base Certificate fields
	certBytes, err := SerializeCertificate(&cert.Certificate)
	if err != nil {
		return nil, fmt.Errorf("error serializing base certificate: %w", err)
	}
	w.WriteIntBytes(certBytes)

	// Serialize CertifierInfo
	w.WriteString(cert.CertifierInfo.Name)
	w.WriteString(cert.CertifierInfo.IconUrl)
	w.WriteString(cert.CertifierInfo.Description)
	w.WriteByte(cert.CertifierInfo.Trust)

	// Serialize PubliclyRevealedKeyring
	w.WriteVarInt(uint64(len(cert.PubliclyRevealedKeyring)))
	for k, v := range cert.PubliclyRevealedKeyring {
		w.WriteString(k)
		w.WriteString(v)
	}

	// Serialize DecryptedFields
	w.WriteVarInt(uint64(len(cert.DecryptedFields)))
	for k, v := range cert.DecryptedFields {
		w.WriteString(k)
		w.WriteString(v)
	}

	return w.Buf, nil
}

func DeserializeIdentityCertificate(data []byte) (*wallet.IdentityCertificate, error) {
	r := util.NewReaderHoldError(data)
	cert := &wallet.IdentityCertificate{}

	// Deserialize base Certificate
	certBytes := r.ReadIntBytes()
	baseCert, err := DeserializeCertificate(certBytes)
	if err != nil {
		return nil, fmt.Errorf("error deserializing base certificate: %w", err)
	}
	cert.Certificate = *baseCert

	// Deserialize CertifierInfo
	cert.CertifierInfo.Name = r.ReadString()
	cert.CertifierInfo.IconUrl = r.ReadString()
	cert.CertifierInfo.Description = r.ReadString()
	cert.CertifierInfo.Trust = r.ReadByte()

	// Deserialize PubliclyRevealedKeyring
	keyringLen := r.ReadVarInt()
	if keyringLen > 0 {
		cert.PubliclyRevealedKeyring = make(map[string]string, keyringLen)
		for i := uint64(0); i < keyringLen; i++ {
			key := r.ReadString()
			value := r.ReadString()
			cert.PubliclyRevealedKeyring[key] = value
		}
	}

	// Deserialize DecryptedFields
	fieldsLen := r.ReadVarInt()
	if fieldsLen > 0 {
		cert.DecryptedFields = make(map[string]string, fieldsLen)
		for i := uint64(0); i < fieldsLen; i++ {
			key := r.ReadString()
			value := r.ReadString()
			cert.DecryptedFields[key] = value
		}
	}

	r.CheckComplete()
	if r.Err != nil {
		return nil, fmt.Errorf("error deserializing identity certificate: %w", r.Err)
	}

	return cert, nil
}
