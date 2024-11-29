package main

import (
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
	request_info.Numwant = 5
	request_info.Key = "umd123"
	data.Send_get(request_info, torrent_file, response_info)
	fmt.Println(request_info, response_info)
	fmt.Println(torrent_file.Announce)
	for _, v := range *peer_list {
		fmt.Println(v.String())
	}
	fmt.Println("temper")
}
