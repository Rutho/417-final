package receive

import (
	"client/bitmap"
)


// Upon receiving a Piece:
// 1. Remove the corresponding Request from RequestQueue
// 2. Update the Piece


func Process(b *bitmap.Bitmap, piece_index int, piece_offset int, buf []byte) {
	piece := &b.Pieces[piece_index]

	var block *bitmap.Block = nil

	// Identify the corresponding block
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

	// We received the block
	block.Complete = true
	block.Incomplete = false

	piece.Pending -= len(buf)
	piece.Progress += len(buf)
}
