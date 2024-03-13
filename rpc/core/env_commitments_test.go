package core

import (
	"testing"

	"github.com/cometbft/cometbft/state/mocks"
	"github.com/stretchr/testify/assert"
)

func TestValidateBridgeCommitmentRange(t *testing.T) {
	cases := []struct {
		start    uint64
		end      uint64
		expError string
	}{
		{5, 1, "last block is smaller than first block"},
		{0, 5, "the first block is 0"},
		{1, 1002, "the query exceeds the limit of allowed blocks 1000"},
		{1, 1, "cannot create the bridge commitments for an empty set of blocks"},
		{5, 102, "end block 102 is higher than current chain height 100"},
		{5, 101, ""}, // Valid since block 101 is not inclusive
		{5, 100, ""}, // Valid
	}
	env := &Environment{}
	mockStore := &mocks.BlockStore{}
	mockStore.On("Height").Return(int64(100))
	env.BlockStore = mockStore

	for _, c := range cases {
		err := env.validateBridgeCommitmentRange(c.start, c.end)
		if c.expError != "" {
			assert.EqualError(t, err, c.expError)
		} else {
			assert.Nil(t, err)
		}
	}
}

func TestValidateBridgeCommitmentInclusionProofRequest(t *testing.T) {
	cases := []struct {
		height   uint64
		start    uint64
		end      uint64
		expError string
	}{
		{150, 1, 100, "height 150 should be in the end exclusive interval first_block 1 last_block 100"},
		{100, 1, 100, "height 100 should be in the end exclusive interval first_block 1 last_block 100"},
		{99, 1, 100, ""}, // Valid
	}
	env := &Environment{}
	mockStore := &mocks.BlockStore{}
	mockStore.On("Height").Return(int64(1000))
	env.BlockStore = mockStore

	for _, c := range cases {
		err := env.validateBridgeCommitmentInclusionProofRequest(c.height, c.start, c.end)
		if c.expError != "" {
			assert.EqualError(t, err, c.expError)
		} else {
			assert.Nil(t, err)
		}
	}
}
