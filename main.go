package main

import (
	"client/data"
	"fmt"
)

func main() {

	var torrent_file *data.Torrent_info = new(data.Torrent_info)
	var request_info *data.Tracker_request = new(data.Tracker_request)
	var s string = "debian-12.8.0-amd64-DVD-1.iso.torrent"
	data.Set_torrent_info(s, torrent_file)
	data.Set_tracker_request(request_info, torrent_file)
	fmt.Println(torrent_file.Announce)
	request_info.Port = 8080
	data.Send_get(request_info, torrent_file)
	fmt.Println(torrent_file.Announce)

}
