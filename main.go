package main

import (
	"client/data"
	"flag"
	"fmt"
)

func main() {
	file_p := flag.String("t", "no_torrent", "filepath/name of torrent")
	usr_port := flag.Int("p", 8080, "port of client")
	flag.Parse()
	fmt.Println(*file_p, *usr_port)
	var torrent_file *data.Torrent_info = new(data.Torrent_info)
	var request_info *data.Tracker_request = new(data.Tracker_request)

	data.Set_torrent_info(*file_p, torrent_file)
	data.Set_tracker_request(request_info, torrent_file)
	fmt.Println(torrent_file.Announce)
	request_info.Port = *usr_port
	request_info.Compact = 1
	request_info.Numwant = 50
	data.Send_get(request_info, torrent_file)
	fmt.Println(torrent_file.Announce)

}
