package receive

import (
	"client/bitmap"
	"client/state"
	"client/request"
)


func Process(s *state.State, piece_index int, piece_offset int64, buf []byte) {
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

	if int64(len(buf)) != block.Length {
		panic("The block I received is the wrong size?!")
	}

	// Update the block
	block.Complete = true
	block.Incomplete = false

	// Update the piece
	piece.Pending -= int64(len(buf))
	piece.Progress += int64(len(buf))

	// Identify the corresponding Request
	var req *request.Request = nil
	req_idx := 0
	for i := range s.RequestQueue {
		if s.RequestQueue[i].Block.Equals(block) {
			req = &s.RequestQueue[i]
			req_idx = i
			break
		}
	}

	if req == nil {
		panic("Couldn't find the pending request for this block?!")
	}

	// Remove the Request from RequestQueue
	s.RequestQueue[req_idx] = s.RequestQueue[len(s.RequestQueue)-1]
	s.RequestQueue = s.RequestQueue[:len(s.RequestQueue)-1]

	// Update the peer's upload rate
	// req.Peer.Uploaded_bytes += int64(len(buf))
}
