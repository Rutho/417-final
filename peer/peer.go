package peer

import (
	"client/bitmap"
	"client/comms"
	"fmt"
	"net"
	"strings"
	"sync"
)

type Peers struct {
	// Description:
	//
	//   structure that stores a list of peers
	//
	// Fields:
	//
	//   List : list of peers

	List []Peer
}

type Peer struct {
	// Description:
	//
	//   structure that stores information about other peers
	//
	// Fields:
	//
	//   Ip                : IP address
	//   Port              : port
	//   Am_choking        :
	//   Am_interested     :
	//   Peer_choking      :
	//   Peer_interested   :
	//   Peer_id           : 20-byte string unique ID. Optionally the same peer_id that is transmitted in tracker requests
	//   Uploaded_bytes    : Number of bytes uploaded in the current epoch
	//   Last_time_updated : int64
	//   Upload_speed      : float64
	//   Bitmap            : Bitmap of pieces that peer has

	Ip                   net.IP
	Port                 int
	Am_choking_lock      *sync.RWMutex
	Am_choking           bool
	Am_interested_lock   *sync.RWMutex
	Am_interested        bool
	Peer_choking_lock    *sync.RWMutex
	Peer_choking         bool
	Peer_interested_lock *sync.RWMutex
	Peer_interested      bool
	Peer_id              string
	Uploaded_bytes       int64
	Downloaded_bytes     int64
	Bitmap_lock          *sync.RWMutex
	Bitmap               bitmap.Bitmap
	Top4_lock            *sync.RWMutex
	Connection           *comms.ConnectionState
	New                  bool
}

func (p Peer) String() string {
	// Description:
	//
	//   method on the Peer struct that provides a human-readable string representation
	//
	// Parameters:
	//
	//   none
	//
	// Returns:
	//
	//   A string containing the formatted Peer information
	//
	// Example usage:
	//
	//   var peer Peer
	//   fmt.Println(peer)

	return fmt.Sprintf("%s:%d Am_choking:%t Am_int:%t peer_chok:%t peer_int:%t uploaded_bytes:%d",
		p.Ip.String(), p.Port, p.Am_choking, p.Am_interested, p.Peer_choking, p.Peer_interested, p.Uploaded_bytes)
	//return fmt.Sprintf("%s:%d", p.Ip.String(), p.Port)
}

func (pl *Peers) FindPeerByID(peer_id string) *Peer {
	// Description:
	//
	//   method on the Peers struct to find a peer by ID
	//
	// Parameters:
	//
	//   a string of length 20 chars, representing the peer ID
	//
	// Returns:
	//
	//   a pointer to the peer with the specified ID, or nil if not found
	//
	// Example usage:
	//
	//   var localState State
	//   peer := localState.Peers.FindPeerByID("-RFRA00-WZW3JVR4dIv3")
	//   fmt.Println(peer)

	for i := range pl.List {
		if pl.List[i].Peer_id == peer_id {
			return &pl.List[i]
		}
	}
	return nil
}

func (pl *Peers) FindPeerByAddr(peer_address net.IP, port int) *Peer {
	// Description:
	//
	//   method on the Peers struct to find a peer by IP address and port
	//
	// Parameters:
	//
	//   a string of length 20 chars, representing the peer ID
	//
	// Returns:
	//
	//   a pointer to the peer with the matching addr and port, or nil if not found
	//
	// Example usage:
	//
	//   var localState State
	//   peer := localState.Peers.FindPeerByID("-RFRA00-WZW3JVR4dIv3")
	//   fmt.Println(peer)

	for i := range pl.List {
		if strings.Compare(pl.List[i].Ip.String(), peer_address.String()) == 0 && pl.List[i].Port == port {
			return &pl.List[i]
		}
	}
	return nil
}

func (pl *Peers) PeerInterested() []Peer {
	// Description:
	//
	//   method on the Peers struct to get all the interested peers
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
	//   for _, peer := localState.Peers.PeerInterested() {
	//       process(peer)
	//   }

	var output []Peer
	for i := range pl.List {
		if pl.List[i].Peer_interested == true {
			output = append(output, pl.List[i])
		}
	}
	return output
}

func (pl *Peers) PeerChoking() []Peer {
	// Description:
	//
	//   method on the Peers struct to get all the peers who are choking us
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

	var output []Peer
	for i := range pl.List {
		if pl.List[i].Peer_choking == true {
			output = append(output, pl.List[i])
		}
	}
	return output
}

func (pl *Peers) AmInterested() []Peer {
	// Description:
	//
	//   method on the Peers struct to get all the peers who we are interested in
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

	var output []Peer
	for i := range pl.List {
		if pl.List[i].Am_interested == true {
			output = append(output, pl.List[i])
		}
	}
	return output
}

func (pl *Peers) AmChoking() []Peer {
	// Description:
	//
	//   method on the Peers struct to get all the peers who we are choking
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

	var output []Peer
	for i := range pl.List {
		if pl.List[i].Am_choking == true {
			output = append(output, pl.List[i])
		}
	}
	return output
}

func (pl *Peers) RemovePeerByID(peer_id string) {
	idx := 0
	for p_idx, p := range pl.List {
		if strings.Compare(p.Peer_id, peer_id) == 0 {
			idx = p_idx
		}
	}
	fmt.Println(idx, len(pl.List))
	if idx == 0 {
		return
	}
	if idx == (len(pl.List)) {
		pl.List = pl.List[:len(pl.List)]
		return
	}
	pl.List = append(pl.List[:idx], pl.List[idx+1:]...)
}

// Mutex updates
func (p *Peer) UpdateChoking(c bool) {
	p.Am_choking_lock.Lock()
	defer p.Am_choking_lock.Unlock()
	p.Am_choking = c
}

func (p *Peer) UpdatePeerChoking(c bool) {
	p.Peer_choking_lock.Lock()
	defer p.Peer_choking_lock.Unlock()
	p.Peer_choking = c
}

func (p *Peer) UpdateInterested(c bool) {
	p.Am_interested_lock.Lock()
	defer p.Am_interested_lock.Unlock()
	p.Am_interested = c
}

func (p *Peer) UpdatePeerInterested(c bool) {
	p.Peer_interested_lock.Lock()
	defer p.Peer_interested_lock.Unlock()
	p.Peer_interested = c
}
