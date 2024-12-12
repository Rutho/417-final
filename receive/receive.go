package receive

import (
	"client/bitmap"
	"client/state"
	"client/request"
	"bytes"
	"crypto/sha1"
)


func Process(s *state.State, piece_index int, piece_offset int64, buf []byte) bool {
	// Description:
	//
	//   Handle the procedure upon receiving a Piece from a peer
	//
	// Returns:
	//
	//   false if the piece hash is incorrect
	//   true otherwise

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
	block.Buf = buf
	block.Complete = true
	block.Incomplete = false

	// Update the piece
	piece.Pending -= int64(len(buf))
	piece.Progress += int64(len(buf))

	// Check the hash of the piece if it's complete
	ok := true
	if piece.Complete() {
		ok = VerifyPieceHash(s, Aggregate(piece), piece_index)
		if !ok {
			// Reset the piece
			s.Bitmap.Pieces[piece_index].Blocks   = nil
			s.Bitmap.Pieces[piece_index].Progress = 0
			s.Bitmap.Pieces[piece_index].Pending  = 0
		}
	}

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
	return ok
}

func Aggregate(p *bitmap.Piece) []byte {
	// Combine all the blocks into a single byte array

	var byteSlices [][]byte
	for i := range p.Blocks {
		byteSlices = append(byteSlices, p.Blocks[i].Buf)
	}

	return bytes.Join(byteSlices, nil)
}

func VerifyPieceHash(s *state.State, buf []byte, piece_index int) bool {
	// Description:
	//
	//   Checks that the hashes match
	//
	// Returns:
	//
	//   true if they match
	//   false if no match

	// Get Bitmap
	b := s.Bitmap

	// Get Piece
	piece := &b.Pieces[piece_index]

	// Get expected hash
	e_hash := piece.Hash

	// Hash the received piece
	h := sha1.New()
	r_hash := h.Sum(buf)

	// compare the hashes
	return bytes.Equal(e_hash, r_hash)
}
