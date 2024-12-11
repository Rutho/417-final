package request

import (
	"client/bitmap"
	"client/peer"
	"math/rand"
)

const MaxBlockSize = 16384 // Maximum block size, in bytes
const QueueSize    = 1     // How many pending requests can be in flight at once

type Request struct {
	// Description:
	//
	//   When we make requests for data from peers, we are actually
	//   requesting a block.
	//
	// Fields:
	//
	//   Block     : the requested block
	//   Peer      : pointer to the peer that the request was sent to
	//   Cancelled : if this request is cancelled. Initially false

	Block      bitmap.Block
	Peer       *peer.Peer
	Cancelled  bool
}

func SelectPiece(p *peer.Peer, self *bitmap.Bitmap) int {
	// Description:
	//
	//   function to select a piece to be requested from a peer
	//
	// Parameters:
	//
	//   a peer and the local client's bitmap
	//
	// Returns:
	//
	//   a piece index
	//
	// Example usage:
	//
	//   piece_index := request.SelectPiece(peer, &localState.Bitmap)

	if p.Am_interested == false {
		panic("state.go: SelectPiece() should only be called on peers that the local client is interested in!")
	}

	// Get the pieces that the peer has
	their_bitmap := p.Bitmap.Complete()

	// Get the pieces that the local client needs
	my_bitmap    := self.PendingIncomplete()

	// Get the intersection of each list
	var intersection []int
	for _, piece := range my_bitmap {
		// Is piece contained in their_bitmap?
		for i := range their_bitmap {
			if their_bitmap[i].Index == piece.Index {
				intersection = append(intersection, piece.Index)
			}
		}
	}

	if len(intersection) == 0 {
		panic("state.go: SendARequest(): We shouldn't be interested in this peer?!?")
	}

	// Pick a random piece
	return intersection[rand.Intn(len(intersection))]
}

func SelectPeer(pl *peer.Peers) *peer.Peer {
	// Description:
	//
	//   function to select a peer to send a request to
	//
	// Parameters:
	//
	//   a pointer to a Peerlist struct
	//
	// Returns:
	//
	//   a pointer to the peer, or nil upon error
	//
	// Example usage:
	//
	//   peer := request.SelectPeer(&localState.Peers)

	interested_peers := pl.PeerInterested()

	if len(interested_peers) == 0 {
		return nil
	}

	// Pick a random peer
	rand_idx := rand.Intn(len(interested_peers))

	return &interested_peers[rand_idx]
}
