package blobs

import (
	"fmt"

	"github.com/cosmos/gogoproto/proto"

	"github.com/cometbft/cometbft/p2p"
)

var _ p2p.Wrapper = &Blob{}
var _ p2p.Unwrapper = &Message{}

// Wrap implements the p2p Wrapper interface and wraps a blob.
func (m *Blob) Wrap() proto.Message {
	mm := &Message{}
	mm.Sum = &Message_Blob{Blob: m}
	return mm
}

// Unwrap implements the p2p Wrapper interface and unwraps a wrapped blob.
func (m *Message) Unwrap() (proto.Message, error) {
	switch msg := m.Sum.(type) {
	case *Message_Blob:
		return m.GetBlob(), nil

	default:
		return nil, fmt.Errorf("unknown message: %T", msg)
	}
}
