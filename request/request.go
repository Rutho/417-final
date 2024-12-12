package request

import (
	"client/bitmap"
	"client/peer"
	"math/rand"
	"sync"
	"time"
)

const MaxBlockSize = 16384 // Maximum block size, in bytes
const QueueSize    = 20    // How many pending requests can be in flight at once

type Request struct {
	// Description:
	//
	//   When we make requests for data from peers, we are actually
	//   requesting a block.
	//
	// Fields:
	//
	//   Block     : pointer to the requested block
	//   Peer      : pointer to the peer that the request was sent to
	//   Cancelled : if this request is cancelled. Initially false
	//   Timestamp : number of seconds since the Unix epoch
	//   Request_lock : for thread safety

	Block      *bitmap.Block
	Peer       *peer.Peer
	Cancelled  bool
	Timestamp  int64
	Request_lock *sync.RWMutex
}

func SelectBlock(p *bitmap.Piece) *bitmap.Block {
	// Description:
	//
	//   TODO
	//
	// Returns:
	//
	//   a pointer to the block

	for i := range p.Blocks {
		if p.Blocks[i].Incomplete && !p.Blocks[i].Pending {
			return &p.Blocks[i]
		}
	}
	return nil
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
		panic("request.go: SelectPiece() should only be called on peers that the local client is interested in!")
	}

	// Get the pieces that the peer has
	their_bitmap := p.Bitmap.Complete()

	// Get the pieces which I have started downloading
	inprog := self.InProgress()

	// Remove pieces which have no unrequested blocks
	for i := len(inprog) - 1; i >= 0; i-- {
		if len(inprog[i].NotRequested()) == 0 {
			// Remove the item by replacing it with the last item and reslicing
			inprog[i] = inprog[len(inprog)-1]
			inprog = inprog[:len(inprog)-1]
		}
	}


	// Get the intersection of each list
	var intersection []int
	for _, piece := range inprog {
		// Is piece contained in their_bitmap?
		for i := range their_bitmap {
			if their_bitmap[i].Index == piece.Index {
				intersection = append(intersection, piece.Index)
			}
		}
	}

	if len(intersection) > 0 {
		// Pick a random piece
		return intersection[rand.Intn(len(intersection))]
	}

	// OTHERWISE, if they don't have any of my in progress pieces,
	// just pick any piece

	// Get the pieces that the local client needs
	my_bitmap    := self.Incomplete()

	// Get the intersection of each list
	intersection = nil
	for _, piece := range my_bitmap {
		// Is piece contained in their_bitmap?
		for i := range their_bitmap {
			if their_bitmap[i].Index == piece.Index {
				intersection = append(intersection, piece.Index)
			}
		}
	}

	if len(intersection) == 0 {
		panic("request.go: SendARequest(): We shouldn't be interested in this peer?!?")
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

	interested_peers := pl.AmInterested()

	if len(interested_peers) == 0 {
		return nil
	}

	// Filter the list by those who are not choking us
	var not_choking_us []peer.Peer

	for i := range interested_peers {
		if interested_peers[i].Peer_choking == false {
			not_choking_us = append(not_choking_us, interested_peers[i])
		}
	}

	if len(not_choking_us) == 0 {
		return nil
	}

	// Pick a random peer
	rand_idx := rand.Intn(len(not_choking_us))

	return &not_choking_us[rand_idx]
}

func Cancel(rq *[]Request) {
	// Description:
	//
	//   function to cancel in-flight outgoing requests
	//
	// Parameters:
	//
	//   a pointer to a list of pending requests
	//
	// Returns:
	//
	//   nothing
	//
	// Example usage:
	//
	//   TODO

	for i := range *rq {
		piece_index := (*rq)[i].Block.PieceIndex
		offset      := (*rq)[i].Block.Offset
		length      := (*rq)[i].Block.Length
		(*rq)[i].Peer.Connection.SendCancel(uint32(piece_index), uint32(offset), uint32(length))
	}

	*rq = nil
}

func Clean(rq *[]Request, b *bitmap.Bitmap) {
	// Description:
	//
	//   a request was never answered
	//
	// Parameters:
	//
	//   TODO
	//
	// Returns:
	//
	//   TODO
	//
	// Example usage:
	//
	//   TODO

	// Iterate in reverse to safely remove items
	for i := len(*rq) - 1; i >= 0; i-- {
		if time.Now().Unix()-(*rq)[i].Timestamp > 4 {
			b.Pieces[(*rq)[i].Block.PieceIndex].Pending -= (*rq)[i].Block.Length
			(*rq)[i].Block.Pending = false

			// Remove the item by replacing it with the last item and reslicing
			(*rq)[i] = (*rq)[len(*rq)-1]
			(*rq) = (*rq)[:len(*rq)-1]
		}
	}
}
