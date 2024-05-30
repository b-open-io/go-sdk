package transaction

import (
	script "github.com/bitcoin-sv/go-sdk/script"
)

// OrdinalsPrefix contains 'ORD' the inscription protocol prefix.
//
// Check the docs here: https://docs.1satordinals.com/
const OrdinalsPrefix = "ord"

// Inscribe adds an output to the transaction with an inscription.
func (tx *Transaction) Inscribe(ia *script.InscriptionArgs) error {
	s := *ia.LockingScriptPrefix // deep copy

	// add Inscription data
	// (Example: 	OP_FALSE
	// 						OP_IF
	//						OP_PUSH
	// 						"ord"
	//						OP_1
	//						OP_PUSH
	//						"text/plain;charset=utf-8"
	//						OP_0
	//						OP_PUSH
	//						"Hello, world!"
	//						OP_ENDIF
	// )
	// see: https://docs.ordinals.com/inscriptions.html
	_ = s.AppendOpcodes(script.OpFALSE, script.OpIF)
	err := s.AppendPushDataString(OrdinalsPrefix)
	if err != nil {
		return err
	}
	_ = s.AppendOpcodes(script.Op1)
	err = s.AppendPushData([]byte(ia.ContentType))
	if err != nil {
		return err
	}
	_ = s.AppendOpcodes(script.Op0)
	err = s.AppendPushData(ia.Data)
	if err != nil {
		return err
	}
	_ = s.AppendOpcodes(script.OpENDIF)

	if ia.EnrichedArgs != nil {
		if len(ia.EnrichedArgs.OpReturnData) > 0 {

			// FIXME: import cycle
			// // Sign with AIP
			// _, outData, _, err := aip.SignOpReturnData(*signingKey, "BITCOIN_ECDSA", opReturn)
			// if err != nil {
			// 	return nil, err
			// }

			_ = s.AppendOpcodes(script.OpRETURN)
			if err := s.AppendPushDataArray(ia.EnrichedArgs.OpReturnData); err != nil {
				return err
			}
		}
	}

	tx.AddOutput(&TransactionOutput{
		Satoshis:      1,
		LockingScript: &s,
	})
	return nil
}

// InscribeSpecificOrdinal gives you the functionality to choose
// a specific ordinal from the inputs to inscribe.
//
// You need to provide the input and Satoshi range indices in order
// to specify the ordinal you would like to inscribe, along with the
// change addresses to be used for the rest of the Satoshis.
//
// One output will be created with the extra Satoshis and then another
// output will be created with 1 Satoshi with the inscription in it.
func (tx *Transaction) InscribeSpecificOrdinal(ia *script.InscriptionArgs, inputIdx uint32, satoshiIdx uint64,
	extraOutputScript *script.Script) error {
	amount, err := rangeAbove(tx.Inputs, inputIdx, satoshiIdx)
	if err != nil {
		return err
	}

	if tx.OutputCount() > 0 {
		return ErrOutputsNotEmpty
	}

	tx.AddOutput(&TransactionOutput{
		Satoshis:      amount,
		LockingScript: extraOutputScript,
	})

	err = tx.Inscribe(ia)
	if err != nil {
		return err
	}

	return nil
}

// This function returns the Satoshi amount needed for the output slot
// above the ordinal required so that we can separate the out the ordinal
// from the inputs to the outputs.
//
// This is the way ordinals go from inputs to outputs:
// [a b] [c] [d e f] → [? ? ? ?] [? ?]
// To figure out which satoshi goes to which output, go through the input
// satoshis in order and assign each to a question mark:
// [a b] [c] [d e f] → [a b c d] [e f]
//
// For more info check the Ordinals Theory Handbook (https://docs.ordinals.com/faq.html).
func rangeAbove(is []*TransactionInput, inputIdx uint32, satIdx uint64) (uint64, error) {
	if uint32(len(is)) < inputIdx {
		return 0, ErrOutputNoExist
	}

	var acc uint64
	for i, in := range is {
		if uint32(i) >= inputIdx {
			break
		}
		if in.PreviousTxSatoshis == 0 {
			return 0, ErrInputSatsZero
		}
		acc += in.PreviousTxSatoshis
	}
	return acc + satIdx, nil
}
