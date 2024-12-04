package main

import (
    "fmt"         // https://pkg.go.dev/fmt
    "flag"        // https://pkg.go.dev/flag
    "client/data"
    "crypto/sha1" // https://pkg.go.dev/crypto/sha1
)

// Structure to store the parsed command-line arguments
type Clargs struct {
    TorrentFile string
    HostPort    int
}

// Structure to store the local state
type State struct {
    clargs Clargs
    peers  []data.Peer
    top4   []data.Peer
    bitmap []byte
}

func (state State) String() string {
    var output string

    output = "State:\n"
    output += fmt.Sprintf(" |  clarg tfile : %s\n", state.clargs.TorrentFile)
    output += fmt.Sprintf(" |  clarg port  : %d\n", state.clargs.HostPort)

    output += fmt.Sprintf(" |  peers       : [")
    for i, peer := range state.peers {
        if (i == 0) {
            output += fmt.Sprintf("%s", peer)
        } else {
            output += fmt.Sprintf(", %s", peer)
        }
    }
    output += fmt.Sprintf("]\n")

    output += fmt.Sprintf(" |  top4        : [")
    for i, peer := range state.top4 {
        if (i == 0) {
            output += fmt.Sprintf("%s", peer)
        } else {
            output += fmt.Sprintf(", %s", peer)
        }
    }
    output += fmt.Sprintf("]\n")

    output += fmt.Sprintf(" |  bitmap      : [")
    for _, bit := range state.bitmap {
        output += fmt.Sprintf("%s", string(bit))
    }
    output += fmt.Sprintf("]\n")

    return output
}

// Parses the command line arguments into a Clargs struct
func parse_clargs() Clargs {
    // Define -t <torrent file>
    file_p := flag.String("t", "no_torrent", "filepath/name of torrent file")

    // Define -p <port number>
    host_port := flag.Int("p", 8080, "port of host")

    // Parse the command line into the defined flags
    flag.Parse()

    return Clargs{
        TorrentFile: *file_p,
        HostPort:    *host_port,
    }
}

func init_bitmap(state State) {
    return
}

func main() {
    var state State
    state.clargs = parse_clargs()

    fmt.Println(state)

    hash := sha1.Sum([]byte("Hello"))
    fmt.Printf("% x\n", hash)
}


function ChokeAlgorithm():
    while true:
        // Choose which peers to unchoke and choke
        selectPeersToUnchoke()  // Select a subset of peers to unchoke
        chokeAllOtherPeers()     // Choke all peers not in the unchoke subset

        // Wait for a short interval before repeating the process
        wait(interval)

function selectPeersToUnchoke():
    // Determine which peers to unchoke based on criteria such as download/upload speed,
    // connection quality, and reciprocity (uploading back to us)
    sortPeersByUploadSpeed()
    unchokeTop3Peers()

function unchokeTop4Peers():
    // Unchoke the top 4 peers
    for each peer in top4Peers:
        unchoke(peer)

function chokeAllOtherPeers():
    // Choke all peers except those that were selected to be unchoked
    for each peer in allPeers:
        if peer not in unchokedPeers:
            choke(peer)

function choke(peer):
    // Send a choke message to the specified peer, indicating that we will not accept
    // incoming requests for downloading from them
    sendChokeMessage(peer)

function unchoke(peer):
    // Send an unchoke message to the specified peer, indicating that we will accept
    // incoming requests for downloading from them
    sendUnchokeMessage(peer)






