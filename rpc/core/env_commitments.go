package core

import "fmt"

const (
	// BridgeCommitmentBlocksLimit is the limit to the number of blocks we can generate a bridge commitment for.
	// This limits the ZK prover time required to compute.
	BridgeCommitmentBlocksLimit = 1000
)

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
