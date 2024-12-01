package main

import (
	"client/data"
	"client/server"
	"encoding/binary"
	"flag"
	"fmt"
)

type msgType byte

const (
	choke         msgType = 0
	unchoke       msgType = 1
	interested    msgType = 2
	notInterested msgType = 3
	have          msgType = 4
	bitfield      msgType = 5
	request       msgType = 6
	piece         msgType = 7
	cancel        msgType = 8
	port          msgType = 9
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
	fmt.Println("1")
	for _, v := range *peer_list {
		fmt.Println(v.String())
	}
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
	for _, v := range *peer_list {
		fmt.Println(v.String())
	}

	s := server.New(fmt.Sprintf("127.0.0.1:%d", *usr_port))
	s.Start()

	//This goroutine processes the packets received by the server
	go func(msgCh chan server.Packet) {
		for msg := range msgCh {
			msgLength := msgType(binary.LittleEndian.Uint32(msg.Payload[:4]))

			//Keep alive message
			if msgLength == 0 {

			}

			msgType := msgType(msg.Payload[4])

			switch msgType {
			case choke:
			case unchoke:
			case interested:
			case notInterested:
			case have:
			case bitfield:
			case request:
			case piece:
			case cancel:
			case port:
			//Invalid message
			default:

			}

		}
	}(s.MsgCh)
}
