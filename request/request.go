package request

import (
	"client/bitmap"
	"client/peer"
)

const MaxBlockSize = 16384 // Maximum block size, in bytes
const QueueSize    = 1     // How many pending requests can be in flight at once

type Request struct {
	// Description:
	//
	//   When we make requests for data from peers, we are actually
	//   requesting a block.
	//
	// Fields:
	//
	//   Block     : the requested block
	//   Peer      : pointer to the peer that the request was sent to
	//   Cancelled : if this request is cancelled. Initially false

	Block      bitmap.Block
	Peer      *peer.Peer
	Cancelled  bool
}

func QueueIsFull(rq []Request) bool {
	// Description:
	//
	//   function to check if the Request Queue is full
	//
	// Parameters:
	//
	//   a pointer to the State struct
	//
	// Returns:
	//
	//   true if full, false otherwise
	//
	// Example usage:
	//
	//   TODO

	return len(rq) >= QueueSize
}
