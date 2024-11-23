package src

import (
	"client/bencode"
	"fmt"
	"os"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func set_torrent_info(file_path string, info *torrent_info) {

	data, err := os.Open(file_path)
	check(err)
	decoded, err := bencode.Decode(data)
	check(err)

	t_url, ok := decoded["announce"].(string)
	if ok {
		info.announce = t_url
	} else {
		panic("INVALID TORRENT FILE")
	}
	an_list, ok := decoded["announce-list"].([]string)
	if ok {
		info.announce_list = an_list
	} else {
		panic("INVALID TORRENT FILE")
	}

	c_date, ok := decoded["creation date"].(int)
	if ok {
		info.creation_date = c_date
	} else {
		panic("INVALID TORRENT FILE")
	}
	cment, ok := decoded["comment"].(string)
	if ok {
		info.comment = cment
	} else {
		panic("INVALID TORRENT FILE")
	}
	author, ok := decoded["creation date"].(string)
	if ok {
		info.created_by = author
	} else {
		panic("INVALID TORRENT FILE")
	}
	encod, ok := decoded["encoding"].(string)
	if ok {
		info.encoding = encod
	} else {
		panic("INVALID TORRENT FILE")
	}

	inf_data, ok := decoded["info"].(map[string]interface{})
	if !ok {
		panic("INVALID TORRENT FILE")
	}
	_, ok = inf_data["length"]
	if ok {
		info.is_multiple = false
	} else {
		info.is_multiple = true
	}
	info.info = inf_data
}
func set_tracker_request(request *tracker_request, info *torrent_info) {
	fmt.Println("placeholder")
}
func send_get(request *tracker_request, info *torrent_info) {
	fmt.Println("placeholder")

}
