package serializer

import (
	"fmt"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/bsv-blockchain/go-sdk/util"
	"github.com/bsv-blockchain/go-sdk/wallet"
)

func SerializeCreateSignatureArgs(args *wallet.CreateSignatureArgs) ([]byte, error) {
	w := util.NewWriter()

	// Encode key related params (protocol, key, counterparty, privileged)
	params := KeyRelatedParams{
		ProtocolID:       args.ProtocolID,
		KeyID:            args.KeyID,
		Counterparty:     args.Counterparty,
		Privileged:       &args.Privileged,
		PrivilegedReason: args.PrivilegedReason,
	}
	keyParams, err := encodeKeyRelatedParams(params)
	if err != nil {
		return nil, fmt.Errorf("error encoding key params: %w", err)
	}
	w.WriteBytes(keyParams)

	// Write data or hash flag and content
	if args.Data != nil {
		w.WriteByte(1)
		w.WriteVarInt(uint64(len(args.Data)))
		w.WriteBytes(args.Data)
	} else {
		w.WriteByte(2)
		w.WriteBytes(args.HashToDirectlySign)
	}

	// Write seekPermission flag (-1 if undefined)
	w.WriteOptionalBool(&args.SeekPermission)

	return w.Buf, nil
}

func DeserializeCreateSignatureArgs(data []byte) (*wallet.CreateSignatureArgs, error) {
	r := util.NewReaderHoldError(data)
	args := &wallet.CreateSignatureArgs{}

	// Decode key related params
	params, err := decodeKeyRelatedParams(r)
	if err != nil {
		return nil, fmt.Errorf("error decoding key params: %w", err)
	}
	args.ProtocolID = params.ProtocolID
	args.KeyID = params.KeyID
	args.Counterparty = params.Counterparty
	args.Privileged = util.ReadOptionalBoolAsBool(params.Privileged)
	args.PrivilegedReason = params.PrivilegedReason

	// Read data or hash
	dataTypeFlag := r.ReadByte()
	if dataTypeFlag == 1 {
		dataLen := r.ReadVarInt()
		args.Data = r.ReadBytes(int(dataLen))
	} else if dataTypeFlag == 2 {
		args.HashToDirectlySign = r.ReadBytes(32)
	} else {
		return nil, fmt.Errorf("invalid data type flag: %d", dataTypeFlag)
	}

	// Read seekPermission
	args.SeekPermission = util.ReadOptionalBoolAsBool(r.ReadOptionalBool())

	if r.Err != nil {
		return nil, fmt.Errorf("error deserializing CreateSignature args: %w", r.Err)
	}

	return args, nil
}

func SerializeCreateSignatureResult(result *wallet.CreateSignatureResult) ([]byte, error) {
	w := util.NewWriter()
	w.WriteByte(0) // errorByte = 0 (success)
	w.WriteBytes(result.Signature.Serialize())
	return w.Buf, nil
}

func DeserializeCreateSignatureResult(data []byte) (*wallet.CreateSignatureResult, error) {
	r := util.NewReaderHoldError(data)
	result := &wallet.CreateSignatureResult{}

	// Read error byte (0 = success)
	errorByte := r.ReadByte()
	if errorByte != 0 {
		return nil, fmt.Errorf("createSignature failed with error byte %d", errorByte)
	}

	// Read signature (remaining bytes)
	sig, err := ec.FromDER(r.ReadRemaining())
	if err != nil {
		return nil, fmt.Errorf("error deserializing signature: %w", err)
	}
	result.Signature = *sig

	if r.Err != nil {
		return nil, fmt.Errorf("error deserializing CreateSignature result: %w", r.Err)
	}

	return result, nil
}
