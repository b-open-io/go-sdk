package substrates

import (
	"errors"
	"fmt"
	"github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/bsv-blockchain/go-sdk/wallet/serializer"
)

// WalletWireProcessor implements the WalletWire interface
type WalletWireProcessor struct {
	Wallet wallet.Interface
}

func NewWalletWireProcessor(wallet wallet.Interface) *WalletWireProcessor {
	return &WalletWireProcessor{Wallet: wallet}
}

func (w *WalletWireProcessor) TransmitToWallet(message []byte) ([]byte, error) {
	if len(message) == 0 {
		return nil, errors.New("empty message")
	}

	requestFrame, err := serializer.ReadRequestFrame(message)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize request frame: %w", err)
	}
	var response []byte
	switch Call(requestFrame.Call) {
	case CallCreateAction:
		response, err = w.processCreateAction(requestFrame)
	case CallSignAction:
		response, err = w.processSignAction(requestFrame)
	case CallAbortAction:
		response, err = w.processAbortAction(requestFrame)
	case CallListActions:
		response, err = w.processListActions(requestFrame)
	case CallInternalizeAction:
		response, err = w.processInternalizeAction(requestFrame)
	case CallListOutputs:
		response, err = w.processListOutputs(requestFrame)
	default:
		return nil, fmt.Errorf("unknown call type: %d", requestFrame.Call)
	}
	if err != nil {
		return nil, fmt.Errorf("error calling %d: %w", requestFrame.Call, err)
	}
	return serializer.WriteResultFrame(response, nil), nil
}

func (w *WalletWireProcessor) processSignAction(requestFrame *serializer.RequestFrame) ([]byte, error) {
	args, err := serializer.DeserializeSignActionArgs(requestFrame.Params)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize sign action args: %w", err)
	}
	result, err := w.Wallet.SignAction(*args, requestFrame.Originator)
	if err != nil {
		return nil, fmt.Errorf("failed to process sign action: %w", err)
	}
	return serializer.SerializeSignActionResult(result)
}

func (w *WalletWireProcessor) processCreateAction(requestFrame *serializer.RequestFrame) ([]byte, error) {
	args, err := serializer.DeserializeCreateActionArgs(requestFrame.Params)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize create action args: %w", err)
	}
	result, err := w.Wallet.CreateAction(*args, requestFrame.Originator)
	if err != nil {
		return nil, fmt.Errorf("failed to process create action: %w", err)
	}
	return serializer.SerializeCreateActionResult(result)
}

func (w *WalletWireProcessor) processAbortAction(requestFrame *serializer.RequestFrame) ([]byte, error) {
	args, err := serializer.DeserializeAbortActionArgs(requestFrame.Params)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize abort action args: %w", err)
	}
	result, err := w.Wallet.AbortAction(*args, requestFrame.Originator)
	if err != nil {
		return nil, fmt.Errorf("failed to process abort action: %w", err)
	}
	return serializer.SerializeAbortActionResult(result)
}

func (w *WalletWireProcessor) processListActions(requestFrame *serializer.RequestFrame) ([]byte, error) {
	args, err := serializer.DeserializeListActionsArgs(requestFrame.Params)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize list action args: %w", err)
	}
	result, err := w.Wallet.ListActions(*args, requestFrame.Originator)
	if err != nil {
		return nil, fmt.Errorf("failed to process list action: %w", err)
	}
	return serializer.SerializeListActionsResult(result)
}

func (w *WalletWireProcessor) processInternalizeAction(requestFrame *serializer.RequestFrame) ([]byte, error) {
	args, err := serializer.DeserializeInternalizeActionArgs(requestFrame.Params)
	if err != nil {
		return nil, fmt.Errorf("failed to internalize list action args: %w", err)
	}
	result, err := w.Wallet.InternalizeAction(*args, requestFrame.Originator)
	if err != nil {
		return nil, fmt.Errorf("failed to process internalize action: %w", err)
	}
	return serializer.SerializeInternalizeActionResult(result)
}

func (w *WalletWireProcessor) processListOutputs(requestFrame *serializer.RequestFrame) ([]byte, error) {
	args, err := serializer.DeserializeListOutputsArgs(requestFrame.Params)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize list outputs args: %w", err)
	}
	result, err := w.Wallet.ListOutputs(*args, requestFrame.Originator)
	if err != nil {
		return nil, fmt.Errorf("failed to process list outputs: %w", err)
	}
	return serializer.SerializeListOutputsResult(result)
}
