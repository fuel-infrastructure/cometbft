package core

import (
	"fmt"
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

//func TestBridgeCommitment(t *testing.T) {
//	block100Results := &abci.ResponseFinalizeBlock{
//		TxResults: []*abci.ExecTxResult{
//			{Code: 0, Data: []byte{0x0a2a307837343144303545393633353835433546634538463037394432346565433839306638376535433845120b0a046675656c12033130351801}, Log: "ok"},
//			{Code: 0, Data: []byte{0x02}, Log: "ok"},
//		},
//	}
//	block101Results := &abci.ResponseFinalizeBlock{
//		TxResults: []*abci.ExecTxResult{
//			{Code: 0, Data: []byte{0x01}, Log: "ok"},
//			{Code: 0, Data: []byte{0x02}, Log: "ok"},
//		},
//	}
//
//	env := &Environment{}
//	env.StateStore = sm.NewStore(dbm.NewMemDB(), sm.StoreOptions{
//		DiscardABCIResponses: false,
//	})
//	err := env.StateStore.SaveFinalizeBlockResponse(100, block100Results)
//	err = env.StateStore.SaveFinalizeBlockResponse(101, block101Results)
//	require.NoError(t, err)
//
//	mockstore := &mocks.BlockStore{}
//	mockstore.On("Height").Return(int64(102))
//	mockstore.On("Base").Return(int64(1))
//	// Mimic block data
//	mockstore.On("LoadBlock", int64(100)).Return(&types.Block{
//		Header: types.Header{
//			DataHash: []byte("B8161C61B8EBBB0AFEDD2FF4921AA839CEA998BE6F202052057A7286D1FF0A67"),
//		},
//	})
//	mockstore.On("LoadBlock", int64(101)).Return(&types.Block{
//		Header: types.Header{
//			DataHash: []byte("C77FB831EBF94EB7AE9323EBD30609EE89F79918B5A532D580894A31A8CFBF37"),
//		},
//	})
//
//	env.BlockStore = mockstore
//
//	testCases := []struct {
//		start   uint64
//		end     uint64
//		wantErr bool
//		wantRes *ctypes.ResultBridgeCommitment
//	}{
//		{100, 102, true, nil},
//	}
//
//	for _, tc := range testCases {
//		res, err := env.BridgeCommitment(&rpctypes.Context{}, tc.start, tc.end)
//		assert.NoError(t, err)
//		assert.NotNil(t, res)
//	}
//}
