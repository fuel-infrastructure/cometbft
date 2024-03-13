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

// BridgeCommitmentInclusionProof creates two inclusion proofs to verify that a transaction in included
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
	// Encode data to match solidity side.
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

	// If there are no transactions in the block it is not possible to generate an inclusion proof for a
	// transaction response.
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
	// header computation.
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
// `abi.encode(...)` in Ethereum. To match `abi.encode(...)`, the height is padding to 32 bytes.
func (env *Environment) encodeBridgeCommitment(leaves []ctypes.BridgeCommitmentLeaf) ([][]byte, error) {

	encodedLeaves := make([][]byte, 0, len(leaves))
	for _, leaf := range leaves {

		// Pad to match `abi.encode` on Ethereum.
		paddedHeight, err := To32PaddedHexBytes(leaf.Height)
		if err != nil {
			return nil, err
		}

		encodedLeaf := append(paddedHeight, leaf.LastResultsHash...)
		encodedLeaves = append(encodedLeaves, encodedLeaf)
	}

	return encodedLeaves, nil
}

// To32PaddedHexBytes takes a number and returns its hex representation padded to 32 bytes.
// Used to mimic the result of `abi.encode(number)` in Ethereum.
func To32PaddedHexBytes(number uint64) ([]byte, error) {
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
