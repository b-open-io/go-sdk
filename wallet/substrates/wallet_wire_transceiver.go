package substrates

import (
	"fmt"
	"github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/bsv-blockchain/go-sdk/wallet/serializer"
)

// WalletWireTransceiver implements wallet.Interface
// A way to make remote calls to a wallet over a wallet wire.
type WalletWireTransceiver struct {
	Wire WalletWire
}

func NewWalletWireTransceiver(processor *WalletWireProcessor) *WalletWireTransceiver {
	return &WalletWireTransceiver{Wire: processor}
}

func (t *WalletWireTransceiver) transmit(call Call, originator string, params []byte) ([]byte, error) {
	// Create frame
	frame := serializer.WriteRequestFrame(serializer.RequestFrame{
		Call:       byte(call),
		Originator: originator,
		Params:     params,
	})

	// Transmit frame to processor
	result, err := t.Wire.TransmitToWallet(frame)
	if err != nil {
		return nil, fmt.Errorf("failed to transmit call to wallet wire: %w", err)
	}

	// Parse response
	return serializer.ReadResultFrame(result)
}

func (t *WalletWireTransceiver) CreateAction(args wallet.CreateActionArgs, originator string) (*wallet.CreateActionResult, error) {
	// Serialize the request
	data, err := serializer.SerializeCreateActionArgs(&args)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize create action arguments: %w", err)
	}

	// Send to processor
	resp, err := t.transmit(CallCreateAction, originator, data)
	if err != nil {
		return nil, fmt.Errorf("failed to transmit create action call: %w", err)
	}

	// Deserialize response
	return serializer.DeserializeCreateActionResult(resp)
}

func (t *WalletWireTransceiver) SignAction(args wallet.SignActionArgs, originator string) (*wallet.SignActionResult, error) {
	// Serialize the request
	data, err := serializer.SerializeSignActionArgs(&args)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize sign action arguments: %w", err)
	}

	// Send to processor
	resp, err := t.transmit(CallSignAction, originator, data)
	if err != nil {
		return nil, fmt.Errorf("failed to transmit sign action call: %w", err)
	}

	// Deserialize response
	return serializer.DeserializeSignActionResult(resp)
}

func (t *WalletWireTransceiver) AbortAction(args wallet.AbortActionArgs, originator string) (*wallet.AbortActionResult, error) {
	// Serialize the request
	data, err := serializer.SerializeAbortActionArgs(&args)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize abort action arguments: %w", err)
	}

	// Send to processor
	resp, err := t.transmit(CallAbortAction, originator, data)
	if err != nil {
		return nil, fmt.Errorf("failed to transmit abort action call: %w", err)
	}

	// Deserialize response
	return serializer.DeserializeAbortActionResult(resp)
}

func (t *WalletWireTransceiver) ListActions(args wallet.ListActionsArgs, originator string) (*wallet.ListActionsResult, error) {
	// Serialize the request
	data, err := serializer.SerializeListActionsArgs(&args)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize list action arguments: %w", err)
	}

	// Send to processor
	resp, err := t.transmit(CallListActions, originator, data)
	if err != nil {
		return nil, fmt.Errorf("failed to transmit list action call: %w", err)
	}

	// Deserialize response
	return serializer.DeserializeListActionsResult(resp)
}

func (t *WalletWireTransceiver) InternalizeAction(args wallet.InternalizeActionArgs, originator string) (*wallet.InternalizeActionResult, error) {
	// Serialize the request
	data, err := serializer.SerializeInternalizeActionArgs(&args)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize internalize action arguments: %w", err)
	}

	// Send to processor
	resp, err := t.transmit(CallInternalizeAction, originator, data)
	if err != nil {
		return nil, fmt.Errorf("failed to transmit internalize action call: %w", err)
	}

	// Deserialize response
	return serializer.DeserializeInternalizeActionResult(resp)
}

func (t *WalletWireTransceiver) ListOutputs(args wallet.ListOutputsArgs, originator string) (*wallet.ListOutputsResult, error) {
	// Serialize the request
	data, err := serializer.SerializeListOutputsArgs(&args)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize list outputs arguments: %w", err)
	}

	// Send to processor
	resp, err := t.transmit(CallListOutputs, originator, data)
	if err != nil {
		return nil, fmt.Errorf("failed to transmit list outputs call: %w", err)
	}

	// Deserialize response
	return serializer.DeserializeListOutputsResult(resp)
}
