package serializer

import (
	"fmt"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/bsv-blockchain/go-sdk/util"
	"github.com/bsv-blockchain/go-sdk/wallet"
)

func SerializeListCertificatesArgs(args *wallet.ListCertificatesArgs) ([]byte, error) {
	w := util.NewWriter()

	// Write certifiers
	w.WriteVarInt(uint64(len(args.Certifiers)))
	for _, certifier := range args.Certifiers {
		w.WriteBytes(certifier.Compressed())
	}

	// Write types
	w.WriteVarInt(uint64(len(args.Types)))
	for _, typ := range args.Types {
		w.WriteBytes(typ[:])
	}

	// Write limit and offset
	w.WriteOptionalUint32(args.Limit)
	w.WriteOptionalUint32(args.Offset)

	// Write privileged params
	w.WriteBytes(encodePrivilegedParams(args.Privileged, args.PrivilegedReason))

	return w.Buf, nil
}

func DeserializeListCertificatesArgs(data []byte) (*wallet.ListCertificatesArgs, error) {
	r := util.NewReaderHoldError(data)
	args := &wallet.ListCertificatesArgs{}

	// Read certifiers
	certifiersLength := r.ReadVarInt()
	args.Certifiers = make([]*ec.PublicKey, 0, certifiersLength)
	for i := uint64(0); i < certifiersLength; i++ {
		certifierBytes := r.ReadBytes(33)
		if r.Err != nil {
			return nil, fmt.Errorf("error deserializing certifier: %w", r.Err)
		}
		parsedCertifier, err := ec.PublicKeyFromBytes(certifierBytes)
		if err != nil {
			return nil, fmt.Errorf("error parsing certifier public key: %w", err)
		}
		args.Certifiers = append(args.Certifiers, parsedCertifier)
	}

	// Read types
	typesLength := r.ReadVarInt()
	args.Types = make([]wallet.CertificateType, 0, typesLength)
	for i := uint64(0); i < typesLength; i++ {
		var typeArray wallet.CertificateType
		copy(typeArray[:], r.ReadBytes(32))
		if r.Err != nil {
			return nil, fmt.Errorf("error deserializing type: %w", r.Err)
		}
		args.Types = append(args.Types, typeArray)
	}

	// Read limit and offset
	args.Limit = r.ReadOptionalUint32()
	args.Offset = r.ReadOptionalUint32()

	// Read privileged params
	args.Privileged, args.PrivilegedReason = decodePrivilegedParams(r)

	r.CheckComplete()
	if r.Err != nil {
		return nil, fmt.Errorf("error deserializing ListCertificates args: %w", r.Err)
	}

	return args, nil
}

func SerializeListCertificatesResult(result *wallet.ListCertificatesResult) ([]byte, error) {
	w := util.NewWriter()

	if result.TotalCertificates != uint32(len(result.Certificates)) {
		return nil, fmt.Errorf("total certificates %d does not match length of certificates %d", result.TotalCertificates, len(result.Certificates))
	}

	// Write total certificates
	w.WriteVarInt(uint64(result.TotalCertificates))

	// Write certificates
	for _, cert := range result.Certificates {
		certBytes, err := SerializeCertificate(&cert.Certificate)
		if err != nil {
			return nil, fmt.Errorf("error serializing certificate: %w", err)
		}
		w.WriteIntBytes(certBytes)

		// Write keyring if present
		if cert.Keyring != nil {
			w.WriteByte(1) // present
			w.WriteVarInt(uint64(len(cert.Keyring)))
			for k, v := range cert.Keyring {
				w.WriteString(k)
				if err := w.WriteIntFromBase64(v); err != nil {
					return nil, fmt.Errorf("invalid keyring value base64: %w", err)
				}
			}
		} else {
			w.WriteByte(0) // not present
		}

		w.WriteIntBytes(cert.Verifier)
	}

	return w.Buf, nil
}

func DeserializeListCertificatesResult(data []byte) (*wallet.ListCertificatesResult, error) {
	r := util.NewReaderHoldError(data)
	result := &wallet.ListCertificatesResult{}

	// Read total certificates
	result.TotalCertificates = uint32(r.ReadVarInt())

	// Read certificates
	if result.TotalCertificates > 0 {
		result.Certificates = make([]wallet.CertificateResult, 0, result.TotalCertificates)
	}
	for i := uint32(0); i < result.TotalCertificates; i++ {
		cert, err := DeserializeCertificate(r.ReadIntBytes())
		if err != nil {
			return nil, fmt.Errorf("error deserializing certificate: %w", err)
		}

		certResult := wallet.CertificateResult{Certificate: *cert}

		// Read keyring if present
		if r.ReadByte() == 1 {
			keyringLen := r.ReadVarInt()
			if keyringLen > 0 {
				certResult.Keyring = make(map[string]string, keyringLen)
			}
			for j := uint64(0); j < keyringLen; j++ {
				key := r.ReadString()
				value := r.ReadBase64Int()
				certResult.Keyring[key] = value
			}
		}

		certResult.Verifier = r.ReadIntBytes()

		result.Certificates = append(result.Certificates, certResult)
	}

	r.CheckComplete()
	if r.Err != nil {
		return nil, fmt.Errorf("error deserializing ListCertificates result: %w", r.Err)
	}

	return result, nil
}
