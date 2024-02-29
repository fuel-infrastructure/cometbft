package core

import (
	"fmt"
	"github.com/cometbft/cometbft/state/mocks"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPaginationPage(t *testing.T) {
	cases := []struct {
		totalCount int
		perPage    int
		page       int
		newPage    int
		expErr     bool
	}{
		{0, 10, 1, 1, false},

		{0, 10, 0, 1, false},
		{0, 10, 1, 1, false},
		{0, 10, 2, 0, true},

		{5, 10, -1, 0, true},
		{5, 10, 0, 1, false},
		{5, 10, 1, 1, false},
		{5, 10, 2, 0, true},
		{5, 10, 2, 0, true},

		{5, 5, 1, 1, false},
		{5, 5, 2, 0, true},
		{5, 5, 3, 0, true},

		{5, 3, 2, 2, false},
		{5, 3, 3, 0, true},

		{5, 2, 2, 2, false},
		{5, 2, 3, 3, false},
		{5, 2, 4, 0, true},
	}

	for _, c := range cases {
		p, err := validatePage(&c.page, c.perPage, c.totalCount)
		if c.expErr {
			assert.Error(t, err)
			continue
		}

		assert.Equal(t, c.newPage, p, fmt.Sprintf("%v", c))
	}

	// nil case
	p, err := validatePage(nil, 1, 1)
	if assert.NoError(t, err) {
		assert.Equal(t, 1, p)
	}
}

func TestPaginationPerPage(t *testing.T) {
	cases := []struct {
		totalCount int
		perPage    int
		newPerPage int
	}{
		{5, 0, defaultPerPage},
		{5, 1, 1},
		{5, 2, 2},
		{5, defaultPerPage, defaultPerPage},
		{5, maxPerPage - 1, maxPerPage - 1},
		{5, maxPerPage, maxPerPage},
		{5, maxPerPage + 1, maxPerPage},
	}
	env := &Environment{}
	for _, c := range cases {
		p := env.validatePerPage(&c.perPage)
		assert.Equal(t, c.newPerPage, p, fmt.Sprintf("%v", c))
	}

	// nil case
	p := env.validatePerPage(nil)
	assert.Equal(t, defaultPerPage, p)
}

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
		{5, 105, "end block 105 needs to be higher than current chain height 100 + 1"},
		{5, 101, "end block 101 needs to be higher than current chain height 100 + 1"},
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
