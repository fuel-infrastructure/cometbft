package core

import (
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/cometbft/cometbft/crypto/merkle"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	rpctypes "github.com/cometbft/cometbft/rpc/jsonrpc/types"
	"github.com/cometbft/cometbft/types"
)

const (
	// BridgeCommitmentBlocksLimit is the limit to the number of blocks we can generate a bridge commitment for.
	// This limits the ZK prover time required to compute.
	BridgeCommitmentBlocksLimit = 1000
)

// BridgeCommitment collects the transactions results roots over a provided ordered range of blocks,
// and then creates a new merkle root. The range is end exclusive.
func (env *Environment) BridgeCommitment(_ *rpctypes.Context,
	start, end uint64,
) (*ctypes.ResultBridgeCommitment, error) {
	err := env.validateBridgeCommitmentRange(start, end)
	if err != nil {
		return nil, err
	}

	// Fetch data.
	leaves, err := env.fetchBridgeCommitmentLeaves(start, end)
	if err != nil {
		return nil, err
	}
	// Encode data to match solidity `abi.encode`.
	encodedLeaves, err := env.encodeBridgeCommitment(leaves)
	if err != nil {
		return nil, err
	}
	root := merkle.HashFromByteSlices(encodedLeaves)

	return &ctypes.ResultBridgeCommitment{
		BridgeCommitment: root,
	}, nil
}

// BridgeCommitmentInclusionProof creates two inclusion proofs to verify that a transaction is included
// in a BridgeCommitment. Users can also verify any data in the transactions responses for data
// availability. Users need to provide the height and the transaction index that the inclusion proof is
// for. They also need to provide the indexes of the start and end blocks for which the BridgeCommitment
// merkle root is constructed from. The range for BridgeCommitment is end exclusive.
func (env *Environment) BridgeCommitmentInclusionProof(
	_ *rpctypes.Context,
	height, txIndex int64,
	start, end uint64,
) (*ctypes.ResultBridgeCommitmentInclusionProof, error) {
	err := env.validateBridgeCommitmentInclusionProofRequest(uint64(height), start, end)
	if err != nil {
		return nil, err
	}

	// Fetch data.
	leaves, err := env.fetchBridgeCommitmentLeaves(start, end)
	if err != nil {
		return nil, err
	}
	// Encode data to match solidity `abi.encode`.
	encodedLeaves, err := env.encodeBridgeCommitment(leaves)
	if err != nil {
		return nil, err
	}
	// Get proofs of the BridgeCommitment leaves.
	_, proofs := merkle.ProofsFromByteSlices(encodedLeaves)
	bcProof := proofs[height-int64(leaves[0].Height)]

	// Load the transactions that composed the LastResultsHash at height.
	finalizeBlockResponse, err := env.StateStore.LoadFinalizeBlockResponse(height - 1)
	if err != nil {
		return nil, err
	}

	// If there are no transactions in the block it is not possible to generate an inclusion proof for a transaction
	// response. However, we can still return the proof showing that the block was included in the bridge commitment.
	if len(finalizeBlockResponse.TxResults) == 0 && int(txIndex) == 0 {
		return &ctypes.ResultBridgeCommitmentInclusionProof{
			BridgeCommitmentMerkleProof: *bcProof,
		}, nil
	}

	// Sanity check.
	if int(txIndex) >= len(finalizeBlockResponse.TxResults) {
		return nil, fmt.Errorf("transaction index too high %d", txIndex)
	}

	// Remove non-deterministic fields from ExecTxResult responses to match LastResultsHash from the
	// header computation. Ref: https://github.com/cometbft/cometbft/blob/v0.38.5/state/store.go#L412
	deterministicTxResults := types.NewResults(finalizeBlockResponse.TxResults)
	// Get the merkle proof for this transaction.
	txMerkleProof := deterministicTxResults.ProveResult(int(txIndex))

	return &ctypes.ResultBridgeCommitmentInclusionProof{
		BridgeCommitmentMerkleProof: *bcProof,
		LastResultsMerkleProof:      txMerkleProof,
	}, nil
}

// fetchBridgeCommitmentLeaves takes an end exclusive range of heights and fetches its
// corresponding BridgeCommitmentLeafs.
func (env *Environment) fetchBridgeCommitmentLeaves(start, end uint64) ([]ctypes.BridgeCommitmentLeaf, error) {

	bridgeCommitmentLeaves := make([]ctypes.BridgeCommitmentLeaf, 0, end-start)
	for height := start; height < end; height++ {

		currentBlock := env.BlockStore.LoadBlock(int64(height))
		if currentBlock == nil {
			return nil, fmt.Errorf("couldn't load block %d", height)
		}

		bridgeCommitmentLeaves = append(bridgeCommitmentLeaves, ctypes.BridgeCommitmentLeaf{
			Height:          height,
			LastResultsHash: currentBlock.Header.LastResultsHash,
		})
	}

	return bridgeCommitmentLeaves, nil
}

// encodeBridgeCommitment takes a height and a last result hash, and returns the equivalent of
// `abi.encode(...)` in Ethereum. To match `abi.encode(...)`, the height is padded to 32 bytes.
func (env *Environment) encodeBridgeCommitment(leaves []ctypes.BridgeCommitmentLeaf) ([][]byte, error) {

	encodedLeaves := make([][]byte, 0, len(leaves))
	for _, leaf := range leaves {

		// Pad to match `abi.encode` on Ethereum.
		paddedHeight, err := to32PaddedHexBytes(leaf.Height)
		if err != nil {
			return nil, err
		}

		encodedLeaf := append(paddedHeight, leaf.LastResultsHash...)
		encodedLeaves = append(encodedLeaves, encodedLeaf)
	}

	return encodedLeaves, nil
}

// to32PaddedHexBytes takes a number and returns its hex representation padded to 32 bytes.
// Used to mimic the result of `abi.encode(number)` in Ethereum.
func to32PaddedHexBytes(number uint64) ([]byte, error) {
	hexRepresentation := strconv.FormatUint(number, 16)
	// Make sure hex representation has even length.
	// The `strconv.FormatUint` can return odd length hex encodings.
	// For example, `strconv.FormatUint(10, 16)` returns `a`.
	// Thus, we need to pad it.
	if len(hexRepresentation)%2 == 1 {
		hexRepresentation = "0" + hexRepresentation
	}
	hexBytes, hexErr := hex.DecodeString(hexRepresentation)
	if hexErr != nil {
		return nil, hexErr
	}
	paddedBytes, padErr := padBytes(hexBytes, 32)
	if padErr != nil {
		return nil, padErr
	}
	return paddedBytes, nil
}

// padBytes Pad bytes to given length.
func padBytes(byt []byte, length int) ([]byte, error) {
	l := len(byt)
	if l > length {
		return nil, fmt.Errorf(
			"cannot pad bytes because length of bytes array: %d is greater than given length: %d",
			l,
			length,
		)
	}
	if l == length {
		return byt, nil
	}
	tmp := make([]byte, length)
	copy(tmp[length-l:], byt)
	return tmp, nil
}

// ----------------------------------------------

// validateBridgeCommitmentRange runs basic checks on the asc sorted list of
// heights that will be used successively to generate bridge commitments over
// the defined set of height/s.
func (env *Environment) validateBridgeCommitmentRange(start, end uint64) error {
	if start == 0 {
		return fmt.Errorf("the first block is 0")
	}
	if start > end {
		return fmt.Errorf("last block is smaller than first block")
	}
	heightsRange := end - start
	if heightsRange == 0 {
		return fmt.Errorf("cannot create the bridge commitments for an empty set of blocks")
	}
	if heightsRange > uint64(BridgeCommitmentBlocksLimit) {
		return fmt.Errorf("the query exceeds the limit of allowed blocks %d", BridgeCommitmentBlocksLimit)
	}
	// The bridge commitment range is end exclusive.
	if end > uint64(env.BlockStore.Height())+1 {
		return fmt.Errorf(
			"end block %d is higher than current chain height %d",
			end,
			env.BlockStore.Height(),
		)
	}
	return nil
}

// validateBridgeCommitmentInclusionProofRequest validates the request to generate a bridge commitment
// inclusion proof.
func (env *Environment) validateBridgeCommitmentInclusionProofRequest(height, start, end uint64) error {
	err := env.validateBridgeCommitmentRange(start, end)
	if err != nil {
		return err
	}
	if height < start || height >= end {
		return fmt.Errorf(
			"height %d should be in the end exclusive interval first_block %d last_block %d",
			height,
			start,
			end,
		)
	}
	return nil
}
