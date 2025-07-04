package serializer

import (
	"fmt"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/bsv-blockchain/go-sdk/util"
	"github.com/bsv-blockchain/go-sdk/wallet"
)

const (
	acquisitionProtocolDirect   = 1
	acquisitionProtocolIssuance = 2

	keyRingRevealerCertifier = 11
)

func SerializeAcquireCertificateArgs(args *wallet.AcquireCertificateArgs) ([]byte, error) {
	w := util.NewWriter()

	// Encode type (base64)
	w.WriteBytes(args.Type[:])

	// Encode certifier (hex)
	w.WriteBytes(args.Certifier.Compressed())

	// Encode fields
	fieldEntries := make([]string, 0, len(args.Fields))
	// TODO: Iterating over maps doesn't guarantee order to be consistent
	for k := range args.Fields {
		fieldEntries = append(fieldEntries, k)
	}
	w.WriteVarInt(uint64(len(fieldEntries)))
	for _, key := range fieldEntries {
		keyBytes := []byte(key)
		w.WriteVarInt(uint64(len(keyBytes)))
		w.WriteBytes(keyBytes)
		valueBytes := []byte(args.Fields[key])
		w.WriteVarInt(uint64(len(valueBytes)))
		w.WriteBytes(valueBytes)
	}

	// Encode privileged params
	w.WriteBytes(encodePrivilegedParams(args.Privileged, args.PrivilegedReason))

	// Encode acquisition protocol (1 = direct, 2 = issuance)
	switch args.AcquisitionProtocol {
	case wallet.AcquisitionProtocolDirect:
		w.WriteByte(acquisitionProtocolDirect)
		// Serial number (base64)
		if args.SerialNumber == [32]byte{} {
			return nil, fmt.Errorf("serialNumber is empty")
		}
		w.WriteBytes(args.SerialNumber[:])

		// Revocation outpoint
		w.WriteBytes(encodeOutpoint(args.RevocationOutpoint))

		// Signature (hex)
		var sigBytes []byte
		if args.Signature != nil {
			sigBytes = args.Signature.Serialize()
		}
		w.WriteIntBytes(sigBytes)

		// Keyring revealer
		if args.KeyringRevealer.Certifier {
			w.WriteByte(keyRingRevealerCertifier)
		} else {
			w.WriteBytes(args.KeyringRevealer.PubKey.Compressed())
		}

		// Keyring for subject
		keyringKeys := make([]string, 0, len(args.KeyringForSubject))
		for k := range args.KeyringForSubject {
			keyringKeys = append(keyringKeys, k)
		}
		w.WriteVarInt(uint64(len(keyringKeys)))
		for _, key := range keyringKeys {
			keyBytes := []byte(key)
			w.WriteVarInt(uint64(len(keyBytes)))
			w.WriteBytes(keyBytes)
			if err := w.WriteIntFromBase64(args.KeyringForSubject[key]); err != nil {
				return nil, fmt.Errorf("invalid keyringForSubject value base64: %w", err)
			}
		}
	case wallet.AcquisitionProtocolIssuance:
		w.WriteByte(acquisitionProtocolIssuance)
		// Certifier URL
		urlBytes := []byte(args.CertifierUrl)
		w.WriteVarInt(uint64(len(urlBytes)))
		w.WriteBytes(urlBytes)
	default:
		return nil, fmt.Errorf("invalid acquisition protocol: %s", args.AcquisitionProtocol)
	}

	return w.Buf, nil
}

func DeserializeAcquireCertificateArgs(data []byte) (*wallet.AcquireCertificateArgs, error) {
	r := util.NewReaderHoldError(data)
	args := &wallet.AcquireCertificateArgs{}

	// Read type (base64) and certifier (hex)
	copy(args.Type[:], r.ReadBytes(sizeType))
	parsedCertifier, err := ec.PublicKeyFromBytes(r.ReadBytes(sizePubKey))
	if err != nil {
		return nil, fmt.Errorf("error parsing certifier public key: %w", err)
	}
	args.Certifier = parsedCertifier

	// Read fields
	fieldsLength := r.ReadVarInt()
	if fieldsLength > 0 {
		args.Fields = make(map[string]string, fieldsLength)
	}
	for i := uint64(0); i < fieldsLength; i++ {
		fieldName := string(r.ReadIntBytes())
		fieldValue := string(r.ReadIntBytes())

		if r.Err != nil {
			return nil, fmt.Errorf("error reading field %s: %w", fieldName, r.Err)
		}

		args.Fields[fieldName] = fieldValue
	}

	// Read privileged parameters
	args.Privileged, args.PrivilegedReason = decodePrivilegedParams(r)

	// Read acquisition protocol
	acquisitionProtocolFlag := r.ReadByte()
	switch acquisitionProtocolFlag {
	case acquisitionProtocolDirect:
		args.AcquisitionProtocol = wallet.AcquisitionProtocolDirect
	case acquisitionProtocolIssuance:
		args.AcquisitionProtocol = wallet.AcquisitionProtocolIssuance
	default:
		return nil, fmt.Errorf("invalid acquisition protocol flag: %d", acquisitionProtocolFlag)
	}

	if args.AcquisitionProtocol == wallet.AcquisitionProtocolDirect {
		// Read serial number
		copy(args.SerialNumber[:], r.ReadBytes(sizeSerial))

		// Read revocation outpoint
		revocationOutpoint, err := decodeOutpoint(&r.Reader)
		if err != nil {
			return nil, fmt.Errorf("error decoding outpoint: %w", err)
		}
		args.RevocationOutpoint = revocationOutpoint

		// Read signature
		sigBytes := r.ReadIntBytes()
		if len(sigBytes) > 0 {
			sig, err := ec.ParseSignature(sigBytes)
			if err != nil {
				return nil, fmt.Errorf("error parsing signature: %w", err)
			}
			args.Signature = sig
		}

		// Read keyring revealer
		keyringRevealerIdentifier := r.ReadByte()
		if keyringRevealerIdentifier == keyRingRevealerCertifier {
			args.KeyringRevealer = wallet.KeyringRevealer{
				Certifier: true,
			}
		} else {
			// The keyringRevealerIdentifier is the first byte of the PubKey
			keyringRevealerFullBytes := append([]byte{keyringRevealerIdentifier}, r.ReadBytes(sizePubKey-1)...)
			parsedKeyringPubKey, err := ec.PublicKeyFromBytes(keyringRevealerFullBytes)
			if err != nil {
				return nil, fmt.Errorf("error parsing keyring revealer public key: %w", err)
			}
			args.KeyringRevealer.PubKey = parsedKeyringPubKey
		}

		// Read keyring for subject
		keyringEntriesLength := r.ReadVarInt()
		if keyringEntriesLength > 0 {
			args.KeyringForSubject = make(map[string]string, keyringEntriesLength)
		}

		for i := uint64(0); i < keyringEntriesLength; i++ {
			fieldKeyLength := r.ReadVarInt()
			fieldKeyBytes := r.ReadBytes(int(fieldKeyLength))
			fieldKey := string(fieldKeyBytes)

			args.KeyringForSubject[fieldKey] = r.ReadBase64Int()
			if r.Err != nil {
				return nil, fmt.Errorf("error reading keyring for subject %s: %w", fieldKey, r.Err)
			}
		}
	} else {
		// Read certifier URL
		certifierUrlLength := r.ReadVarInt()
		certifierUrlBytes := r.ReadBytes(int(certifierUrlLength))
		args.CertifierUrl = string(certifierUrlBytes)
	}

	r.CheckComplete()
	if r.Err != nil {
		return nil, fmt.Errorf("error deserializing acquireCertificate args: %w", r.Err)
	}

	return args, nil
}
