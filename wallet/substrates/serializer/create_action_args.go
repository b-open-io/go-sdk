package serializer

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/bsv-blockchain/go-sdk/wallet"
)

// SerializeCreateActionArgs serializes a wallet.CreateActionArgs object into a byte slice
func SerializeCreateActionArgs(args *wallet.CreateActionArgs) ([]byte, error) {
	buf := make([]byte, 0)
	paramWriter := newWriter(&buf)

	// Serialize description
	descBytes := []byte(args.Description)
	paramWriter.writeVarInt(uint64(len(descBytes)))
	paramWriter.writeBytes(descBytes)

	// Serialize input BEEF
	if args.InputBEEF != nil {
		paramWriter.writeVarInt(uint64(len(args.InputBEEF)))
		paramWriter.writeBytes(args.InputBEEF)
	} else {
		paramWriter.writeVarInt(0xFFFFFFFFFFFFFFFF) // -1 in varint
	}

	// Serialize inputs
	if args.Inputs != nil {
		paramWriter.writeVarInt(uint64(len(args.Inputs)))
		for _, input := range args.Inputs {
			// Serialize outpoint
			outpoint, err := encodeOutpoint(input.Outpoint)
			if err != nil {
				return nil, err
			}
			paramWriter.writeBytes(outpoint)

			// Serialize unlocking script
			if input.UnlockingScript != "" {
				script, err := hex.DecodeString(input.UnlockingScript)
				if err != nil {
					return nil, fmt.Errorf("error decoding unlocking script: %v", err)
				}
				paramWriter.writeVarInt(uint64(len(script)))
				paramWriter.writeBytes(script)
			} else {
				paramWriter.writeVarInt(0xFFFFFFFFFFFFFFFF) // -1
				paramWriter.writeVarInt(uint64(input.UnlockingScriptLength))
			}

			// Serialize input description
			inputDesc := []byte(input.InputDescription)
			paramWriter.writeVarInt(uint64(len(inputDesc)))
			paramWriter.writeBytes(inputDesc)

			// Serialize sequence number
			if input.SequenceNumber > 0 {
				paramWriter.writeVarInt(uint64(input.SequenceNumber))
			} else {
				paramWriter.writeVarInt(0xFFFFFFFFFFFFFFFF) // -1
			}
		}
	} else {
		paramWriter.writeVarInt(0xFFFFFFFFFFFFFFFF) // -1
	}

	// Serialize outputs
	if args.Outputs != nil {
		paramWriter.writeVarInt(uint64(len(args.Outputs)))
		for _, output := range args.Outputs {
			// Serialize locking script
			script, err := hex.DecodeString(output.LockingScript)
			if err != nil {
				return nil, fmt.Errorf("error decoding locking script: %v", err)
			}
			paramWriter.writeVarInt(uint64(len(script)))
			paramWriter.writeBytes(script)

			// Serialize satoshis
			paramWriter.writeVarInt(output.Satoshis)

			// Serialize output description
			outputDesc := []byte(output.OutputDescription)
			paramWriter.writeVarInt(uint64(len(outputDesc)))
			paramWriter.writeBytes(outputDesc)

			// Serialize basket
			if output.Basket != "" {
				basket := []byte(output.Basket)
				paramWriter.writeVarInt(uint64(len(basket)))
				paramWriter.writeBytes(basket)
			} else {
				paramWriter.writeVarInt(0xFFFFFFFFFFFFFFFF) // -1
			}

			// Serialize custom instructions
			if output.CustomInstructions != "" {
				ci := []byte(output.CustomInstructions)
				paramWriter.writeVarInt(uint64(len(ci)))
				paramWriter.writeBytes(ci)
			} else {
				paramWriter.writeVarInt(0xFFFFFFFFFFFFFFFF) // -1
			}

			// Serialize tags
			if output.Tags != nil {
				paramWriter.writeVarInt(uint64(len(output.Tags)))
				for _, tag := range output.Tags {
					tagBytes := []byte(tag)
					paramWriter.writeVarInt(uint64(len(tagBytes)))
					paramWriter.writeBytes(tagBytes)
				}
			} else {
				paramWriter.writeVarInt(0xFFFFFFFFFFFFFFFF) // -1
			}
		}
	} else {
		paramWriter.writeVarInt(0xFFFFFFFFFFFFFFFF) // -1
	}

	// Serialize lockTime
	if args.LockTime > 0 {
		paramWriter.writeVarInt(uint64(args.LockTime))
	} else {
		paramWriter.writeVarInt(0xFFFFFFFFFFFFFFFF) // -1
	}

	// Serialize version
	if args.Version > 0 {
		paramWriter.writeVarInt(uint64(args.Version))
	} else {
		paramWriter.writeVarInt(0xFFFFFFFFFFFFFFFF) // -1
	}

	// Serialize labels
	if args.Labels != nil {
		paramWriter.writeVarInt(uint64(len(args.Labels)))
		for _, label := range args.Labels {
			labelBytes := []byte(label)
			paramWriter.writeVarInt(uint64(len(labelBytes)))
			paramWriter.writeBytes(labelBytes)
		}
	} else {
		paramWriter.writeVarInt(0xFFFFFFFFFFFFFFFF) // -1
	}

	// Serialize options
	if args.Options != nil {
		paramWriter.writeByte(1) // options present

		// signAndProcess
		if args.Options.SignAndProcess != nil {
			if *args.Options.SignAndProcess {
				paramWriter.writeByte(1)
			} else {
				paramWriter.writeByte(0)
			}
		} else {
			paramWriter.writeByte(0xFF) // -1
		}

		// acceptDelayedBroadcast
		if args.Options.AcceptDelayedBroadcast != nil {
			if *args.Options.AcceptDelayedBroadcast {
				paramWriter.writeByte(1)
			} else {
				paramWriter.writeByte(0)
			}
		} else {
			paramWriter.writeByte(0xFF) // -1
		}

		// trustSelf
		if args.Options.TrustSelf == "known" {
			paramWriter.writeByte(1)
		} else {
			paramWriter.writeByte(0xFF) // -1
		}

		// knownTxids
		if args.Options.KnownTxids != nil {
			paramWriter.writeVarInt(uint64(len(args.Options.KnownTxids)))
			for _, txid := range args.Options.KnownTxids {
				txidBytes, err := hex.DecodeString(txid)
				if err != nil {
					return nil, fmt.Errorf("error decoding known txid: %v", err)
				}
				paramWriter.writeBytes(txidBytes)
			}
		} else {
			paramWriter.writeVarInt(0xFFFFFFFFFFFFFFFF) // -1
		}

		// returnTXIDOnly
		if args.Options.ReturnTXIDOnly != nil {
			if *args.Options.ReturnTXIDOnly {
				paramWriter.writeByte(1)
			} else {
				paramWriter.writeByte(0)
			}
		} else {
			paramWriter.writeByte(0xFF) // -1
		}

		// noSend
		if args.Options.NoSend != nil {
			if *args.Options.NoSend {
				paramWriter.writeByte(1)
			} else {
				paramWriter.writeByte(0)
			}
		} else {
			paramWriter.writeByte(0xFF) // -1
		}

		// noSendChange
		if args.Options.NoSendChange != nil {
			paramWriter.writeVarInt(uint64(len(args.Options.NoSendChange)))
			for _, outpoint := range args.Options.NoSendChange {
				op, err := encodeOutpoint(outpoint)
				if err != nil {
					return nil, err
				}
				paramWriter.writeBytes(op)
			}
		} else {
			paramWriter.writeVarInt(0xFFFFFFFFFFFFFFFF) // -1
		}

		// sendWith
		if args.Options.SendWith != nil {
			paramWriter.writeVarInt(uint64(len(args.Options.SendWith)))
			for _, txid := range args.Options.SendWith {
				txidBytes, err := hex.DecodeString(txid)
				if err != nil {
					return nil, fmt.Errorf("error decoding send with txid: %v", err)
				}
				paramWriter.writeBytes(txidBytes)
			}
		} else {
			paramWriter.writeVarInt(0xFFFFFFFFFFFFFFFF) // -1
		}

		// randomizeOutputs
		if args.Options.RandomizeOutputs != nil {
			if *args.Options.RandomizeOutputs {
				paramWriter.writeByte(1)
			} else {
				paramWriter.writeByte(0)
			}
		} else {
			paramWriter.writeByte(0xFF) // -1
		}
	} else {
		paramWriter.writeByte(0) // options not present
	}

	return buf, nil
}

// DeserializeCreateActionArgs deserializes a byte slice into a wallet.CreateActionArgs object
func DeserializeCreateActionArgs(data []byte) (*wallet.CreateActionArgs, error) {
	if len(data) == 0 {
		return nil, errors.New("empty message")
	}

	messageReader := newReader(data)
	args := &wallet.CreateActionArgs{}

	// Read description
	descLen, err := messageReader.readVarInt()
	if err != nil {
		return nil, fmt.Errorf("error reading description length: %v", err)
	}
	descBytes, err := messageReader.readBytes(int(descLen))
	if err != nil {
		return nil, fmt.Errorf("error reading description: %v", err)
	}
	args.Description = string(descBytes)

	// Read input BEEF
	inputBeefLen, err := messageReader.readVarInt()
	if err != nil {
		return nil, fmt.Errorf("error reading input BEEF length: %v", err)
	}
	if inputBeefLen != 0xFFFFFFFFFFFFFFFF { // -1 means nil
		args.InputBEEF, err = messageReader.readBytes(int(inputBeefLen))
		if err != nil {
			return nil, fmt.Errorf("error reading input BEEF: %v", err)
		}
	}

	// Read inputs
	inputsLen, err := messageReader.readVarInt()
	if err != nil {
		return nil, fmt.Errorf("error reading inputs length: %v", err)
	}
	if inputsLen != 0xFFFFFFFFFFFFFFFF { // -1 means nil
		args.Inputs = make([]wallet.CreateActionInput, 0, inputsLen)
		for i := uint64(0); i < inputsLen; i++ {
			input := wallet.CreateActionInput{}

			// Read outpoint
			outpointBytes, err := messageReader.readBytes(36) // 32 txid + 4 index
			if err != nil {
				return nil, fmt.Errorf("error reading outpoint: %v", err)
			}
			input.Outpoint, err = decodeOutpoint(outpointBytes)
			if err != nil {
				return nil, fmt.Errorf("error decoding outpoint: %v", err)
			}

			// Read unlocking script
			unlockingScriptLen, err := messageReader.readVarInt()
			if err != nil {
				return nil, fmt.Errorf("error reading unlocking script length: %v", err)
			}
			if unlockingScriptLen != 0xFFFFFFFFFFFFFFFF { // -1 means nil
				scriptBytes, err := messageReader.readBytes(int(unlockingScriptLen))
				if err != nil {
					return nil, fmt.Errorf("error reading unlocking script: %v", err)
				}
				input.UnlockingScript = hex.EncodeToString(scriptBytes)
			} else {
				// Read unlocking script length value
				length, err := messageReader.readVarInt()
				if err != nil {
					return nil, fmt.Errorf("error reading unlocking script length value: %v", err)
				}
				input.UnlockingScriptLength = uint32(length)
			}

			// Read input description
			inputDescLen, err := messageReader.readVarInt()
			if err != nil {
				return nil, fmt.Errorf("error reading input description length: %v", err)
			}
			inputDescBytes, err := messageReader.readBytes(int(inputDescLen))
			if err != nil {
				return nil, fmt.Errorf("error reading input description: %v", err)
			}
			input.InputDescription = string(inputDescBytes)

			// Read sequence number
			seqNum, err := messageReader.readVarInt()
			if err != nil {
				return nil, fmt.Errorf("error reading sequence number: %v", err)
			}
			if seqNum != 0xFFFFFFFFFFFFFFFF { // -1 means nil
				input.SequenceNumber = uint32(seqNum)
			}

			args.Inputs = append(args.Inputs, input)
		}
	}

	// Read outputs
	outputsLen, err := messageReader.readVarInt()
	if err != nil {
		return nil, fmt.Errorf("error reading outputs length: %v", err)
	}
	if outputsLen != 0xFFFFFFFFFFFFFFFF { // -1 means nil
		args.Outputs = make([]wallet.CreateActionOutput, 0, outputsLen)
		for i := uint64(0); i < outputsLen; i++ {
			output := wallet.CreateActionOutput{}

			// Read locking script
			lockingScriptLen, err := messageReader.readVarInt()
			if err != nil {
				return nil, fmt.Errorf("error reading locking script length: %v", err)
			}
			lockingScriptBytes, err := messageReader.readBytes(int(lockingScriptLen))
			if err != nil {
				return nil, fmt.Errorf("error reading locking script: %v", err)
			}
			output.LockingScript = hex.EncodeToString(lockingScriptBytes)

			// Read satoshis
			satoshis, err := messageReader.readVarInt()
			if err != nil {
				return nil, fmt.Errorf("error reading satoshis: %v", err)
			}
			output.Satoshis = satoshis

			// Read output description
			outputDescLen, err := messageReader.readVarInt()
			if err != nil {
				return nil, fmt.Errorf("error reading output description length: %v", err)
			}
			outputDescBytes, err := messageReader.readBytes(int(outputDescLen))
			if err != nil {
				return nil, fmt.Errorf("error reading output description: %v", err)
			}
			output.OutputDescription = string(outputDescBytes)

			// Read basket
			basketLen, err := messageReader.readVarInt()
			if err != nil {
				return nil, fmt.Errorf("error reading basket length: %v", err)
			}
			if basketLen != 0xFFFFFFFFFFFFFFFF { // -1 means nil
				basketBytes, err := messageReader.readBytes(int(basketLen))
				if err != nil {
					return nil, fmt.Errorf("error reading basket: %v", err)
				}
				output.Basket = string(basketBytes)
			}

			// Read custom instructions
			customInstLen, err := messageReader.readVarInt()
			if err != nil {
				return nil, fmt.Errorf("error reading custom instructions length: %v", err)
			}
			if customInstLen != 0xFFFFFFFFFFFFFFFF { // -1 means nil
				customInstBytes, err := messageReader.readBytes(int(customInstLen))
				if err != nil {
					return nil, fmt.Errorf("error reading custom instructions: %v", err)
				}
				output.CustomInstructions = string(customInstBytes)
			}

			// Read tags
			tagsLen, err := messageReader.readVarInt()
			if err != nil {
				return nil, fmt.Errorf("error reading tags length: %v", err)
			}
			if tagsLen != 0xFFFFFFFFFFFFFFFF { // -1 means nil
				output.Tags = make([]string, 0, tagsLen)
				for j := uint64(0); j < tagsLen; j++ {
					tagLen, err := messageReader.readVarInt()
					if err != nil {
						return nil, fmt.Errorf("error reading tag length: %v", err)
					}
					tagBytes, err := messageReader.readBytes(int(tagLen))
					if err != nil {
						return nil, fmt.Errorf("error reading tag: %v", err)
					}
					output.Tags = append(output.Tags, string(tagBytes))
				}
			}

			args.Outputs = append(args.Outputs, output)
		}
	}

	// Read lockTime
	lockTime, err := messageReader.readVarInt()
	if err != nil {
		return nil, fmt.Errorf("error reading lockTime: %v", err)
	}
	if lockTime != 0xFFFFFFFFFFFFFFFF { // -1 means nil
		args.LockTime = uint32(lockTime)
	}

	// Read version
	version, err := messageReader.readVarInt()
	if err != nil {
		return nil, fmt.Errorf("error reading version: %v", err)
	}
	if version != 0xFFFFFFFFFFFFFFFF { // -1 means nil
		args.Version = uint32(version)
	}

	// Read labels
	labelsLen, err := messageReader.readVarInt()
	if err != nil {
		return nil, fmt.Errorf("error reading labels length: %v", err)
	}
	if labelsLen != 0xFFFFFFFFFFFFFFFF { // -1 means nil
		args.Labels = make([]string, 0, labelsLen)
		for i := uint64(0); i < labelsLen; i++ {
			labelLen, err := messageReader.readVarInt()
			if err != nil {
				return nil, fmt.Errorf("error reading label length: %v", err)
			}
			labelBytes, err := messageReader.readBytes(int(labelLen))
			if err != nil {
				return nil, fmt.Errorf("error reading label: %v", err)
			}
			args.Labels = append(args.Labels, string(labelBytes))
		}
	}

	// Read options
	optionsPresent, err := messageReader.readByte()
	if err != nil {
		return nil, fmt.Errorf("error reading options present flag: %v", err)
	}
	if optionsPresent == 1 {
		args.Options = &wallet.CreateActionOptions{}

		// Read signAndProcess
		signAndProcessFlag, err := messageReader.readByte()
		if err != nil {
			return nil, fmt.Errorf("error reading signAndProcess flag: %v", err)
		}
		if signAndProcessFlag != 0xFF { // -1 means nil
			args.Options.SignAndProcess = new(bool)
			*args.Options.SignAndProcess = signAndProcessFlag == 1
		}

		// Read acceptDelayedBroadcast
		acceptDelayedBroadcastFlag, err := messageReader.readByte()
		if err != nil {
			return nil, fmt.Errorf("error reading acceptDelayedBroadcast flag: %v", err)
		}
		if acceptDelayedBroadcastFlag != 0xFF { // -1 means nil
			args.Options.AcceptDelayedBroadcast = new(bool)
			*args.Options.AcceptDelayedBroadcast = acceptDelayedBroadcastFlag == 1
		}

		// Read trustSelf
		trustSelfFlag, err := messageReader.readByte()
		if err != nil {
			return nil, fmt.Errorf("error reading trustSelf flag: %v", err)
		}
		if trustSelfFlag == 1 {
			args.Options.TrustSelf = "known"
		}

		// Read knownTxids
		knownTxidsLen, err := messageReader.readVarInt()
		if err != nil {
			return nil, fmt.Errorf("error reading knownTxids length: %v", err)
		}
		if knownTxidsLen != 0xFFFFFFFFFFFFFFFF { // -1 means nil
			args.Options.KnownTxids = make([]string, 0, knownTxidsLen)
			for i := uint64(0); i < knownTxidsLen; i++ {
				txidBytes, err := messageReader.readBytes(32)
				if err != nil {
					return nil, fmt.Errorf("error reading known txid: %v", err)
				}
				args.Options.KnownTxids = append(args.Options.KnownTxids, hex.EncodeToString(txidBytes))
			}
		}

		// Read returnTXIDOnly
		returnTXIDOnlyFlag, err := messageReader.readByte()
		if err != nil {
			return nil, fmt.Errorf("error reading returnTXIDOnly flag: %v", err)
		}
		if returnTXIDOnlyFlag != 0xFF { // -1 means nil
			args.Options.ReturnTXIDOnly = new(bool)
			*args.Options.ReturnTXIDOnly = returnTXIDOnlyFlag == 1
		}

		// Read noSend
		noSendFlag, err := messageReader.readByte()
		if err != nil {
			return nil, fmt.Errorf("error reading noSend flag: %v", err)
		}
		if noSendFlag != 0xFF { // -1 means nil
			args.Options.NoSend = new(bool)
			*args.Options.NoSend = noSendFlag == 1
		}

		// Read noSendChange
		noSendChangeLen, err := messageReader.readVarInt()
		if err != nil {
			return nil, fmt.Errorf("error reading noSendChange length: %v", err)
		}
		if noSendChangeLen != 0xFFFFFFFFFFFFFFFF { // -1 means nil
			args.Options.NoSendChange = make([]string, 0, noSendChangeLen)
			for i := uint64(0); i < noSendChangeLen; i++ {
				outpointBytes, err := messageReader.readBytes(36) // 32 txid + 4 index
				if err != nil {
					return nil, fmt.Errorf("error reading noSendChange outpoint: %v", err)
				}
				outpoint, err := decodeOutpoint(outpointBytes)
				if err != nil {
					return nil, fmt.Errorf("error decoding noSendChange outpoint: %v", err)
				}
				args.Options.NoSendChange = append(args.Options.NoSendChange, outpoint)
			}
		}

		// Read sendWith
		sendWithLen, err := messageReader.readVarInt()
		if err != nil {
			return nil, fmt.Errorf("error reading sendWith length: %v", err)
		}
		if sendWithLen != 0xFFFFFFFFFFFFFFFF { // -1 means nil
			args.Options.SendWith = make([]string, 0, sendWithLen)
			for i := uint64(0); i < sendWithLen; i++ {
				txidBytes, err := messageReader.readBytes(32)
				if err != nil {
					return nil, fmt.Errorf("error reading sendWith txid: %v", err)
				}
				args.Options.SendWith = append(args.Options.SendWith, hex.EncodeToString(txidBytes))
			}
		}

		// Read randomizeOutputs
		randomizeOutputsFlag, err := messageReader.readByte()
		if err != nil {
			return nil, fmt.Errorf("error reading randomizeOutputs flag: %v", err)
		}
		if randomizeOutputsFlag != 0xFF { // -1 means nil
			args.Options.RandomizeOutputs = new(bool)
			*args.Options.RandomizeOutputs = randomizeOutputsFlag == 1
		}
	}

	return args, nil
}
