package blobs

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/p2p"
	protoblobs "github.com/cometbft/cometbft/proto/tendermint/blobs"
)

// Reactor handles blob broadcasting amongst peers.
type Reactor struct {
	p2p.BaseReactor

	myTurnToSend bool
	lastReceive  time.Time

	dataSizeBytes  int
	waitBeforeSend time.Duration

	recvBufferCapacity  int
	recvMessageCapacity int
}

// NewReactor returns a new Reactor.
func NewReactor(config *config.BlobsConfig) *Reactor {
	memR := &Reactor{
		myTurnToSend:        config.SendFirst,
		dataSizeBytes:       config.DataSizeBytes,
		waitBeforeSend:      config.WaitBeforeSend,
		recvBufferCapacity:  config.RecvBufferCapacity,
		recvMessageCapacity: config.RecvMessageCapacity,
	}
	memR.BaseReactor = *p2p.NewBaseReactor("Blobs", memR)

	return memR
}

// GetChannels implements Reactor by returning the list of channels for this
// reactor.
func (blobsR *Reactor) GetChannels() []*p2p.ChannelDescriptor {
	return []*p2p.ChannelDescriptor{
		{
			ID:                  BlobsChannel,
			Priority:            5,
			RecvBufferCapacity:  blobsR.recvBufferCapacity,
			RecvMessageCapacity: blobsR.recvMessageCapacity,
			MessageType:         &protoblobs.Message{},
		},
	}
}

// AddPeer implements Reactor.
// It starts a broadcast routine ensuring all txs are forwarded to the given peer.
func (memR *Reactor) AddPeer(peer p2p.Peer) {
	go func() {
		memR.broadcastBlobRoutine(peer)
	}()
}

// Receive implements Reactor.
// It acknowledges any received blobs.
func (blobsR *Reactor) Receive(e p2p.Envelope) {
	blobsR.Logger.Debug("Receive", "src", e.Src, "chId", e.ChannelID, "msg", e.Message)
	switch msg := e.Message.(type) {
	case *protoblobs.Blob:
		blobData := msg.GetData()
		if len(blobData) == 0 {
			blobsR.Logger.Error("received empty blob from peer", "src", e.Src)
			return
		}

		timeElapsed := time.Now().Sub(msg.TimeSent)
		bytesPerSecond := float64(len(blobData)) / timeElapsed.Seconds()
		blobsR.Logger.Info("BLOBS :: received blob",
			"size", len(blobData),
			"id", msg.Id,
			"time_sent", msg.TimeSent,
			"time_elapsed", timeElapsed,
			"bytes_per_second", bytesPerSecond,
		)
		blobsR.myTurnToSend = true
		blobsR.lastReceive = time.Now()

	default:
		blobsR.Logger.Error("unknown message type", "src", e.Src, "chId", e.ChannelID, "msg", e.Message)
		blobsR.Switch.StopPeerForError(e.Src, fmt.Errorf("blobs cannot handle message of type: %T", e.Message))
		return
	}

	// broadcasting happens from go routines per peer
}

func generateRandomData(size int) []byte {
	data := make([]byte, size)
	rand.New(rand.NewSource(time.Now().UnixNano())).Read(data)
	return data
}

func generateRandomId() uint64 {
	return rand.New(rand.NewSource(time.Now().UnixNano())).Uint64()
}

// Send new blobs to peer.
func (blobsR *Reactor) broadcastBlobRoutine(peer p2p.Peer) {

	for {
		if !blobsR.IsRunning() || !peer.IsRunning() {
			return
		}

		if blobsR.myTurnToSend {
			timeUntil := time.Until(blobsR.lastReceive.Add(blobsR.waitBeforeSend))
			blobsR.Logger.Info("BLOBS :: my turn but might need to wait", "wait", timeUntil)
			select {
			case <-time.After(timeUntil):
				break
			case <-peer.Quit():
				return
			case <-blobsR.Quit():
				return
			}

			blobsR.Logger.Info("BLOBS :: generating data...")
			id := generateRandomId()
			data := generateRandomData(blobsR.dataSizeBytes)
			blobsR.Logger.Info(fmt.Sprintf("BLOBS :: generated data of size %d with id %d...", len(data), id))

			success := peer.Send(p2p.Envelope{
				ChannelID: BlobsChannel,
				Message:   &protoblobs.Blob{Id: id, Data: data, TimeSent: time.Now()},
			})
			if !success {
				time.Sleep(UnsuccessfulSendSleepIntervalMS * time.Millisecond)
				continue
			}

			blobsR.myTurnToSend = false
			blobsR.Logger.Info("successfully initialised send", "now", time.Now())
		}

		select {
		case <-time.After(SleepIntervalMS * time.Millisecond):
			continue
		case <-peer.Quit():
			return
		case <-blobsR.Quit():
			return
		}
	}
}
