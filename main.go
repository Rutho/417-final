package main

//TODO: validate handshake we receive (sane info_hash and peer_id)

import (
	"client/comms"
	"client/data"
	"fmt"
	"math/rand"
	"sort"
	"time"
	"client/state"
	"client/peer"
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

	// Initialize the bitmap
	localState.Bitmap.New(50) // TODO placeholder length of 50 pieces

	// Generate an ID
	localState.GenerateID()

	// Store the peer list into the State struct
	for _, p := range *response_info.Peers {
		localState.Peers = append(localState.Peers, p) // deep copy
	}

	// Pick 4 random peers to be the Top4
	random := rand.Perm(len(localState.Peers))
	for i := 0; i < len(localState.Peers) && i < 4; i++ {
		localState.Top4 = append(localState.Top4, &localState.Peers[random[i]])
	}

	// Print the State struct
	fmt.Println(localState)

	fmt.Println(torrent_file.Announce)

	s := comms.NewServer(localState.Port, 2048, string(request_info.Peer_id), request_info.Info_hash)
	go s.Start()
	for _, v := range localState.Peers {
		conn := comms.NewConnectionState(v.Ip, v.Port, v.Peer_id, string(request_info.Peer_id), request_info.Info_hash, nil)
		s.EstablishConnection(conn)
	}

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

	//2048 is a placeholder. Should be substituted by piece length from tracker response

	//This goroutine processes the packets received by the server (our client)
	//Function will be moved to another file
	// go func(pktCh chan comms.Packet) {
	// 	for pkt := range pktCh {
	// 		// msg := message.PacketPayloadToMessage(pkt.Payload)

	// 		// switch msg {
	// 		// case message.KeepAlive:
	// 		// case message.Choke:
	// 		// case message.Unchoke:
	// 		// case message.Interested:
	// 		// case message.NotInterested:
	// 		// case message.Have:
	// 		// case message.Bitfield:
	// 		// case message.Request:
	// 		// case message.Piece:
	// 		// case message.Cancel:
	// 		// case message.Port:
	// 		// //Invalid message
	// 		// default:

	// 		// }

	// 	}
	// }(s.PktCh)
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
		if(time_elapsed == 0) {
			fmt.Println("Time elapsed is 0");
			peer.Uploaded_bytes = 0
			continue;
		}
		upload_speed := float64(peer.Uploaded_bytes) / float64(time_elapsed)
		peer.Upload_speed = upload_speed
		peer.Uploaded_bytes = 0
	}
}
