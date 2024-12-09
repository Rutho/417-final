package peer

import (
	"client/bitmap"
	"fmt"
	"net"
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
	//   Uploaded_bytes    : int64
	//   Last_time_updated : int64
	//   Upload_speed      : float64
	//   Bitmap            : Bitmap of pieces that peer has

	Ip                net.IP
	Port              int
	Am_choking        bool
	Am_interested     bool
	Peer_choking      bool
	Peer_interested   bool
	Peer_id           string
	Uploaded_bytes    int64
	Last_time_updated int64
	Upload_speed      float64
	Bitmap            bitmap.Bitmap
	InTop4            bool
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
