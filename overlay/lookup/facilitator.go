package lookup

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/bsv-blockchain/go-sdk/transaction"
	"github.com/bsv-blockchain/go-sdk/util"
)

// Facilitator defines the interface for overlay lookup facilitators that can execute lookup queries
type Facilitator interface {
	Lookup(ctx context.Context, url string, question *LookupQuestion) (*LookupAnswer, error)
}

// HTTPSOverlayLookupFacilitator implements the Facilitator interface using HTTPS requests
type HTTPSOverlayLookupFacilitator struct {
	Client util.HTTPClient
}

// Lookup executes a lookup question against the specified URL and returns the answer
func (f *HTTPSOverlayLookupFacilitator) Lookup(ctx context.Context, url string, question *LookupQuestion) (*LookupAnswer, error) {
	q, err := json.Marshal(question)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url+"/lookup", bytes.NewBuffer(q))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Aggregation", "yes")

	resp, err := f.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &util.HTTPError{
			StatusCode: resp.StatusCode,
			Err:        errors.New("lookup failed"),
		}
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "application/octet-stream" {
		return f.parseAggregatedResponse(resp.Body)
	}

	// Default to JSON response
	answer := &LookupAnswer{}
	if err := json.NewDecoder(resp.Body).Decode(&answer); err != nil {
		return nil, err
	}
	return answer, nil
}

// parseAggregatedResponse parses the binary aggregated response format.
// Format:
//
//	[varint: number of outpoints]
//	For each outpoint:
//	  [32 bytes: txid]
//	  [varint: outputIndex]
//	  [varint: contextLength]
//	  [contextLength bytes: context data]
//	[remaining bytes: aggregated BEEF containing all transactions]
func (f *HTTPSOverlayLookupFacilitator) parseAggregatedResponse(body io.Reader) (*LookupAnswer, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	r := util.NewReader(data)

	// Read number of outpoints
	nOutpoints, err := r.ReadVarInt()
	if err != nil {
		return nil, fmt.Errorf("failed to read outpoint count: %w", err)
	}

	// Parse each outpoint
	type outpointInfo struct {
		txid        string
		outputIndex uint32
		context     []byte
	}
	outpoints := make([]outpointInfo, 0, nOutpoints)

	for i := uint64(0); i < nOutpoints; i++ {
		txidBytes, err := r.ReadBytes(32)
		if err != nil {
			return nil, fmt.Errorf("failed to read txid for outpoint %d: %w", i, err)
		}
		txid := hex.EncodeToString(txidBytes)

		outputIndex, err := r.ReadVarInt()
		if err != nil {
			return nil, fmt.Errorf("failed to read outputIndex for outpoint %d: %w", i, err)
		}

		contextLength, err := r.ReadVarInt()
		if err != nil {
			return nil, fmt.Errorf("failed to read contextLength for outpoint %d: %w", i, err)
		}

		var context []byte
		if contextLength > 0 {
			context, err = r.ReadBytes(int(contextLength))
			if err != nil {
				return nil, fmt.Errorf("failed to read context for outpoint %d: %w", i, err)
			}
		}

		outpoints = append(outpoints, outpointInfo{
			txid:        txid,
			outputIndex: uint32(outputIndex),
			context:     context,
		})
	}

	// Read remaining bytes as aggregated BEEF
	aggregatedBeef := r.ReadRemaining()

	// Parse the aggregated BEEF
	beef, err := transaction.NewBeefFromBytes(aggregatedBeef)
	if err != nil {
		return nil, fmt.Errorf("failed to parse aggregated BEEF: %w", err)
	}

	// Extract individual BEEFs for each outpoint
	outputs := make([]*OutputListItem, 0, len(outpoints))
	for _, op := range outpoints {
		tx := beef.FindTransaction(op.txid)
		if tx == nil {
			// Skip outpoints where we can't find the transaction
			continue
		}

		// Get BEEF for this specific transaction
		txBeef, err := transaction.NewBeefFromTransaction(tx)
		if err != nil {
			continue
		}
		beefBytes, err := txBeef.Bytes()
		if err != nil {
			continue
		}

		outputs = append(outputs, &OutputListItem{
			Beef:        beefBytes,
			OutputIndex: op.outputIndex,
			Context:     op.context,
		})
	}

	return &LookupAnswer{
		Type:    AnswerTypeOutputList,
		Outputs: outputs,
	}, nil
}
