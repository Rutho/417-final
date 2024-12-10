package bitmap

import (
	"sync"
	"fmt"
	"math"
)

type Bitmap struct {
	// Description:
	//
	//   structure that stores a bitmap representing the download progress of the target file
	//
	// Fields:
	//
	//   bitmap : stores a boolean value for each piece. true = downloaded, false = not downloaded
	//   Length : length of the bitmap
	bitmap_lock *sync.RWMutex
	bitmap      []bool
	Length      int
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

	if b.Length > 100 {
		count_ones := 0
		count_zeros := 0
		for _, bit := range b.bitmap {
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
	for _, bit := range b.bitmap {
		if bit == true {
			output += "1"
		} else {
			output += "0"
		}
	}
	return output + "]\n"
}

func (b *Bitmap) New(length int) {
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

	if length < 1 {
		panic("New(): bitmap length must be positive")
	}

	b.bitmap = make([]bool, length)
	b.bitmap_lock = new(sync.RWMutex)
	b.Length = length
}

func (b *Bitmap) HasPiece(index int) bool {
	// Description:
	//
	//   method on the Bitmap struct that checks if a specific piece is downloaded
	//
	// Parameters:
	//
	//   index : an integer representing the index of the piece we are checking for
	//
	// Returns:
	//
	//   true if the piece is downloaded, false if the piece is not downloaded
	//
	// Example usage:
	//
	//   var localState State
	//   if (!localState.Bitmap.HasPiece(38)) {
	//       fmt.Println("I am missing piece 38")
	//   }
	b.bitmap_lock.RLock()
	defer b.bitmap_lock.RUnlock()
	if index < 0 || index >= len(b.bitmap) {
		panic("HasPiece(): Index out of range")
	}

	return b.bitmap[index]
}

func (b *Bitmap) MarkPiece(index int) {
	// Description:
	//
	//   method on the Bitmap struct that marks a specific piece as downloaded
	//
	// Parameters:
	//
	//   index : an integer representing the index of the piece to mark
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
	if index < 0 || index >= len(b.bitmap) {
		panic("HasPiece(): Index out of range")
	}

	b.bitmap[index] = true
}

func (b *Bitmap) UnmarkPiece(index int) {
	// Description:
	//
	//   method on the Bitmap struct that marks a specific piece as not downloaded
	//
	// Parameters:
	//
	//   index : an integer representing the index of the piece to mark
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
	b.bitmap_lock.Lock()
	defer b.bitmap_lock.Unlock()
	if index < 0 || index >= len(b.bitmap) {
		panic("HasPiece(): Index out of range")
	}

	b.bitmap[index] = false
}

func (b *Bitmap) MissingPieces(totalPieces int) []int {
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
	b.bitmap_lock.RLock()
	defer b.bitmap_lock.RUnlock()
	var missing []int
	for i := 0; i < len(b.bitmap); i++ {
		if !b.HasPiece(i) {
			missing = append(missing, i)
		}
	}
	return missing
}

func (b *Bitmap) IsComplete() bool {
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
	//   if (localState.Bitmap.IsComplete()) {
	//       fmt.Println("I have all the pieces")
	//   }
	b.bitmap_lock.RLock()
	defer b.bitmap_lock.RUnlock()
	for i := 0; i < len(b.bitmap); i++ {
		if !b.HasPiece(i) {
			return false
		}
	}

	return true
}

func (b *Bitmap) Get() []bool {
	// Description:
	//
	//   method on the Bitmap struct
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
	b.bitmap_lock.RLock()
	defer b.bitmap_lock.RUnlock()
	return b.bitmap
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
	b.bitmap_lock.RLock()
	defer b.bitmap_lock.RUnlock()
	compressedBytes := int64(math.Ceil(float64(b.Length) / float64(8)))
	compressedBitfield := make([]byte, compressedBytes)
	for i := 0; i < b.Length; i++ {
		if b.bitmap[i] {
			placedByte := i / 8
			bitInByte := i - placedByte
			compressedBitfield[placedByte] = compressedBitfield[placedByte] | (1 << (8 - (bitInByte + 1)))
		}
	}
	return compressedBitfield, int(compressedBytes)
}
