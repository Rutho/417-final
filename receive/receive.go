package receive

import (
	"client/bitmap"
	"client/state"
	"client/peer"
)


func Process(s *state.State, p *peer.Peer, piece_index int, piece_offset int, buf []byte) {
	// Description:
	//
	//   Handle the procedure upon receiving a Piece from a peeer

	// Get Bitmap
	b := s.Bitmap

	// Get Piece
	piece := &b.Pieces[piece_index]

	// Identify the corresponding block
	var block *bitmap.Block = nil
	for i := range b.Pieces[piece_index].Blocks {
		if b.Pieces[piece_index].Blocks[i].Offset == piece_offset {
			block = &b.Pieces[piece_index].Blocks[i]
			break
		}
	}

	if block == nil {
		panic("This isn't the block I asked for?!")
	}

	if len(buf) != block.Length{
		panic("The block I received is the wrong size?!")
	}

	// Update the block
	block.Complete = true
	block.Incomplete = false

	// Update the piece
	piece.Pending -= len(buf)
	piece.Progress += len(buf)

	// Identify the corresponding Request
	var req *request.Request = nil
	for i := range s.RequestQueue {
		if s.RequestQueue[i].Block.Equals(Block) {
			req = &s.RequestQueue[i]
			break
		}
	}

	if req == nil {
		panic("Couldn't find the pending request for this block?!")
	}

	// Remove the Request from RequestQueue
	s.RequestQueue[i] = s.RequestQueue[len(s.RequestQueue)-1]
	s.RequestQueue = s.RequestQueue[:len(s.RequestQueue)-1]

	// Update the peer's upload rate
	p.Uploaded_bytes += len(buf)
}
