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

	expOutput2, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000068")
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
	lastResultsHash1, err := hex.DecodeString("2769641FA3FCF635E78A3DCDAA1FB88B6ED68369100E4E5C3703A54E834C08FE")
	assert.NoError(t, err)
	lastResultsHash2, err := hex.DecodeString("63B766303EF0EA13BA3D9E281C2E498F76294FEDEEAA32E3D7F1B517BE9CD956")
	assert.NoError(t, err)

	inputs := []ctypes.BridgeCommitmentLeaf{
		{
			Height:          1,
			LastResultsHash: lastResultsHash1,
		},
		{
			Height:          2,
			LastResultsHash: lastResultsHash2,
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
	mockStore.On("LoadBlock", int64(100)).Return(&types.Block{
		Header: types.Header{
			LastResultsHash: bytes.HexBytes("63B766303EF0EA13BA3D9E281C2E498F76294FEDEEAA32E3D7F1B517BE9CD956"),
		},
	})
	mockStore.On("LoadBlock", int64(101)).Return(&types.Block{
		Header: types.Header{
			LastResultsHash: bytes.HexBytes("2769641FA3FCF635E78A3DCDAA1FB88B6ED68369100E4E5C3703A54E834C08FE"),
		},
	})
	env.BlockStore = mockStore

	expectedLeaves := []ctypes.BridgeCommitmentLeaf{
		{
			Height:          100,
			LastResultsHash: bytes.HexBytes("63B766303EF0EA13BA3D9E281C2E498F76294FEDEEAA32E3D7F1B517BE9CD956"),
		},
		{
			Height:          101,
			LastResultsHash: bytes.HexBytes("2769641FA3FCF635E78A3DCDAA1FB88B6ED68369100E4E5C3703A54E834C08FE"),
		},
	}

	actualLeaves, err := env.fetchBridgeCommitmentLeaves(100, 102)
	assert.NoError(t, err)
	assert.Equal(t, expectedLeaves, actualLeaves)

	// Block not found case
	mockStore.On("LoadBlock", int64(102)).Return(nil)
	_, err = env.fetchBridgeCommitmentLeaves(100, 103)
	assert.EqualError(t, err, "couldn't load block 102")
}

func TestBridgeCommitment(t *testing.T) {
	/**
	BridgeCommitmentLeaf memory bridgeCommitmentLeaf = BridgeCommitmentLeaf(100, hex"2F082AF1B4E2E26251EC6F658AF6528BC8D28BA8AB1F89D0108E0CD8187D6006");
	bytes32 height100Hash = sha256(abi.encodePacked(
		bytes1(0x00), // LEAF_PREFIX
		abi.encode(bridgeCommitmentLeaf)
	));
	BridgeCommitmentLeaf memory bridgeCommitmentLeaf2 = BridgeCommitmentLeaf(101, hex"52F3AC2AD13294B90F8B35B238A3F4B11707F18CD4DB0620A17EACE59C04DC89");
	bytes32 height101Hash = sha256(abi.encodePacked(
		bytes1(0x00), // LEAF_PREFIX
		abi.encode(bridgeCommitmentLeaf2)
	));
	bytes32 bridgeCommitmentRoot = sha256(abi.encodePacked(
		bytes1(0x01), // NODE_PREFIX
		height100Hash,
		height101Hash
	));
	console.logBytes32(bridgeCommitmentRoot);
	*/
	// Root calculated using the above.
	bridgeCommitmentRoot, err := hex.DecodeString("6a9fc4ba63cc5a1bcc97fd79dc7304c64bd530d82d88fb4e4a234a35776be209")
	assert.NoError(t, err)

	height100ResultsHash, err := hex.DecodeString("2F082AF1B4E2E26251EC6F658AF6528BC8D28BA8AB1F89D0108E0CD8187D6006")
	assert.NoError(t, err)

	height101ResultsHash, err := hex.DecodeString("52F3AC2AD13294B90F8B35B238A3F4B11707F18CD4DB0620A17EACE59C04DC89")
	assert.NoError(t, err)

	env := &Environment{}
	mockStore := &mocks.BlockStore{}
	mockStore.On("Height").Return(int64(1000))
	mockStore.On("LoadBlock", int64(100)).Return(&types.Block{
		Header: types.Header{
			LastResultsHash: height100ResultsHash,
		},
	})
	mockStore.On("LoadBlock", int64(101)).Return(&types.Block{
		Header: types.Header{
			LastResultsHash: height101ResultsHash,
		},
	})
	env.BlockStore = mockStore

	actualResult, err := env.BridgeCommitment(nil, 100, 102)
	assert.NoError(t, err)
	assert.Equal(t, bytes.HexBytes(bridgeCommitmentRoot), actualResult.BridgeCommitment)
}

func TestBridgeCommitmentInclusionProof(t *testing.T) {
	// Generate the last results hash in the block's header, transactions are already deterministic
	txResultsH10 := []*abci.ExecTxResult{
		{Code: 0, Data: []byte("one")},
		{Code: 0, Data: []byte("two")},
	}

	dummyStateStore := &mocks.Store{}
	dummyStateStore.On("LoadFinalizeBlockResponse", int64(10)).Return(&abci.ResponseFinalizeBlock{
		TxResults: txResultsH10,
	}, nil)
	txResultsH1Root := sm.TxResultsHash(txResultsH10)

	// Mock the block containing the last results hash, which is height + 1
	dummyBlockStore := &mocks.BlockStore{}
	dummyBlockStore.On("Height").Return(int64(20))
	dummyBlockStore.On("LoadBlock", int64(11)).Return(&types.Block{
		Header: types.Header{
			LastResultsHash: txResultsH1Root,
		},
	})

	env := &Environment{}
	env.BlockStore = dummyBlockStore
	env.StateStore = dummyStateStore

	// Generate the bridge commitment root. Here we are using the transactions from Height 10 and their merkle root
	// is found in the last results hash in block header 11
	bcRoot, err := env.BridgeCommitment(nil, 11, 12)
	assert.NoError(t, err)

	// Get the inclusion proofs
	proofs, err := env.BridgeCommitmentInclusionProof(nil, 11, 1, 11, 12)
	assert.NoError(t, err)

	// First, proof the transaction is included in the last results hash on the next block
	tx1Bz, err := txResultsH10[1].Marshal()
	assert.NoError(t, err)

	err = proofs.LastResultsMerkleProof.Verify(txResultsH1Root, tx1Bz)
	assert.NoError(t, err)

	// Second, proof that the block containing the last results hash was included in the bridge commitment
	leafBz, err := env.encodeBridgeCommitment([]ctypes.BridgeCommitmentLeaf{
		{
			Height:          11,
			LastResultsHash: txResultsH1Root,
		},
	})
	assert.NoError(t, err)

	err = proofs.BridgeCommitmentMerkleProof.Verify(bcRoot.BridgeCommitment, leafBz[0])
	assert.NoError(t, err)
}
