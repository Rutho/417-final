package receive

import (
	"client/bitmap"
	"client/state"
	"client/request"
	"bytes"
	"crypto/sha1"
	"fmt"
)


func Process(s *state.State, piece_index int, piece_offset int64, buf []byte) bool {
	// Description:
	//
	//   Handle the procedure upon receiving a Piece from a peer
	//
	// Returns:
	//
	//   false if something went wrong
	//   true otherwise

	s.RequestLock.Lock()
	defer s.RequestLock.Unlock()
	s.BitmapLock.Lock()
	defer s.BitmapLock.Unlock()

	// Use it to find the request
	req_idx := FindRequest(s.RequestQueue, piece_index, piece_offset, int64(len(buf)))
	if req_idx == -1 {
		fmt.Println("Received a block, but couldn't find the request I made for it?!")
		return false
	}

	// Update the block
	s.RequestQueue[req_idx].Block.Buf        = buf
	s.RequestQueue[req_idx].Block.Complete   = true
	s.RequestQueue[req_idx].Block.Incomplete = false

	// Remove the Request from RequestQueue
	s.RequestQueue[req_idx] = s.RequestQueue[len(s.RequestQueue)-1]
	s.RequestQueue = s.RequestQueue[:len(s.RequestQueue)-1]

	// Get Piece
	piece := &s.Bitmap.Pieces[piece_index]

	// Update the piece
	piece.Pending -= int64(len(buf))
	piece.Progress += int64(len(buf))

	// Check the hash of the piece if it's complete
	ok := true
	if piece.Complete() {
		ok = VerifyPieceHash(s, Aggregate(piece), piece_index)
		if !ok {
			fmt.Printf(s.Bitmap.Pieces[piece_index].Blocks)
			// Reset the piece
			s.Bitmap.Pieces[piece_index].Blocks   = nil
			s.Bitmap.Pieces[piece_index].Progress = 0
			s.Bitmap.Pieces[piece_index].Pending  = 0
			fmt.Printf("Piece %d fully downloaded, but hash is incorrect.\n", piece_index)
		} else {
			fmt.Printf("Piece %d fully downloaded!\n", piece_index)
		}
	}

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

	// Get Piece
	piece := &s.Bitmap.Pieces[piece_index]

	// Get expected hash
	e_hash := piece.Hash

	// Hash the received piece
	h := sha1.New()
	r_hash := h.Sum(buf)

	// compare the hashes
	return bytes.Equal(e_hash, r_hash)
}

func FindRequest(rq []request.Request, idx int, offset int64, length int64) int {
	// Construct a temporary block for search
	var temp_block bitmap.Block
	temp_block.New(idx, offset, length)

	for i := range rq {
		if rq[i].Block == nil {
			fmt.Printf("[ERROR] The request at index %d has Block set to nil\n", i)
			continue
		}
		if rq[i].Block.Equals(&temp_block) {
			return i
		}
	}
	return -1
}
