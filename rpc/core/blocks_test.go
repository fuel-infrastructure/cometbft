package core

import (
	"encoding/hex"
	"fmt"
	"github.com/cometbft/cometbft/libs/bytes"
	"github.com/cometbft/cometbft/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dbm "github.com/cometbft/cometbft-db"

	abci "github.com/cometbft/cometbft/abci/types"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	rpctypes "github.com/cometbft/cometbft/rpc/jsonrpc/types"
	sm "github.com/cometbft/cometbft/state"
	"github.com/cometbft/cometbft/state/mocks"
)

func TestBlockchainInfo(t *testing.T) {
	cases := []struct {
		min, max     int64
		base, height int64
		limit        int64
		resultLength int64
		wantErr      bool
	}{
		// min > max
		{0, 0, 0, 0, 10, 0, true},  // min set to 1
		{0, 1, 0, 0, 10, 0, true},  // max set to height (0)
		{0, 0, 0, 1, 10, 1, false}, // max set to height (1)
		{2, 0, 0, 1, 10, 0, true},  // max set to height (1)
		{2, 1, 0, 5, 10, 0, true},

		// negative
		{1, 10, 0, 14, 10, 10, false}, // control
		{-1, 10, 0, 14, 10, 0, true},
		{1, -10, 0, 14, 10, 0, true},
		{-9223372036854775808, -9223372036854775788, 0, 100, 20, 0, true},

		// check base
		{1, 1, 1, 1, 1, 1, false},
		{2, 5, 3, 5, 5, 3, false},

		// check limit and height
		{1, 1, 0, 1, 10, 1, false},
		{1, 1, 0, 5, 10, 1, false},
		{2, 2, 0, 5, 10, 1, false},
		{1, 2, 0, 5, 10, 2, false},
		{1, 5, 0, 1, 10, 1, false},
		{1, 5, 0, 10, 10, 5, false},
		{1, 15, 0, 10, 10, 10, false},
		{1, 15, 0, 15, 10, 10, false},
		{1, 15, 0, 15, 20, 15, false},
		{1, 20, 0, 15, 20, 15, false},
		{1, 20, 0, 20, 20, 20, false},
	}

	for i, c := range cases {
		caseString := fmt.Sprintf("test %d failed", i)
		min, max, err := filterMinMax(c.base, c.height, c.min, c.max, c.limit)
		if c.wantErr {
			require.Error(t, err, caseString)
		} else {
			require.NoError(t, err, caseString)
			require.Equal(t, 1+max-min, c.resultLength, caseString)
		}
	}
}

func TestBlockResults(t *testing.T) {
	results := &abci.ResponseFinalizeBlock{
		TxResults: []*abci.ExecTxResult{
			{Code: 0, Data: []byte{0x01}, Log: "ok"},
			{Code: 0, Data: []byte{0x02}, Log: "ok"},
			{Code: 1, Log: "not ok"},
		},
	}

	env := &Environment{}
	env.StateStore = sm.NewStore(dbm.NewMemDB(), sm.StoreOptions{
		DiscardABCIResponses: false,
	})
	err := env.StateStore.SaveFinalizeBlockResponse(100, results)
	require.NoError(t, err)
	mockstore := &mocks.BlockStore{}
	mockstore.On("Height").Return(int64(100))
	mockstore.On("Base").Return(int64(1))
	env.BlockStore = mockstore

	testCases := []struct {
		height  int64
		wantErr bool
		wantRes *ctypes.ResultBlockResults
	}{
		{-1, true, nil},
		{0, true, nil},
		{101, true, nil},
		{100, false, &ctypes.ResultBlockResults{
			Height:                100,
			TxsResults:            results.TxResults,
			FinalizeBlockEvents:   results.Events,
			ValidatorUpdates:      results.ValidatorUpdates,
			ConsensusParamUpdates: results.ConsensusParamUpdates,
		}},
	}

	for _, tc := range testCases {
		res, err := env.BlockResults(&rpctypes.Context{}, &tc.height)
		if tc.wantErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tc.wantRes, res)
		}
	}
}

func TestPadBytes(t *testing.T) {
	input, err := hex.DecodeString("01")
	assert.NoError(t, err)
	expInput, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000001")
	assert.NoError(t, err)
	errInput, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000001")
	assert.NoError(t, err)

	testCases := []struct {
		input     []byte
		length    int
		expOutput []byte
		expError  string
	}{
		{errInput, 16, expInput, "cannot pad bytes because length of bytes array: 32 is greater than given length: 16"},
		{input, 32, expInput, ""}, // Valid
	}

	for _, c := range testCases {
		output, err := padBytes(c.input, c.length)
		if c.expError != "" {
			assert.EqualError(t, err, c.expError)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, c.expOutput, output)
		}
	}
}

func TestTo32PaddedHexBytes(t *testing.T) {
	expOutput, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000001")
	assert.NoError(t, err)

	expOutput2, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000105")
	assert.NoError(t, err)

	testCases := []struct {
		number    uint64
		expOutput []byte
		expError  string
	}{
		{1, expOutput, ""},    // Valid
		{104, expOutput2, ""}, // Valid
	}

	for _, c := range testCases {
		output, err := To32PaddedHexBytes(c.number)
		if c.expError != "" {
			assert.EqualError(t, err, c.expError)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, c.expOutput, output)
		}
	}
}

func TestEncodeBridgeCommitment(t *testing.T) {
	resultsHash1, err := hex.DecodeString("2769641FA3FCF635E78A3DCDAA1FB88B6ED68369100E4E5C3703A54E834C08FE")
	assert.NoError(t, err)
	resultsHash2, err := hex.DecodeString("63B766303EF0EA13BA3D9E281C2E498F76294FEDEEAA32E3D7F1B517BE9CD956")
	assert.NoError(t, err)

	inputs := []ctypes.BridgeCommitmentLeaf{
		{
			Height:      1,
			ResultsHash: resultsHash1,
		},
		{
			Height:      2,
			ResultsHash: resultsHash2,
		},
	}

	expectedEncoding, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000001" +
		"2769641FA3FCF635E78A3DCDAA1FB88B6ED68369100E4E5C3703A54E834C08FE")
	require.NoError(t, err)
	expectedEncoding2, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000002" +
		"63B766303EF0EA13BA3D9E281C2E498F76294FEDEEAA32E3D7F1B517BE9CD956")
	require.NoError(t, err)

	output := make([][]byte, 0, 2)
	output = append(output, expectedEncoding)
	output = append(output, expectedEncoding2)

	env := &Environment{}
	actualEncoding, err := env.encodeBridgeCommitment(inputs)
	require.NoError(t, err)
	require.NotNil(t, actualEncoding)

	// Check that the length of packed bridge commitment leaves is correct
	assert.Equal(t, len(actualEncoding[0]), 64)
	assert.Equal(t, len(actualEncoding[1]), 64)

	assert.Equal(t, output, actualEncoding)
}

func TestFetchBridgeCommitmentLeaves(t *testing.T) {

	env := &Environment{}
	mockStore := &mocks.BlockStore{}
	mockStore.On("LoadBlock", int64(101)).Return(&types.Block{
		Header: types.Header{
			LastResultsHash: bytes.HexBytes("63B766303EF0EA13BA3D9E281C2E498F76294FEDEEAA32E3D7F1B517BE9CD956"),
		},
	})
	mockStore.On("LoadBlock", int64(102)).Return(&types.Block{
		Header: types.Header{
			LastResultsHash: bytes.HexBytes("2769641FA3FCF635E78A3DCDAA1FB88B6ED68369100E4E5C3703A54E834C08FE"),
		},
	})
	env.BlockStore = mockStore

	expectedLeaves := []ctypes.BridgeCommitmentLeaf{
		{
			Height:      100, // Height 100 but getting 101 LastResultsHash
			ResultsHash: bytes.HexBytes("63B766303EF0EA13BA3D9E281C2E498F76294FEDEEAA32E3D7F1B517BE9CD956"),
		},
		{
			Height:      101, // Height 101 but getting 102 LastResultsHash
			ResultsHash: bytes.HexBytes("2769641FA3FCF635E78A3DCDAA1FB88B6ED68369100E4E5C3703A54E834C08FE"),
		},
	}

	actualLeaves, err := env.fetchBridgeCommitmentLeaves(100, 102)
	assert.NoError(t, err)
	assert.Equal(t, expectedLeaves, actualLeaves)

	// Block not found case
	mockStore.On("LoadBlock", int64(103)).Return(nil)
	_, err = env.fetchBridgeCommitmentLeaves(100, 103)
	assert.EqualError(t, err, "couldn't load block 103")
}

//func TestBridgeCommitment(t *testing.T) {
//
//	env := &Environment{}
//	mockStore := &mocks.BlockStore{}
//	mockStore.On("Height").Return(int64(1000))
//	env.BlockStore = mockStore
//
//}
