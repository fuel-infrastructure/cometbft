package types

import (
	"github.com/cometbft/cometbft/pkg/consts"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
)

func UnmarshalBlobTx(tx Tx) (blobTx cmtproto.BlobTx, isBlob bool) {
	err := blobTx.Unmarshal(tx)
	if err != nil {
		return cmtproto.BlobTx{}, false
	}
	// perform some quick basic checks to prevent false positives
	if blobTx.TypeId != consts.ProtoBlobTxTypeID {
		return blobTx, false
	}
	return blobTx, true
}

func MarshalBlobTx(tx []byte, blob *cmtproto.Blob) (Tx, error) {
	bTx := cmtproto.BlobTx{
		Tx:     tx,
		Blob:   blob,
		TypeId: consts.ProtoBlobTxTypeID,
	}
	return bTx.Marshal()
}
