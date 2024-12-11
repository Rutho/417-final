package bitmap

import (
	"fmt"
	"math"
	"sync"
)

type Bitmap struct {
	// Description:
	//
	//   structure that stores a bitmap representing the download progress of the target file
	//
	// Fields:
	//
	//   TODO

	Pieces_lock *sync.RWMutex
	Pieces      []Piece
}

type Piece struct {
	// Description:
	//
	//   TODO
	//
	// Fields:
	//
	//   TODO

	Index      int
	Length     int64
	Hash       []byte
	Blocks     []Block
	Progress   int
	Complete   bool
	Incomplete bool
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
	//   Offset     : the byte offset into the piece
	//   Length     : number of bytes this block is

	PieceIndex int
	Offset     int
	Length     int
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

	if len(b.Pieces) > 100 {
		count_ones := 0
		count_zeros := 0
		for _, bit := range b.Get() {
			if bit == true {
				count_ones++
			} else {
				count_zeros++
			}
		}
		output += fmt.Sprintf("Omitted due to length. Number of 1s = %d, Number of 0s = %d\n", count_ones, count_zeros)
		return output
	}

	output += "["
	for _, bit := range b.Get() {
		if bit == true {
			output += "1"
		} else {
			output += "0"
		}
	}
	return output + "]\n"
}

func (b *Bitmap) New(hashes []byte, piece_len int64) {
	// Description:
	//
	//   method on the Bitmap struct that initializes it
	//
	// Parameters:
	//
	//   length : an integer representing the length to initialize the bitmap with
	//
	// Returns:
	//
	//   nothing
	//
	// Example usage:
	//
	//   var localState State
	//   localState.Bitmap.New(50)
	//   fmt.Println(localState.Bitmap)

	if len(hashes) == 0 {
		panic("New(): something went wrong")
	}

	if (len(hashes) % 20) != 0 {
		panic("New(): hash of each piece must be 20 bytes long")
	}

	// Split the concatenated hash values into pieces
	idx := 0
	offset := 0
	for offset < len(hashes) {
		p := new(Piece)
		p.Index      = idx
		p.Length     = piece_len
		p.Hash       = hashes[offset:offset+20]
		p.Complete   = false
		p.Incomplete = true

		b.Pieces = append(b.Pieces, *p)

		idx++
		offset += 20
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

	return b.Pieces[piece_index].Complete
}

func (b *Bitmap) MarkPiece(piece_index int) {
	// Description:
	//
	//   method on the Bitmap struct that marks a specific piece as downloaded
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

	b.Pieces[piece_index].Complete = true
	b.Pieces[piece_index].Incomplete = false
}

func (b *Bitmap) MissingPieces() []Piece {
	// Description:
	//
	//   method on the Bitmap struct that returns which pieces are not yet downloaded
	//
	// Parameters:
	//
	//   none
	//
	// Returns:
	//
	//   a list of piece indices that are not yet downloaded
	//
	// Example usage:
	//
	//   var localState State
	//   fmt.Println("I need:")
	//   for _, piece := range localState.Bitmap.MissingPieces() {
	//       fmt.Println(piece)
	//  }

	b.Pieces_lock.RLock()
	defer b.Pieces_lock.RUnlock()

	var output []Piece
	for i := 0; i < len(b.Pieces); i++ {
		if b.Pieces[i].Incomplete {
			output = append(output, b.Pieces[i])
		}
	}
	return output
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
		if b.Pieces[i].Incomplete {
			return false
		}
	}
	return true
}

func (b *Bitmap) Get() []bool {
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
	//   var bitfield []bool := localState.Bitmap.Get()

	b.Pieces_lock.RLock()
	defer b.Pieces_lock.RUnlock()

	var output []bool
	for i := 0; i < len(b.Pieces); i++ {
		if b.Pieces[i].Complete {
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
//	var bitfield []bool := localState.Bitmap.Get()
func (b *Bitmap) Compress() ([]byte, int) {

	b.Pieces_lock.RLock()
	defer b.Pieces_lock.RUnlock()

	length := len(b.Pieces)

	compressedBytes := int64(math.Ceil(float64(length) / float64(8)))
	compressedBitfield := make([]byte, compressedBytes)
	for i := 0; i < length; i++ {
		if b.Get()[i] {
			placedByte := i / 8
			bitInByte := i - placedByte
			compressedBitfield[placedByte] = compressedBitfield[placedByte] | (1 << (8 - (bitInByte + 1)))
		}
	}
	return compressedBitfield, int(compressedBytes)
}

func (b *Bitmap) HasExtraPieces(b2 *Bitmap) (bool, error) {
	bitmap := b.Get()
	b2_bitmap := b.Get()

	if len(bitmap) != len(b2_bitmap) {
		fmt.Printf("HasExtraPieces error: unequal bitmap sizes")
		return false, fmt.Errorf("Unequal bitmap sizes")
	}
	bitmapSize := len(b2_bitmap)
	b.Pieces_lock.RLock()
	b2.Pieces_lock.RLock()
	defer b.Pieces_lock.RUnlock()
	defer b2.Pieces_lock.RUnlock()
	for i := 0; i < bitmapSize; i++ {
		if bitmap[i] == false && b2_bitmap[i] == true {
			return true, nil
		}
	}
	return false, nil
}

