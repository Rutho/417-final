package main

//TODO: validate handshake we receive (sane info_hash and peer_id)

import (
	"client/comms"
	"client/data"
	"flag"
	"fmt"
	"math/rand"
	"sort"
	"time"
)

type LocalPeer struct {
	peers    *[]data.Peer
	top_four *[]data.Peer
}

func main() {
	// Instance of LocalPeer representing our current peer
	var client LocalPeer

	file_p := flag.String("t", "debian-12.8.0-amd64-DVD-1.iso.torrent", "filepath/name of torrent")
	usr_port := flag.Int("p", 8080, "port of client")
	flag.Parse()
	fmt.Println(*file_p, *usr_port)
	var torrent_file *data.Torrent_info = new(data.Torrent_info)
	var request_info *data.Tracker_request = new(data.Tracker_request)
	var response_info *data.Tracker_response = new(data.Tracker_response)
	var peer_list *[]data.Peer = new([]data.Peer)
	response_info.Peers = peer_list
	data.Set_torrent_info(*file_p, torrent_file)
	data.Set_tracker_request(request_info, torrent_file)
	fmt.Println(torrent_file.Announce)
	request_info.Port = *usr_port
	request_info.Compact = 1
	request_info.Numwant = 30
	request_info.Key = "wow"
	torrent_file.Tracker_Type = data.HTTP

	// set the fields of our client
	client.peers = response_info.Peers
	// TODO[Anthony]: set the top 4 peers field

	data.Send_get(request_info, torrent_file, response_info)
	//fmt.Println(request_info, response_info)
	fmt.Println(torrent_file.Announce)

	s := comms.NewServer(*usr_port, 2048, string(request_info.Peer_id), request_info.Info_hash)
	go s.Start()
	for _, v := range *peer_list {
		conn := comms.NewConnectionState(v.Ip, v.Port, v.Peer_id, string(request_info.Peer_id), request_info.Info_hash, nil)
		s.EstablishConnection(conn)
	}

	/*
		Call Optimistic Unchoking every 30 seconds.
		Calculate the upload rate of every peer every 10 seconds
		Will need to add feature to have this run continously
	*/
	time.AfterFunc(30*time.Second, func() {
		OptimisticUnchoking(&client)
	})

	time.AfterFunc(10*time.Second, func() {
		CalculateUploadRate(&client)
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

func OptimisticUnchoking(client *LocalPeer) {
	if len(*client.top_four) == 4 && len(*client.peers) > 4 {
		// Update the top 4 peers. Latest unchoked peer may be a replacement
		sort.Slice(*client.peers, func(i, j int) bool {
			return (*client.peers)[i].Upload_speed < (*client.peers)[j].Upload_speed
		})
		first_four := (*client.peers)[:4]
		client.top_four = &first_four

		// Check for any unchoked peers and choke them. Also generate a list of interested peers
		var interested_peers []*data.Peer
		for i := range *client.peers {
			peer := &(*client.peers)[i]
			if !contains(*client.top_four, *peer) {
				if !peer.Am_choking {
					peer.Am_choking = true
				}
				if peer.Am_interested {
					interested_peers = append(interested_peers, peer)
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

func contains(top_four []data.Peer, peer data.Peer) bool {
	for _, v := range top_four {
		if v.Ip.Equal(peer.Ip) && v.Port == peer.Port {
			return true
		}
	}
	return false
}

func CalculateUploadRate(client *LocalPeer) {
	for i := range *client.peers {
		peer := &(*client.peers)[i]
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
