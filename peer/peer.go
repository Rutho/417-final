package peer

import (
	"client/bitmap"
	"fmt"
	"net"
	"sync"
)

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
	Last_time_updated    int64
	Upload_speed         float64
	Bitmap_lock          *sync.RWMutex
	Bitmap               bitmap.Bitmap
	Top4_lock            *sync.RWMutex
	Top4                 bool
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

	return fmt.Sprintf("%s:%d", p.Ip.String(), p.Port)
}

// Mutex updates
func (p *Peer) UpdateChoking(c bool) {
	p.Am_choking_lock.Lock()
	defer p.Am_choking_lock.Unlock()
	p.Am_choking = c
}

func (p *Peer) UpdatePeerChoking(c bool) {
	if p.Peer_choking_lock == nil {
		fmt.Printf("bruh\n")
	}
	p.Peer_choking_lock.Lock()
	defer p.Peer_choking_lock.Unlock()
	p.Peer_choking = c
}

func (p *Peer) UpdateInterested(c bool) {
	p.Am_interested_lock.Lock()
	defer p.Am_interested_lock.Unlock()
	p.Peer_choking = c
}

func (p *Peer) UpdatePeerInterested(c bool) {
	p.Peer_interested_lock.Lock()
	defer p.Peer_interested_lock.Unlock()
	p.Peer_interested = c
}

func (p *Peer) UpdateTop4(c bool) {
	p.Top4_lock.Lock()
	defer p.Top4_lock.Unlock()
	p.Top4 = c
}
