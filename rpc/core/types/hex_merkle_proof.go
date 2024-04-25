package coretypes

import (
	"github.com/cometbft/cometbft/crypto/merkle"
	"github.com/cometbft/cometbft/libs/bytes"
)

// HexMerkleProof is structurally identical to merkle.Proof but uses HexBytes instead of []byte.
type HexMerkleProof struct {
	Total    int64            `json:"total"`     // Total number of items.
	Index    int64            `json:"index"`     // Index of item to prove.
	LeafHash bytes.HexBytes   `json:"leaf_hash"` // Hash of item value.
	Aunts    []bytes.HexBytes `json:"aunts"`     // Hashes from leaf's sibling to a root's child.
}

// NewHexMerkleProof creates a HexMerkleProof from a merkle.Proof.
func NewHexMerkleProof(proof merkle.Proof) HexMerkleProof {

	var newAunts []bytes.HexBytes
	for _, aunt := range proof.Aunts {
		newAunts = append(newAunts, aunt)
	}

	return HexMerkleProof{
		Total:    proof.Total,
		Index:    proof.Index,
		LeafHash: proof.LeafHash,
		Aunts:    newAunts,
	}
}

// ToMerkleProof converts HexMerkleProof into the original merkle.Proof.
func (proof *HexMerkleProof) ToMerkleProof() *merkle.Proof {

	var newAunts [][]byte
	for _, aunt := range proof.Aunts {
		newAunts = append(newAunts, aunt)
	}

	return &merkle.Proof{
		Total:    proof.Total,
		Index:    proof.Index,
		LeafHash: proof.LeafHash,
		Aunts:    newAunts,
	}
}
