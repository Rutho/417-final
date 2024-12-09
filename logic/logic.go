package logic

import (
	"client/state"
	"client/peer"
	"math/rand"
	"sort"
)

func ChokeAlgorithm(s *state.State) {
	// Remove all peers from the Top4 list
	for i := range s.Peers {
		s.Peers[i].Top4 = false
	}

	// Sort the peer list by upload speed, so that the best peers are first
	sort.Slice(s.Peers, func(i, j int) bool {
		return s.Peers[i].Upload_speed > s.Peers[j].Upload_speed
	})

	// Get the peers which are interested in us
	interested := s.PeerInterested()
	n := len(interested)

	// Select the top 3 peers
	for i := 0; i < n && i < 3; i++ {
		interested[i].Top4 = true
	}

	// Select 1 random peer
	if (n >= 4) {
		interested[rand.Intn(n - 3) + 3].Top4 = true
	}

	// Choke all peers except those in Top4
	for i := range s.Peers {
		if s.Peers[i].Top4 == true {
			unchoke(&s.Peers[i])
		} else {
			choke(&s.Peers[i])
		}
	}

	// Reset each peer
	for i := range s.Peers {
		s.Peers[i].Uploaded_bytes = 0
	}
}

func choke(p *peer.Peer) {
	// Description:
	//
	//   function to choke a peer
	//
	// Parameters:
	//
	//   pointer to the peer to choke
	//
	// Returns:
	//
	//   nothing
	//
	// Example usage:
	//
	//   TODO

	if p.Am_choking == false {
		p.Am_choking = true
		p.Connection.SendChoke()
	}
}

func unchoke(p *peer.Peer) {
	// Description:
	//
	//   function to choke a peer
	//
	// Parameters:
	//
	//   pointer to the peer to choke
	//
	// Returns:
	//
	//   nothing
	//
	// Example usage:
	//
	//   TODO

	if p.Am_choking == true {
		p.Am_choking = false
		p.Connection.SendUnchoke()
	}
}

