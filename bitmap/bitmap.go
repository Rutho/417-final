package bitmap

import (
	"fmt"
	"math"
	"sync"
)

type Bitmap struct {
	// Description:
	//
	//   structure that stores the download progress of a target file
	//
	// Fields:
	//
	//   Pieces_lock : for thread safety
	//   Pieces      : slice of pieces

	Pieces_lock *sync.RWMutex
	Pieces      []Piece
}

type Piece struct {
	// Description:
	//
	//   A file is broken into pieces.
	//
	// Fields:
	//
	//   Index      : index of this piece
	//   Length     : size of this piece, in bytes
	//   Hash       : hash of this piece
	//   FileOffset : how many bytes into the file this piece is
	//   Piece_lock : for thread safety
	//   Blocks     : slice of blocks representing this piece. Blocks may overlap (?)
	//   Progress   : how many bytes are currently downloaded
	//   Pending    : how many bytes are downloaded, assuming in-flight requests succeed

	Index      int
	Length     int64
	Hash       []byte
	FileOffset int64
	Piece_lock *sync.RWMutex

	Blocks     []Block
	Progress   int64
	Pending    int64
}

type Block struct {
	// Description:
	//
	//   A block is a part of a piece, and we can fully define a block by
	//   the index of the piece it’s part of, its byte offset within the
	//   piece, and its length.
	//
	// Fields:
	//
	//   PieceIndex : the index of the piece that this block belongs to
	//   Offset     : the byte offset into the piece
	//   Length     : number of bytes this block is
	//   Block_lock : for thread safety
	//   Buf        : the actual data
	//   Complete   : whether the block is received
	//   Incomplete : whether the block is received

	PieceIndex int
	Offset     int64
	Length     int64
	Block_lock *sync.RWMutex

	Buf        []byte

	Complete   bool
	Pending    bool
	Incomplete bool
}

func (b Bitmap) String() string {
	// Description:
	//
	//   method on the Bitmap struct that provides a human-readable string representation
	//
	// Parameters:
	//
	//   none
	//
	// Returns:
	//
	//   A string of 1's and 0's representing the Bitmap
	//
	// Example usage:
	//
	//   var localState State
	//   localState.Bitmap.New()
	//   fmt.Println(localState.Bitmap)

	var output string = ""

	// Calculate the total number of blocks
	num_b := 0
	for i := range b.Pieces {
		num_b += len(b.Pieces[i].Blocks)
	}

	// Calculate how many blocks are complete
	completed_b := 0
	for i := range b.Pieces {
		for j := range b.Pieces[i].Blocks {
			if b.Pieces[i].Blocks[j].Complete {
				completed_b += 1
			}
		}
	}

	prog := fmt.Sprintf("(%d/%d)", completed_b, num_b)

	output += "["
	for i, bit := range b.Bitmap() {
		if bit == true {
			output += "1"
		} else {
			output += "0"
		}
		if i > 80 {
			output += fmt.Sprintf("...] Pieces: (%d/%d) Blocks: %s", len(b.Complete()), len(b.Pieces), prog)
			return output
		}
	}
	return output + "]"
}


func (b *Bitmap) New(hashes []byte, piece_len int64, file_len int64) {
	// Description:
	//
	//   method on the Bitmap struct that initializes it
	//
	// Parameters:
	//
	//   hashes    : byte string consisting of the concatenation of all 20-byte SHA1 hash values, one per piece
	//   piece_len : number of bytes in each piece
	//
	// Returns:
	//
	//   nothing
	//
	// Example usage:
	//
	//   var localState State
	//   localState.Bitmap.New([]byte(torrent_file.Info["pieces"].(string)), torrent_file.Info["piece length"].(int64))
	//   fmt.Println(localState.Bitmap)

	if len(hashes) == 0 {
		panic("New(): something went wrong")
	}

	if (len(hashes) % 20) != 0 {
		panic("New(): hash of each piece must be 20 bytes long")
	}

	// Make pieces
	idx := 0
	hash_offset := 0 // how many bytes into hashes we are currently at
	var file_offset int64 = 0 // how many bytes into the file we are currently at
	for hash_offset < len(hashes) {

		// The last piece may be smaller in size
		var min int64
		if file_offset + piece_len <= file_len {
			min = piece_len
		} else {
			min = file_len - file_offset
		}

		// Create a new piece
		var p Piece
		p.New(idx, min, hashes[hash_offset:hash_offset+20], file_offset)

		// Add the piece to the bitmap
		b.Pieces = append(b.Pieces, p)

		// Next piece
		idx++
		hash_offset += 20
		file_offset += p.Length
	}

	b.Pieces_lock = new(sync.RWMutex)
}

func (b *Bitmap) HasPiece(piece_index int) bool {
	// Description:
	//
	//   method on the Bitmap struct that checks if a specific piece is fully downloaded
	//
	// Parameters:
	//
	//   piece_index : an integer representing the index of the piece we are checking for
	//
	// Returns:
	//
	//   true if the piece is fully downloaded, false if the piece is not fully downloaded
	//
	// Example usage:
	//
	//   var localState State
	//   if (!localState.Bitmap.HasPiece(38)) {
	//       fmt.Println("I am missing piece 38")
	//   }

	b.Pieces_lock.Lock()
	defer b.Pieces_lock.Unlock()

	return b.Pieces[piece_index].Complete()
}

func (b *Bitmap) MarkPiece(piece_index int) {
	// Description:
	//
	//   method on the Bitmap struct that marks a specific piece as downloaded/complete
	//
	// Parameters:
	//
	//   piece_index : an integer representing the index of the piece to mark
	//
	// Returns:
	//
	//   nothing
	//
	// Example usage:
	//
	//   var localState State
	//   if (!localState.Bitmap.HasPiece(38)) {
	//       fmt.Println("I am missing piece 38")
	//   }

	b.Pieces_lock.Lock()
	defer b.Pieces_lock.Unlock()

	// fmt.Printf("MarkPiece(%d): %d = %d\n", piece_index, b.Pieces[piece_index].Progress, b.Pieces[piece_index].Length)
	b.Pieces[piece_index].Progress = b.Pieces[piece_index].Length
}

func (b *Bitmap) AllDownloaded() bool {
	// Description:
	//
	//   method on the Bitmap struct that checks if all pieces have been downloaded.
	//
	// Parameters:
	//
	//   none
	//
	// Returns:
	//
	//   true if all pieces are downloaded, false otherwise
	//
	// Example usage:
	//
	//   var localState State
	//   if (localState.Bitmap.AllDownloaded()) {
	//       fmt.Println("I have all the pieces")
	//   }

	b.Pieces_lock.RLock()
	defer b.Pieces_lock.RUnlock()

	for i := 0; i < len(b.Pieces); i++ {
		if b.Pieces[i].Incomplete() {
			return false
		}
	}

	return true
}

func (b *Bitmap) Bitmap() []bool {
	// Description:
	//
	//   method on the Bitmap struct that returns a boolean representation of the Pieces
	//
	// Parameters:
	//
	//   none
	//
	// Returns:
	//
	//   a slice representation of the bitmap
	//
	// Example usage:
	//
	//   var localState State
	//   localState.Bitmap.New(50)
	//   var bitfield []bool := localState.Bitmap.Bitmap()

	b.Pieces_lock.RLock()
	defer b.Pieces_lock.RUnlock()

	var output []bool
	for i := 0; i < len(b.Pieces); i++ {
		if b.Pieces[i].Complete() {
			output = append(output, true)
		} else {
			output = append(output, false)
		}
	}
	return output
}

// Description:
//
//	method on the Bitmap struct that returns a compressed version of the bitmap (each bitmap byte is compressed into a bit)
//	each piece on the compressed bitmap is represented in one bit (high bit of the first byte represents the first piece/0th piece index).
//	the bit that represents a piece is set to 1 if we have that piece, 0 if we don't.
//
// Parameters:
//
//	none
//
// Returns:
//
//	a compressed representation of the bitmap
//
// Example usage:
//
//	var localState State
//	localState.Bitmap.New(50)
//	var bitfield []bool := localState.Bitmap.Bitmap()
func (b *Bitmap) Compress() ([]byte, int) {

	b.Pieces_lock.RLock()
	defer b.Pieces_lock.RUnlock()

	length := len(b.Pieces)

	compressedBytes := int64(math.Ceil(float64(length) / float64(8)))
	compressedBitfield := make([]byte, compressedBytes)
	for i := 0; i < length; i++ {
		if b.Bitmap()[i] {
			placedByte := i / 8
			bitInByte := i - placedByte
			compressedBitfield[placedByte] = compressedBitfield[placedByte] | (1 << (8 - (bitInByte + 1)))
		}
	}
	return compressedBitfield, int(compressedBytes)
}

func (b *Bitmap) HasExtraPieces(b2 *Bitmap) (bool, error) {
	// bitmap := b.Bitmap()
	// b2_bitmap := b.Bitmap()

	// if len(bitmap) != len(b2_bitmap) {
	// 	fmt.Println("HasExtraPieces error: unequal bitmap sizes")
	// 	return false, fmt.Errorf("Unequal bitmap sizes")
	// }
	// bitmapSize := len(b2_bitmap)
	// b.Pieces_lock.RLock()
	// b2.Pieces_lock.RLock()
	// defer b.Pieces_lock.RUnlock()
	// defer b2.Pieces_lock.RUnlock()
	// for i := 0; i < bitmapSize; i++ {
	// 	// fmt.Printf("%t == %t\n", bitmap[i], b2_bitmap[i])
 //
	// 	if bitmap[i] == false && b2_bitmap[i] == true {
	// 		fmt.Println("Peer has pieces that I need")
	// 		return true, nil
	// 	}
	// }
	// return false, nil

	b.Pieces_lock.RLock()
	b2.Pieces_lock.RLock()
	defer b.Pieces_lock.RUnlock()
	defer b2.Pieces_lock.RUnlock()
	for i := 0; i < len(b.Pieces); i++ {
		if b.Pieces[i].PendingIncomplete() && b2.Pieces[i].Complete() {
			return true, nil
		}
	}
	return false, nil
}


func (b *Bitmap) Complete() []Piece {
	// Description:
	//    Returns a list of Pieces that are 100% downloaded
	b.Pieces_lock.RLock()
	defer b.Pieces_lock.RUnlock()

	var output []Piece
	for i := 0; i < len(b.Pieces); i++ {
		if b.Pieces[i].Complete() {
			output = append(output, b.Pieces[i])
		}
	}
	return output
}

func (b *Bitmap) Incomplete() []Piece {
	// Description:
	//    Returns a list of Pieces that are not 100% downloaded
	b.Pieces_lock.RLock()
	defer b.Pieces_lock.RUnlock()

	var output []Piece
	for i := 0; i < len(b.Pieces); i++ {
		if b.Pieces[i].Incomplete() {
			output = append(output, b.Pieces[i])
		}
	}
	return output
}

func (b *Bitmap) PendingComplete() []Piece {
	// Description:
	//    Returns a list of Pieces that will be 100% downloaded,
	//    after all pending requests are completed
	b.Pieces_lock.RLock()
	defer b.Pieces_lock.RUnlock()

	var output []Piece
	for i := 0; i < len(b.Pieces); i++ {
		if b.Pieces[i].PendingComplete() {
			output = append(output, b.Pieces[i])
		}
	}
	return output
}

func (b *Bitmap) PendingIncomplete() []Piece {
	// Description:
	//    Returns a list of Pieces that will not be 100% downloaded,
	//    even after all pending requests are completed
	b.Pieces_lock.RLock()
	defer b.Pieces_lock.RUnlock()

	var output []Piece
	for i := 0; i < len(b.Pieces); i++ {
		if b.Pieces[i].PendingIncomplete() {
			output = append(output, b.Pieces[i])
		}
	}
	return output
}

func (p *Piece) BlocksDone() int {
	count := 0
	// How many blocks of this piece are completed?
	for i := range p.Blocks {
		if p.Blocks[i].Complete {
			count += 1
		}
	}
	return count
}

func (b *Bitmap) InProgress() []Piece {
	// Returns:
	//    a list of Pieces that have at least 1 block downloaded but not all

	b.Pieces_lock.RLock()
	defer b.Pieces_lock.RUnlock()

	var output []Piece
	for i := 0; i < len(b.Pieces); i++ {
		if b.Pieces[i].BlocksDone() > 0 && b.Pieces[i].PendingIncomplete() {
			output = append(output, b.Pieces[i])
		}
	}
	return output
}

func (p *Piece) Complete() bool {
	// Returns true if:
	//   the piece is 100% downloaded
	p.Piece_lock.RLock()
	defer p.Piece_lock.RUnlock()
	return p.Progress >= p.Length
}

func (p *Piece) Incomplete() bool {
	// Returns true if:
	//   the piece is not 100% downloaded
	p.Piece_lock.RLock()
	defer p.Piece_lock.RUnlock()
	return p.Progress < p.Length
}

func (p *Piece) PendingComplete() bool {
	// Returns true if:
	//   the piece will be 100% downloaded,
	//   after all pending requests are completed
	p.Piece_lock.RLock()
	defer p.Piece_lock.RUnlock()
	return p.Pending >= p.Length
}

func (p *Piece) PendingIncomplete() bool {
	// Returns true if:
	//   the piece will not be 100% downloaded,
	//   even after all pending requests are completed
	p.Piece_lock.RLock()
	defer p.Piece_lock.RUnlock()
	return p.Pending < p.Length
}



func (a *Block) Equals(b *Block) bool {
	// Description:
	//
	//    method on the Block struct that returns true if a and b are equal

	if a == nil || b == nil {
		return false
	}

	a.Block_lock.RLock()
	defer a.Block_lock.RUnlock()
	b.Block_lock.RLock()
	defer b.Block_lock.RUnlock()

	return a.PieceIndex == b.PieceIndex && a.Offset == b.Offset && a.Length == b.Length
}

func (b *Block) New(idx int, offset int64, length int64) {
	// Description:
	//
	//   method on the Block struct that initializes it
	//
	// Parameters:
	//
	//   idx    : index of the piece that this block belongs to
	//   offset : how many bytes into the piece
	//   length : how many bytes long this block is
	//
	// Returns:
	//
	//   TODO
	//
	// Example usage:
	//
	//   TODO

	b.PieceIndex = idx
	b.Offset     = offset
	b.Length     = length
	b.Block_lock = new(sync.RWMutex)
	b.Buf        = nil
	b.Complete   = false
	b.Pending    = false
	b.Incomplete = true
}

func (b Block) String() string {
	return fmt.Sprintf("<%d %d %d>", b.PieceIndex, b.Offset, b.Length)
}

func (p *Piece) Reset() {
	for i := range p.Blocks {
		p.Blocks[i].Buf        = nil
		p.Blocks[i].Complete   = false
		p.Blocks[i].Pending    = false
		p.Blocks[i].Incomplete = true
	}
	p.Progress = 0
	p.Pending  = 0
}

func (p *Piece) New(idx int, length int64, hash []byte, file_offset int64) {
	p.Index      = idx
	p.Length     = length
	p.Hash       = hash
	p.FileOffset = file_offset
	p.Piece_lock = new(sync.RWMutex)

	p.Blocks     = nil
	p.Progress   = 0
	p.Pending    = 0


	// Take the length and divide it into block sizes
	var offset int64 = 0
	tlen := p.Length

	for tlen > 0 {
		var b Block
		var curr_b_len int64

		if tlen - 16384 > 0 {
			curr_b_len = 16384
		} else {
			curr_b_len = tlen
		}

		b.New(p.Index, offset, curr_b_len)
		p.Blocks = append(p.Blocks, b)

		offset += curr_b_len
		tlen -= curr_b_len
	}
}
