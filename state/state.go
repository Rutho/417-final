package state



import (
    "fmt"         // https://pkg.go.dev/fmt
    "flag"        // https://pkg.go.dev/flag
    "client/bitmap"
)



type Peer struct {
	// Description:
	//
	//   structure that stores information about other peers
	//
	// Fields:
	//
	//   ip              : ip address
	//   port            : port
	//   Am_choking      : TODO
	//   Am_interested   : TODO
	//   Peer_choking    : TODO
	//   Peer_interested : TODO

	ip              string
	port            int
	Am_choking      bool
	Am_interested   bool
	Peer_choking    bool
	Peer_interested bool
}



type State struct {
    // Description:
    //
    //   structure that stores local state information
    //
    // Fields:
    //
    //   TorrentFile  : name of torrent file in the current directory
    //   Port   : port that this client will run on
    //   Peers  : all the peers known to the local client (excluding self)
    //   Top4   : pointers to the top4 peers
    //   bitmap : represents which pieces of the file being downloaded have been received

    TorrentFile   string
    Port          int
    Peers         []Peer
    Top4          []*Peer
    Bitmap        bitmap.Bitmap
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
    output += fmt.Sprintf(" |  Port        : %d\n", s.Port)

    output += " |  Peers       : ["
    for i, peer := range s.Peers {
        if (i == 0) {
            output += fmt.Sprintf("%s", peer)
        } else {
            output += fmt.Sprintf(", %s", peer)
        }
    }
    output += "]\n"

    output += " |  Top4        : ["
    for i, peer := range s.Top4 {
        if (i == 0) {
            output += fmt.Sprintf("%s", peer)
        } else {
            output += fmt.Sprintf(", %s", peer)
        }
    }
    output += "]\n"

    output += " |  Bitmap      : " + fmt.Sprintf(s.Bitmap.String())

    return output
}



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
//
func (s *State) ParseClargs() {
    // Define -t <torrent file>
    // Default value "debian-12.8.0-amd64-DVD-1.iso.torrent"
    s.TorrentFile = *flag.String("t", "debian-12.8.0-amd64-DVD-1.iso.torrent", "filepath/name of torrent file")

    // Define -p <port number>
    // Default value 8080 (BitTorrent typically uses 6881-6889, and 6969 for tracker)
    s.Port = *flag.Int("p", 8080, "port of host")

    // Parse the command line into the defined flags
    flag.Parse()
}





