package coretypes

import (
	"github.com/cometbft/cometbft/libs/bytes"
)

// BridgeCommitmentLeaf the leaf to form a BridgeCommitment.
type BridgeCommitmentLeaf struct {
	Height uint64 `json:"height"`

	// The ResultsHash of blocks is derived at (Height + 1) in the LastResultsHash variable in the Tendermint
	// block header, ref: https://github.com/cometbft/cometbft/blob/v0.38.5/proto/tendermint/types/types.proto#L66.
	// Thus, to reconstruct this root at Height X, you would need the transactions results from Height X - 1.
	LastResultsHash bytes.HexBytes `json:"last_results_hash"`
}

// ResultBridgeCommitment contains the merkle root of successive BridgeCommitmentLeaf.
type ResultBridgeCommitment struct {
	BridgeCommitment bytes.HexBytes `json:"bridge_commitment"`
}

// ResultBridgeCommitmentInclusionProof contains merkle proofs to show that a
// transaction response was used to construct the BridgeCommitment merkle root.
// It also includes the marshalled deterministic transaction result.
type ResultBridgeCommitmentInclusionProof struct {

	// BridgeCommitmentMerkleProof is a merkle proof proving a BridgeCommitmentLeaf was used to
	// construct the BridgeCommitment merkle root.
	BridgeCommitmentMerkleProof HexMerkleProof `json:"bridge_commitment_proof"`

	// LastResultsMerkleProof is a merkle proof proving a transaction response was used to form
	// the LastResultsHash merkle root.
	LastResultsMerkleProof HexMerkleProof `json:"last_results_proof"`

	// BridgeCommitmentLeaf is the bridge commitment leaf involved in the BridgeCommitmentMerkleProof
	// and also the one containing the LastResultsHash that is the root of the LastResultsMerkleProof.
	BridgeCommitmentLeaf BridgeCommitmentLeaf `json:"bridge_commitment_leaf"`

	// TxResultMarshalled is the marshalled deterministic form of the queried transaction's ExecTxResult.
	TxResultMarshalled bytes.HexBytes `json:"tx_result_marshalled"`
}
