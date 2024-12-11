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
	//   Peers   : all the peers known to the local client (excluding self)
	//   Top4    : pointers to the top4 peers
	//   Bitmap  : represents which pieces of the file being downloaded have been received
	//   BitmapLock : TODO
	//   Cycle  : control variable for usage in Choke Algorithm
	//   RequestQueue : queue storing pending outgoing requests

	TorrentFile  string
	IP           net.IP
	Port         int
	ID           string
	PeersLock    sync.RWMutex
	Top4Lock     sync.RWMutex
	Peers        []peer.Peer
	PeerList     peer.PeerList
	Top4         []*peer.Peer
	BitmapLock   sync.RWMutex
	Bitmap       bitmap.Bitmap
	RequestQueue []request.Request
	PieceHashes  []byte
	PieceLength  int64
	FileSize     int64
	NumPieces    int
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

	output += " |  PeerList    : ["
	for i, peer := range s.PeerList.List {
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

func (s *State) FindPeer(peerId string) *peer.Peer {
	// Description:
	//
	//   method on the State struct to find a peer by ID
	//
	// Parameters:
	//
	//   a string of length 20 chars, representing the ID
	//
	// Returns:
	//
	//   a pointer to the peer with the specified ID, or nil if not found
	//
	// Example usage:
	//
	//   var localState State
	//   peer := localState.FindPeer("-RFRA00-WZW3JVR4dIv3")
	//   fmt.Println(peer)

	s.PeersLock.RLock()
	defer s.PeersLock.RUnlock()

	for i := range s.Peers {
		if s.Peers[i].Peer_id == peerId {
			return &s.Peers[i]
		}
	}
	return nil
}

func (s *State) FindPeerByAddr(peerAddress net.IP, port int) *peer.Peer {
	// Description:
	//
	//   method on the State struct to find a peer by ID
	//
	// Parameters:
	//
	//   a net.IP representing the peer's ip address
	//	 an int representing the peer's port
	//
	// Returns:
	//
	//   a pointer to the peer with the specified ID, or nil if not found
	//
	// Example usage:
	//
	//   var localState State
	//   peer := localState.FindPeer("-RFRA00-WZW3JVR4dIv3")
	//   fmt.Println(peer)

	s.PeersLock.RLock()
	defer s.PeersLock.RUnlock()

	return s.PeerList.FindPeerByAddr(peerAddress, port)
}

func (s *State) PeerInterested() []peer.Peer {
	// Description:
	//
	//   method on the State struct to get all the interested peers
	//
	// Parameters:
	//
	//   none
	//
	// Returns:
	//
	//   a slice of peers who are interested in the local client
	//
	// Example usage:
	//
	//   var localState State
	//   for _, peer := localState.PeerInterested() {
	//       process(peer)
	//   }

	s.PeersLock.RLock()
	defer s.PeersLock.RUnlock()

	return s.PeerList.PeerInterested()
}

func (s *State) PeerChoking() []peer.Peer {
	// Description:
	//
	//   method on the State struct to get all the peers who are choking us
	//
	// Parameters:
	//
	//   none
	//
	// Returns:
	//
	//   a slice of peers who are choking the local client
	//
	// Example usage:
	//
	//   var localState State
	//   for _, peer := localState.PeerChoking() {
	//       process(peer)
	//   }

	s.PeersLock.RLock()
	defer s.PeersLock.RUnlock()

	return s.PeerList.PeerChoking()
}

func (s *State) AmInterested() []peer.Peer {
	// Description:
	//
	//   method on the State struct to get all the peers who we are interested in
	//
	// Parameters:
	//
	//   none
	//
	// Returns:
	//
	//   a slice of peers who the local client is interested in
	//
	// Example usage:
	//
	//   var localState State
	//   for _, peer := localState.AmInterested() {
	//       process(peer)
	//   }

	s.PeersLock.RLock()
	defer s.PeersLock.RUnlock()

	return s.PeerList.AmInterested()
}

func (s *State) AmChoking() []peer.Peer {
	// Description:
	//
	//   method on the State struct to get all the peers who we are choking
	//
	// Parameters:
	//
	//   none
	//
	// Returns:
	//
	//   a slice of peers who the local client is choking
	//
	// Example usage:
	//
	//   var localState State
	//   for _, peer := localState.AmChoking() {
	//       process(peer)
	//   }

	s.PeersLock.RLock()
	defer s.PeersLock.RUnlock()

	return s.PeerList.AmChoking()
}

func (s *State) RemovePeer(peer_id string) {
	s.PeersLock.Lock()
	defer s.PeersLock.Unlock()

	s.PeerList.RemovePeerByID(peer_id)
}

func (s *State) SendARequest() {
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
	//   nothing
	//
	// Example usage:
	//
	//   TODO

	peer := s.SelectPeer()

	if peer == nil {
		return
	}

	// piece_index := s.SelectPiece(peer)


	// TODO
	// peer.Connection.SendRequest(idx, offset, length)
}

func (s *State) SelectPiece(p *peer.Peer) int {
	// Description:
	//
	//   method on the State struct to select a piece to be requested from a peer
	//
	// Parameters:
	//
	//   a peer
	//
	// Returns:
	//
	//   a piece index
	//
	// Example usage:
	//
	//   TODO

	if p.Am_interested == false {
		panic("state.go: SelectPiece() should only be called on peers that the local client is interested in!")
	}

	// Get the pieces that the peer has
	their_bitmap := p.Bitmap.Complete()

	// Get the pieces that the local client is missing
	my_bitmap    := s.Bitmap.Incomplete()

	// Get the intersection of each list
	var intersection []int
	for _, piece := range my_bitmap {
		// Is piece contained in their_bitmap?
		for i := range their_bitmap {
			if their_bitmap[i].Index == piece.Index {
				intersection = append(intersection, piece.Index)
			}
		}
	}

	if len(intersection) == 0 {
		panic("state.go: SendARequest(): We shouldn't be interested in this peer?!?")
	}

	// Pick a random piece
	return intersection[rand.Intn(len(intersection))]
}

func (s *State) SelectPeer() *peer.Peer {
	// Description:
	//
	//   method on the State struct to select a peer to send a request to
	//
	// Parameters:
	//
	//   none
	//
	// Returns:
	//
	//   a pointer to the peer, or nil upon error
	//
	// Example usage:
	//
	//   var s State
	//   peer := s.SelectPeer()

	if request.QueueIsFull(s.RequestQueue) {
		return nil
	}

	interested_peers := s.PeerInterested()

	if len(interested_peers) == 0 {
		return nil
	}

	// Pick a random peer
	rand_idx := rand.Intn(len(interested_peers))

	return &interested_peers[rand_idx]
}



