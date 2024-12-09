package logic

import (
	"client/state"
	"client/peer"
	"math/rand"
	"sort"
	"time"
	"fmt"
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
			unchoke(s.Peers[i])
		} else {
			choke(s.Peers[i])
		}
	}

	// Reset the upload speeds of every peer
	for i := range s.Peers {
		s.Peers[i].Upload_speed = 0
	}
}

func CalculateUploadRate(s *state.State) {
	for _, peer := range s.Peers {
		time_elapsed := time.Now().UnixNano() - peer.Last_time_updated
		peer.Last_time_updated = time.Now().UnixNano()
		if time_elapsed == 0 {
			fmt.Println("Time elapsed is 0")
			peer.Uploaded_bytes = 0
			continue
		}
		upload_speed := float64(peer.Uploaded_bytes) / float64(time_elapsed)
		peer.Upload_speed = upload_speed
		peer.Uploaded_bytes = 0
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
		// Send choke message over the wire
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
		// Send unchoke message over the wire
	}
}

