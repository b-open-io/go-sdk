// Package unlocker comment
package unlocker

import (
	"context"
	"errors"

	ec "github.com/bitcoin-sv/go-sdk/primitives/ec"
	script "github.com/bitcoin-sv/go-sdk/script"
	"github.com/bitcoin-sv/go-sdk/transaction"
	sighash "github.com/bitcoin-sv/go-sdk/transaction/sighash"
)

// Getter implements the `bt.UnlockerGetter` interface. It unlocks a Tx locally,
// using a bec PrivateKey.
type Getter struct {
	PrivateKey *ec.PrivateKey
}

// Unlocker builds a new `*unlocker.Local` with the same private key
// as the calling `*local.Getter`.
//
// For an example implementation, see `examples/unlocker_getter/`.
func (g *Getter) Unlocker(ctx context.Context, lockingScript *script.Script) (transaction.Unlocker, error) {
	return &P2PKH{PrivateKey: g.PrivateKey}, nil
}

// P2PKH implements the a simple `bt.Unlocker` interface. It is used to build an unlocking script
// using a bec Private Key.
type P2PKH struct {
	PrivateKey *ec.PrivateKey
}

// UnlockingScript create the unlocking script for a given input using the PrivateKey passed in through the
// the `unlock.Local` struct.
//
// UnlockingScript generates and uses an ECDSA signature for the provided hash digest using the private key
// as well as the public key corresponding to the private key used. The produced
// signature is deterministic (same message and same key yield the same signature) and
// canonical in accordance with RFC6979 and BIP0062.
//
// For example usage, see `examples/create_tx/create_tx.go`
func (l *P2PKH) UnlockingScript(ctx context.Context, tx *transaction.Transaction, params transaction.UnlockerParams) (*script.Script, error) {
	if params.SigHashFlags == 0 {
		params.SigHashFlags = sighash.AllForkID
	}

	if tx.Inputs[params.InputIdx].PreviousTxScript == nil {
		return nil, transaction.ErrEmptyPreviousTxScript
	}
	switch tx.Inputs[params.InputIdx].PreviousTxScript.ScriptType() {
	case script.ScriptTypePubKeyHash, script.ScriptTypePubKeyHashInscription:
		sh, err := tx.CalcInputSignatureHash(params.InputIdx, params.SigHashFlags)
		if err != nil {
			return nil, err
		}

		sig, err := l.PrivateKey.Sign(sh)
		if err != nil {
			return nil, err
		}

		pubKey := l.PrivateKey.PubKey().SerialiseCompressed()
		signature := sig.Serialise()

		uscript, err := script.NewP2PKHUnlockingScript(pubKey, signature, params.SigHashFlags)
		if err != nil {
			return nil, err
		}

		return uscript, nil
	}

	return nil, errors.New("currently only p2pkh supported")
}
