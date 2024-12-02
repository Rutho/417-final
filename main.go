package main

import (
	"client/comms"
	"client/data"
	"flag"
	"fmt"
)

func main() {
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

	data.Send_get(request_info, torrent_file, response_info)
	//fmt.Println(request_info, response_info)
	fmt.Println(torrent_file.Announce)

	s := comms.NewServer(*usr_port, 2048, string(request_info.Peer_id), request_info.Info_hash)
	go s.Start()
	for _, v := range *peer_list {
		conn := comms.NewConnectionState(v.Ip, v.Port, v.Peer_id, string(request_info.Peer_id), request_info.Info_hash, nil)
		s.EstablishConnection(conn)
	}

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
