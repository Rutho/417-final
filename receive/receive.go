package receive

import (
	"bytes"
	"client/bitmap"
	"client/request"
	"client/state"
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

	// Use it to find the request
	req_idx := FindRequest(s.RequestQueue, piece_index, piece_offset, int64(len(buf)))
	if req_idx == -1 {
		fmt.Println("Received a block, but couldn't find the request I made for it?!")
		return false
	}

	// Did we already receive this block?
	if s.RequestQueue[req_idx].Block.Buf != nil {
		fmt.Println("Tried to write to a block that already has data??")
		return false
	}

	// Update the block
	for i := range buf {
		s.RequestQueue[req_idx].Block.Buf = append(s.RequestQueue[req_idx].Block.Buf, buf[i])
	}
	s.RequestQueue[req_idx].Block.Complete   = true
	s.RequestQueue[req_idx].Block.Pending    = false
	s.RequestQueue[req_idx].Block.Incomplete = false

	// Get Piece
	piece := &s.Bitmap.Pieces[piece_index]

	// Update the block again for good measure (???)
	for i := range s.Bitmap.Pieces[piece_index].Blocks {
		if s.Bitmap.Pieces[piece_index].Blocks[i].Equals(s.RequestQueue[req_idx].Block) {
			// if s.Bitmap.Pieces[piece_index].Blocks[i].Buf != nil {
				// fmt.Println("it's already got data.")
			// }
			copy(s.Bitmap.Pieces[piece_index].Blocks[i].Buf, buf)
		}
	}

	// Remove the Request from RequestQueue
	s.RequestQueue[req_idx] = s.RequestQueue[len(s.RequestQueue)-1]
	s.RequestQueue = s.RequestQueue[:len(s.RequestQueue)-1]


	// Update the piece
	s.Bitmap.Pieces[piece_index].Progress += int64(len(buf))

	// Check the hash of the piece if it's complete
	ok := true
	if s.Bitmap.Pieces[piece_index].Complete() {
		// fmt.Printf("Piece %d fully downloaded, verifying hash...\n", piece_index)
		// for _, b := range s.Bitmap.Pieces[piece_index].Blocks {
		// 	fmt.Printf("%s ", b)
		// 	if b.Buf == nil {
		// 		// panic("Block has no data!?")
		// 		fmt.Printf("<-NO DATA ")
		// 	}
		// }
		// fmt.Printf("\n")

		ok = VerifyPieceHash(s, Aggregate(piece), piece_index)
		if !ok {
			// Reset the piece
			s.Bitmap.Pieces[piece_index].Reset()

			fmt.Printf("Piece %d fully downloaded, but hash is incorrect.\n", piece_index)
		} else {
			PieceCompleted(s, piece_index)
		}
	} else {
		// fmt.Printf("Piece %d not yet fully downloaded. Progress = %d / %d\n", piece_index, piece.Progress, piece.Length)
	}

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
	h.Write(buf)
	r_hash := h.Sum(nil)

	// compare the hashes
	return bytes.Equal(e_hash, r_hash)
}

func PieceCompleted(s *state.State, piece_index int) {
	// This function is run once a piece is fully downloaded
	fmt.Printf("Piece %d fully downloaded!\n", piece_index)

	// Send "Have" messages
	for i := range s.Peers.List {
		s.Peers.List[i].Connection.SendHave(uint32(piece_index))
	}

	// Write the piece to the file
	s.FM.WriteOffset(s.Bitmap.Pieces[piece_index].FileOffset, Aggregate(&s.Bitmap.Pieces[piece_index]))

	// Release the memory associated with the piece
	for i := range s.Bitmap.Pieces[piece_index].Blocks {
		s.Bitmap.Pieces[piece_index].Blocks[i].Buf = nil
	}
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
