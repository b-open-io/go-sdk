package serializer

import (
	"encoding/hex"
	"fmt"

	"github.com/bsv-blockchain/go-sdk/util"
	"github.com/bsv-blockchain/go-sdk/wallet"
)

func SerializeListActionsArgs(args *wallet.ListActionsArgs) ([]byte, error) {
	w := util.NewWriter()

	// Serialize labels
	w.WriteStringSlice(args.Labels)

	// Serialize labelQueryMode
	switch args.LabelQueryMode {
	case "any":
		w.WriteByte(1)
	case "all":
		w.WriteByte(2)
	case "":
		w.WriteByte(0xFF) // -1
	default:
		return nil, fmt.Errorf("invalid label query mode: %s", args.LabelQueryMode)
	}

	// Serialize include options
	w.WriteOptionalBool(args.IncludeLabels)
	w.WriteOptionalBool(args.IncludeInputs)
	w.WriteOptionalBool(args.IncludeInputSourceLockingScripts)
	w.WriteOptionalBool(args.IncludeInputUnlockingScripts)
	w.WriteOptionalBool(args.IncludeOutputs)
	w.WriteOptionalBool(args.IncludeOutputLockingScripts)

	// Serialize limit, offset, and seekPermission
	w.WriteOptionalUint32(args.Limit)
	w.WriteOptionalUint32(args.Offset)
	w.WriteOptionalBool(args.SeekPermission)

	return w.Buf, nil
}

func DeserializeListActionsArgs(data []byte) (*wallet.ListActionsArgs, error) {
	r := util.NewReaderHoldError(data)
	args := &wallet.ListActionsArgs{}

	// Deserialize labels
	args.Labels = r.ReadStringSlice()

	// Deserialize labelQueryMode
	switch r.ReadByte() {
	case 1:
		args.LabelQueryMode = "any"
	case 2:
		args.LabelQueryMode = "all"
	case 0xFF:
		args.LabelQueryMode = ""
	default:
		return nil, fmt.Errorf("invalid label query mode byte: %d", r.ReadByte())
	}

	// Deserialize include options
	args.IncludeLabels = r.ReadOptionalBool()
	args.IncludeInputs = r.ReadOptionalBool()
	args.IncludeInputSourceLockingScripts = r.ReadOptionalBool()
	args.IncludeInputUnlockingScripts = r.ReadOptionalBool()
	args.IncludeOutputs = r.ReadOptionalBool()
	args.IncludeOutputLockingScripts = r.ReadOptionalBool()

	// Deserialize limit, offset, and seekPermission
	args.Limit = r.ReadOptionalUint32()
	args.Offset = r.ReadOptionalUint32()
	args.SeekPermission = r.ReadOptionalBool()

	if r.Err != nil {
		return nil, fmt.Errorf("error reading list action args: %w", r.Err)
	}

	return args, nil
}

func SerializeListActionsResult(result *wallet.ListActionsResult) ([]byte, error) {
	w := util.NewWriter()

	// Serialize totalActions
	w.WriteVarInt(uint64(result.TotalActions))

	// Serialize actions
	w.WriteVarInt(uint64(len(result.Actions)))
	for _, action := range result.Actions {
		// Serialize basic action fields
		txid, err := hex.DecodeString(action.Txid)
		if err != nil {
			return nil, fmt.Errorf("invalid txid hex: %w", err)
		}
		w.WriteBytes(txid)
		w.WriteVarInt(action.Satoshis)

		// Serialize status
		switch action.Status {
		case wallet.ActionStatusCompleted:
			w.WriteByte(byte(wallet.ActionStatusCodeCompleted))
		case wallet.ActionStatusUnprocessed:
			w.WriteByte(byte(wallet.ActionStatusCodeUnprocessed))
		case wallet.ActionStatusSending:
			w.WriteByte(byte(wallet.ActionStatusCodeSending))
		case wallet.ActionStatusUnproven:
			w.WriteByte(byte(wallet.ActionStatusCodeUnproven))
		case wallet.ActionStatusUnsigned:
			w.WriteByte(byte(wallet.ActionStatusCodeUnsigned))
		case wallet.ActionStatusNoSend:
			w.WriteByte(byte(wallet.ActionStatusCodeNoSend))
		case wallet.ActionStatusNonFinal:
			w.WriteByte(byte(wallet.ActionStatusCodeNonFinal))
		default:
			return nil, fmt.Errorf("invalid action status: %s", action.Status)
		}

		// Serialize IsOutgoing, Description, Labels, Version, and LockTime
		w.WriteOptionalBool(&action.IsOutgoing)
		w.WriteString(action.Description)
		w.WriteStringSlice(action.Labels)
		w.WriteVarInt(uint64(action.Version))
		w.WriteVarInt(uint64(action.LockTime))

		// Serialize inputs
		w.WriteVarInt(uint64(len(action.Inputs)))
		for _, input := range action.Inputs {
			opBytes, err := encodeOutpoint(input.SourceOutpoint)
			if err != nil {
				return nil, fmt.Errorf("invalid source outpoint: %w", err)
			}
			w.WriteBytes(opBytes)
			w.WriteVarInt(input.SourceSatoshis)

			// SourceLockingScript
			if err = w.WriteOptionalFromHex(input.SourceLockingScript); err != nil {
				return nil, fmt.Errorf("invalid source locking script: %w", err)
			}

			// UnlockingScript
			if err = w.WriteOptionalFromHex(input.UnlockingScript); err != nil {
				return nil, fmt.Errorf("invalid unlocking script: %w", err)
			}

			w.WriteString(input.InputDescription)
			w.WriteVarInt(uint64(input.SequenceNumber))
		}

		// Serialize outputs
		w.WriteVarInt(uint64(len(action.Outputs)))
		for _, output := range action.Outputs {
			w.WriteVarInt(uint64(output.OutputIndex))
			w.WriteVarInt(output.Satoshis)

			// LockingScript
			if err = w.WriteOptionalFromHex(output.LockingScript); err != nil {
				return nil, fmt.Errorf("invalid locking script: %w", err)
			}

			// Serialize Spendable, OutputDescription, Basket, Tags, and CustomInstructions
			w.WriteOptionalBool(&output.Spendable)
			w.WriteString(output.OutputDescription)
			w.WriteString(output.Basket)
			w.WriteStringSlice(output.Tags)
			w.WriteOptionalString(output.CustomInstructions)
		}
	}

	return w.Buf, nil
}

func DeserializeListActionsResult(data []byte) (*wallet.ListActionsResult, error) {
	r := util.NewReaderHoldError(data)
	result := &wallet.ListActionsResult{}

	// Deserialize totalActions
	result.TotalActions = r.ReadVarInt32()

	// Deserialize actions
	actionCount := r.ReadVarInt()
	result.Actions = make([]wallet.Action, 0, actionCount)
	for i := uint64(0); i < actionCount; i++ {
		action := wallet.Action{}

		// Deserialize basic action fields
		txid := r.ReadBytes(32)
		action.Txid = hex.EncodeToString(txid)
		action.Satoshis = r.ReadVarInt()

		// Deserialize status
		status := r.ReadByte()
		switch wallet.ActionStatusCode(status) {
		case wallet.ActionStatusCodeCompleted:
			action.Status = wallet.ActionStatusCompleted
		case wallet.ActionStatusCodeUnprocessed:
			action.Status = wallet.ActionStatusUnprocessed
		case wallet.ActionStatusCodeSending:
			action.Status = wallet.ActionStatusSending
		case wallet.ActionStatusCodeUnproven:
			action.Status = wallet.ActionStatusUnproven
		case wallet.ActionStatusCodeUnsigned:
			action.Status = wallet.ActionStatusUnsigned
		case wallet.ActionStatusCodeNoSend:
			action.Status = wallet.ActionStatusNoSend
		case wallet.ActionStatusCodeNonFinal:
			action.Status = wallet.ActionStatusNonFinal
		default:
			return nil, fmt.Errorf("invalid status byte %d", status)
		}

		// Deserialize IsOutgoing, Description, Labels, Version, and LockTime
		action.IsOutgoing = r.ReadByte() == 1
		action.Description = r.ReadString()
		action.Labels = r.ReadStringSlice()
		action.Version = r.ReadVarInt32()
		action.LockTime = r.ReadVarInt32()

		// Deserialize inputs
		inputCount := r.ReadVarInt()
		action.Inputs = make([]wallet.ActionInput, 0, inputCount)
		for j := uint64(0); j < inputCount; j++ {
			input := wallet.ActionInput{}

			opBytes := r.ReadBytes(36)
			input.SourceOutpoint, _ = decodeOutpoint(opBytes)

			// Serialize source satoshis, locking script, unlocking script, input description, and sequence number
			input.SourceSatoshis = r.ReadVarInt()
			input.SourceLockingScript = r.ReadOptionalToHex()
			input.UnlockingScript = r.ReadOptionalToHex()
			input.InputDescription = r.ReadString()
			input.SequenceNumber = r.ReadVarInt32()

			// Check for error each loop
			if r.Err != nil {
				return nil, fmt.Errorf("error reading list action input %d: %w", j, r.Err)
			}

			action.Inputs = append(action.Inputs, input)
		}

		// Deserialize outputs
		outputCount := r.ReadVarInt()
		action.Outputs = make([]wallet.ActionOutput, 0, outputCount)
		for k := uint64(0); k < outputCount; k++ {
			output := wallet.ActionOutput{}

			// Serialize output index, satoshis, locking script, spendable, output description, basket, tags,
			// and custom instructions
			output.OutputIndex = r.ReadVarInt32()
			output.Satoshis = r.ReadVarInt()
			output.LockingScript = r.ReadOptionalToHex()
			output.Spendable = r.ReadByte() == 1
			output.OutputDescription = r.ReadString()
			output.Basket = r.ReadString()
			output.Tags = r.ReadStringSlice()
			output.CustomInstructions = r.ReadString()

			// Check for error each loop
			if r.Err != nil {
				return nil, fmt.Errorf("error reading list action output %d: %w", k, r.Err)
			}

			action.Outputs = append(action.Outputs, output)
		}

		// Check for error each loop
		if r.Err != nil {
			return nil, fmt.Errorf("error reading list action %d: %w", i, r.Err)
		}

		result.Actions = append(result.Actions, action)
	}

	if r.Err != nil {
		return nil, fmt.Errorf("error reading list action result: %w", r.Err)
	}

	return result, nil
}
