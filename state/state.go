package state

import (
	"client/bitmap"
	"client/peer"
	"client/request"
	"flag" // https://pkg.go.dev/flag
	"fmt"  // https://pkg.go.dev/fmt
	"math/rand"
	"net"
	"sync"
	"time"
)

type State struct {
	// Description:
	//
	//   structure that stores local state information
	//
	// Fields:
	//
	//   TorrentFile  : name of torrent file in the current directory
	//   IP      : IP address of the local machine
	//   Port    : port that this client will run on
	//   ID      : unique 20-byte string identifier for use in tracker requests and peer handshakes.
	//   Peers : all the peers known to the local client (excluding self)
	//   Top4    : pointers to the top4 peers
	//   Bitmap  : represents which pieces of the file being downloaded have been received
	//   RequestQueue : queue storing pending outgoing requests
	//   FileSize     : file size in bytes, assuming single file mode
	//   InfoHash     : 20-byte SHA1 hash of the info key in the torrent metainfo file. This is the same info_hash that is transmitted in tracker requests.

	TorrentFile  string
	IP           net.IP
	Port         int
	ID           string
	PeersLock    sync.RWMutex
	Peers        peer.Peers
	Top4Lock     sync.RWMutex
	Top4         []*peer.Peer
	BitmapLock   sync.RWMutex
	Bitmap       bitmap.Bitmap
	RequestLock  sync.RWMutex
	RequestQueue []request.Request
	PieceHashes  []byte
	PieceLength  int64
	FileSize     int64
	NumPieces    int
	Info_hash    []byte
	Optimistic   *peer.Peer
}

func (s State) String() string {
	// Description:
	//
	//   method on the State struct that provides a human-readable string representation
	//
	// Parameters:
	//
	//   none
	//
	// Returns:
	//
	//   A string containing the formatted State information
	//
	// Example usage:
	//
	//   var localState State
	//   localState.ParseClargs()
	//   fmt.Println(localState)

	var output string = "State:\n"
	output += fmt.Sprintf(" |  TorrentFile : %s\n", s.TorrentFile)
	output += fmt.Sprintf(" |  IP          : %s\n", s.IP.String())
	output += fmt.Sprintf(" |  Port        : %d\n", s.Port)
	output += fmt.Sprintf(" |  ID          : %s\n", s.ID)
	output += fmt.Sprintf(" |  FileSize    : %d\n", s.FileSize)
	output += fmt.Sprintf(" |  NumPieces   : %d\n", s.NumPieces)

	output += " |  Peers       : ["
	for i, peer := range s.Peers.List {
		if i == 0 {
			output += fmt.Sprintf("%s", peer)
		} else {
			output += fmt.Sprintf(",\n |                 %s", peer)
		}
	}
	output += "]\n"

	output += " |  Top4        : ["
	for i, peer := range s.Top4 {
		if i == 0 {
			output += fmt.Sprintf("%s", peer)
		} else {
			output += fmt.Sprintf(",\n |                 %s", peer)
		}
	}
	output += "]\n"

	output += " |  Bitmap      : " + fmt.Sprintf(s.Bitmap.String())
	output += "\n"

	return output
}

func (s *State) ParseClargs() {
	// Description:
	//
	//   method on the State struct to parse command-line arguments
	//
	// Parameters:
	//
	//   none
	//
	// Returns:
	//
	//   nothing
	//
	// Example usage:
	//
	//   var localState State
	//   localState.ParseClargs()
	//   fmt.Println(localState)

	// Define -t <torrent file>
	// Default value "debian-12.8.0-amd64-DVD-1.iso.torrent"
	t := flag.String("t", "debian-12.8.0-amd64-DVD-1.iso.torrent", "filepath/name of torrent file")

	// Define -p <port number>
	// Default value 8080 (BitTorrent typically uses 6881-6889, and 6969 for tracker)
	s.Port = *flag.Int("p", 8080, "port of host")

	// Parse the command line into the defined flags
	flag.Parse()
	s.TorrentFile = *t
}

func (s *State) GenerateID() string {
	// Description:
	//
	//   method on the State struct to generate a unique 20-byte ID
	//
	// Parameters:
	//
	//   none
	//
	// Returns:
	//
	//   a string of length 20 chars, representing the ID
	//
	// Example usage:
	//
	//   var localState State
	//   localState.GenerateID()
	//   fmt.Println(localState)

	// RFRA : name of our BitTorrent client
	// 00   : version number of our BitTorrent client
	s.ID = "-RFRA00-"

	// Random
	chars := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

	for i := len(s.ID); i < 20; i++ {
		s.ID = s.ID + string(chars[rand.Intn(len(chars))])
	}

	return s.ID
}

func (s *State) FindPeerByAddr(peerAddress net.IP, port int) *peer.Peer {
	s.PeersLock.RLock()
	defer s.PeersLock.RUnlock()
	return s.Peers.FindPeerByAddr(peerAddress, port)
}

func (s *State) PeerInterested() []peer.Peer {
	s.PeersLock.RLock()
	defer s.PeersLock.RUnlock()
	return s.Peers.PeerInterested()
}

func (s *State) PeerChoking() []peer.Peer {
	s.PeersLock.RLock()
	defer s.PeersLock.RUnlock()
	return s.Peers.PeerChoking()
}

func (s *State) AmInterested() []peer.Peer {
	s.PeersLock.RLock()
	defer s.PeersLock.RUnlock()
	return s.Peers.AmInterested()
}

func (s *State) AmChoking() []peer.Peer {
	s.PeersLock.RLock()
	defer s.PeersLock.RUnlock()
	return s.Peers.AmChoking()
}

func (s *State) RemovePeer(peer_id string) {
	s.PeersLock.Lock()
	defer s.PeersLock.Unlock()
	s.Peers.RemovePeerByID(peer_id)
}

func (s *State) SendARequest() bool {
	// Description:
	//
	//   method on the State struct to send a request to a peer for a piece.
	//   Is not guaranteed to send a request.
	//
	// Parameters:
	//
	//   none
	//
	// Returns:
	//
	//   true if no more requests should be sent
	//
	// Example usage:
	//
	//   localState.SendARequest()

	s.PeersLock.Lock()
	s.BitmapLock.Lock()

	defer s.PeersLock.Unlock()
	defer s.BitmapLock.Unlock()

	if s.Bitmap.AllDownloaded() {
		fmt.Println("Tried to SendARequest() but all pieces are downloaded")
		return true
	}

	if len(s.RequestQueue) >= request.QueueSize {
		fmt.Println("Tried to SendARequest() but the queue was full")
		return false
	}

	peer := request.SelectPeer(&s.Peers)

	if peer == nil {
		fmt.Println("Tried to SendARequest() but no peer was found!")
		return false
	}

	piece_index := request.SelectPiece(peer, &s.Bitmap)

	// The last block may be smaller in size
	var min int64
	if request.MaxBlockSize < s.Bitmap.Pieces[piece_index].Length-s.Bitmap.Pieces[piece_index].Pending {
		min = request.MaxBlockSize
	} else {
		min = s.Bitmap.Pieces[piece_index].Length - s.Bitmap.Pieces[piece_index].Pending
	}

	if min == 0 {
		panic("Can't request a block of size zero!")
	}

	// Make a new block
	var b bitmap.Block
	b.New(piece_index, s.Bitmap.Pieces[piece_index].Pending, min)

	// Add the block to the piece
	s.Bitmap.Pieces[piece_index].Blocks = append(s.Bitmap.Pieces[piece_index].Blocks, b)

	// Increase pending bytes
	s.Bitmap.Pieces[piece_index].Pending += b.Length

	// Make a new request
	r := new(request.Request)
	r.Block = &s.Bitmap.Pieces[piece_index].Blocks[len(s.Bitmap.Pieces[piece_index].Blocks)-1]
	r.Peer = peer
	r.Cancelled = false
	r.Timestamp = time.Now().Unix()
	r.Request_lock = new(sync.RWMutex)

	// Add the request to the queue
	s.RequestQueue = append(s.RequestQueue, *r)

	peer.Connection.SendRequest(uint32(piece_index), uint32(b.Offset), uint32(b.Length))

	// fmt.Printf("REQUEST %d\n", piece_index)
	// fmt.Printf("REQUEST %s\n", r.Block)
	return false
}

func (s State) PrintTop4() {
	// Description:
	//
	//   method on the State struct that provides a human-readable string representation
	//
	// Parameters:
	//
	//   none
	//
	// Returns:
	//
	//   A string containing the formatted State information
	//
	// Example usage:
	//
	//   var localState State
	//   localState.ParseClargs()
	//   fmt.Println(localState)

	var output string = "Top4:\n["

	for i, peer := range s.Top4 {
		if i == 0 {
			output += fmt.Sprintf("%s", peer)
		} else {
			output += fmt.Sprintf(",\n %s", peer)
		}
	}
	output += "]\n"

	fmt.Println(output)
}
