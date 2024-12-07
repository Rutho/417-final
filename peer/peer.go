package peer



import (
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
    //   TODO

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
