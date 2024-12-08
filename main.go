package main

//TODO: validate handshake we receive (sane info_hash and peer_id)

import (
	"client/comms"
	"client/data"
	"client/peer"
	"client/state"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"net"
	"sort"
	"time"
)

func main() {
	// Seed the random number generator
	rand.Seed(time.Now().UTC().UnixNano())

	// Create a State object
	var localState state.State

	// Parse command line arguments
	localState.ParseClargs()

	// --------
	// Get data from the tracker
	// --------
	var torrent_file *data.Torrent_info = new(data.Torrent_info)
	var request_info *data.Tracker_request = new(data.Tracker_request)
	var response_info *data.Tracker_response = new(data.Tracker_response)
	response_info.Peers = new([]peer.Peer)
	data.Set_torrent_info(localState.TorrentFile, torrent_file)
	data.Set_tracker_request(request_info, torrent_file)
	fmt.Println(torrent_file.Announce)
	request_info.Port = localState.Port
	request_info.Compact = 1
	request_info.Numwant = 30
	request_info.Key = "wow"
	torrent_file.Tracker_Type = data.HTTP
	data.Send_get(request_info, torrent_file, response_info)

	// --------
	// Initialize State
	// --------

	// Initialize the IP address
	localState.IP = net.ParseIP("127.0.0.1")

	// Initialize the bitmap
	numberOfpieces := int(math.Ceil(float64(len(torrent_file.Info["pieces"].(string))) / float64(20)))

	localState.Bitmap.New(numberOfpieces)

	// Generate an ID
	localState.GenerateID()

	// Store the peer list into the State struct
	// --------

	s := comms.NewServer(localState.Port, 2048, string(request_info.Peer_id), request_info.Info_hash)
	go s.Start()

	establishCh := make(chan peer.Peer, 60)
	//Message Processing
	go func(msgCh chan comms.Message) {
		for {
			select {
			case msg := <-msgCh:
				peer := localState.FindPeer(msg.PeerId)
				fmt.Printf("msg type: %d\n", msg.MsgType)
				switch msg.MsgType {
				case comms.Bitfield:
				case comms.Choke:
					peer.Peer_choking = true
				case comms.Unchoke:
					peer.Peer_choking = false
				case comms.Interested:
					peer.Peer_interested = true
				case comms.NotInterested:
					peer.Peer_interested = false
				case comms.Piece:
					//Piece is a little complicated since we have to
					//consider the possibility of cancels
				case comms.Cancel:
				case comms.Port:
				case comms.KeepAlive:
				case comms.Request:
				case comms.Have:
					piece_index := binary.BigEndian.Uint32(msg.Payload)
					//casting may be problematic if # of pieces exceed INT_MAX
					peer.Bitmap.MarkPiece(int(piece_index))
				}
			case connectedPeer := <-establishCh:
				localState.Peers = append(localState.Peers, connectedPeer) //deep copy
			}
		}
	}(s.MsgCh)

	//Actively establish connections to peers

	for _, p := range *response_info.Peers {
		go func(peer peer.Peer) {
			conn := comms.NewConnectionState(peer.Ip, peer.Port, peer.Peer_id, string(request_info.Peer_id),
				request_info.Info_hash, nil)
			err := s.EstablishConnection(conn)
			if err == nil {
				peer.Bitmap.New(numberOfpieces)
				establishCh <- peer
			}
		}(p)
	}

	time.Sleep(10000000000)
	// --------
	// Pick 4 random peers to be the Top4
	random := rand.Perm(len(localState.Peers))
	for i := 0; i < len(localState.Peers) && i < 4; i++ {
		localState.Top4 = append(localState.Top4, &localState.Peers[random[i]])
	}

	// Print the State struct
	fmt.Println(localState)

	fmt.Println(torrent_file.Announce)

	fmt.Println("Hello!!!")

	/*
		Call Optimistic Unchoking every 30 seconds.
		Calculate the upload rate of every peer every 10 seconds
		Will need to add feature to have this run continously
	*/
	time.AfterFunc(30*time.Second, func() {
		fmt.Println("OptimisticUnchoking()")
		OptimisticUnchoking(&localState)
	})

	time.AfterFunc(10*time.Second, func() {
		fmt.Println("CalculateUploadRate()")
		CalculateUploadRate(&localState)
	})
}

func OptimisticUnchoking(localState *state.State) {
	if len(localState.Top4) == 4 && len(localState.Peers) > 4 {
		// Update the top 4 peers. Latest unchoked peer may be a replacement
		sort.Slice(localState.Peers, func(i, j int) bool {
			return (localState.Peers)[i].Upload_speed < (localState.Peers)[j].Upload_speed
		})
		// Top4 stores pointers to Peer
		localState.Top4[0] = &localState.Peers[0]
		localState.Top4[1] = &localState.Peers[1]
		localState.Top4[2] = &localState.Peers[2]
		localState.Top4[3] = &localState.Peers[3]

		// Check for any unchoked peers and choke them. Also generate a list of interested peers
		var interested_peers []*peer.Peer
		for _, peer := range localState.Peers {
			if !contains(localState.Top4, &peer) {
				if !peer.Am_choking {
					peer.Am_choking = true
				}
				if peer.Am_interested {
					interested_peers = append(interested_peers, &peer)
				}
			}
		}
		if len(interested_peers) < 1 {
			fmt.Println("No interested peers to unchoke")
			return
		}

		// Randomly unchoke a interested peer
		interested_peers[rand.Intn(len(interested_peers))].Am_choking = false
	}
}

func contains(top_four []*peer.Peer, p *peer.Peer) bool {
	for _, v := range top_four {
		if v.Ip.Equal(p.Ip) && v.Port == p.Port {
			return true
		}
	}
	return false
}

func CalculateUploadRate(localState *state.State) {
	for _, peer := range localState.Peers {
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
