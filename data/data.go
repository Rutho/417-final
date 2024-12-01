package data

import (
	"bytes"
	"client/bencode"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type UDP_OR_HTTP int

const (
	HTTP UDP_OR_HTTP = 0
	UDP  UDP_OR_HTTP = 1
)

type Tracker_request struct {
	Info_hash  []byte
	Peer_id    []byte
	Port       int
	Uploaded   int
	Downloaded int
	Left       int
	Compact    int
	No_peer_id int
	Event      string
	Ip         string
	Numwant    int
	Key        string
	Trackerid  int
}
type Peer struct {
	Ip              net.IP
	Port            int
	Am_choking      bool
	Am_interested   bool
	Peer_choking    bool
	Peer_interested bool
	Peer_id         string
}
type Tracker_response struct {
	Failure_reason  string
	Warning_message string
	Interval        int
	Min_interval    int
	Tracker_id      int
	Complete        int
	Incomplete      int
	Peers           *[]Peer
}

type Torrent_info struct {
	Is_multiple   bool
	Info          map[string]interface{}
	Announce      string
	Announce_list []interface{}
	Creation_date int
	Comment       string
	Created_by    string
	Encoding      string
	Tracker_Type  UDP_OR_HTTP
}

func Event_to_num(event string) int32 {

	if event == "completed" {
		return 1
	}
	if event == "stopped" {
		return 3
	}
	if event == "started" {
		return 2
	}
	if event == "none" {
		return 0
	}

	return -1

}

func Get_addr_set_type(announce string, info *Torrent_info) (string, int) {

	regstr := `(\A[a-z][a-z0-9+\-.]*:\/\/)([a-z0-9\-._~%!$&'()*+,;=]+@?[a-z0-9\-._~%]+):*(\d{4})*`
	reg := regexp.MustCompile(regstr)
	m_list := reg.FindStringSubmatch(announce)
	var port int
	var err error
	if len(m_list) < 3 {
		info.Tracker_Type = UDP
		return "none", 0
	}
	ip := m_list[2]
	if m_list[3] != "" {
		port, err = strconv.Atoi(m_list[3])
		check(err)
	} else {
		port = 80
	}

	if m_list[1] == "http://" {
		info.Tracker_Type = HTTP

	} else if m_list[1] == "udp://" {
		info.Tracker_Type = UDP
	}

	return ip, port

}

func (p Peer) String() string {
	return fmt.Sprintf("%s:%d", p.Ip.String(), p.Port)
}
func check(e error) {
	if e != nil {
		panic(e)
	}
}

func Set_torrent_info(file_path string, info *Torrent_info) {

	data, err := os.Open(file_path)
	if err != nil {
		panic("No such file!")
	}
	decoded, err := bencode.Decode(data)
	check(err)

	t_url, ok := decoded["announce"].(string)
	if ok {
		info.Announce = t_url

	} else {
		fmt.Println("weird no announce")
	}
	an_list, ok := decoded["announce-list"].([]interface{})
	if ok {
		info.Announce_list = an_list

	} else {
		//fmt.Println("no_list")
	}

	c_date, ok := decoded["creation date"].(int)
	if ok {
		info.Creation_date = c_date
	} else {
		//fmt.Println("no_date")
	}
	cment, ok := decoded["comment"].(string)
	if ok {
		info.Comment = cment
	} else {
		//fmt.Println("no_comment")
	}
	author, ok := decoded["creation date"].(string)
	if ok {
		info.Created_by = author
	} else {
		//fmt.Println("no author")
	}
	encod, ok := decoded["encoding"].(string)
	if ok {
		info.Encoding = encod
	} else {
		//fmt.Println("no_encod")
	}

	inf_data, ok := decoded["info"].(map[string]interface{})
	if !ok {
		panic("INVALID TORRENT FILE")
	}
	_, ok = inf_data["length"]
	if ok {
		info.Is_multiple = false
	} else {
		info.Is_multiple = true
	}
	info.Info = inf_data
	if info.Announce != "" {
		Get_addr_set_type(info.Announce, info)
	}

}
func Set_tracker_request(request *Tracker_request, info *Torrent_info) {

	h := sha1.New()
	var ben_info string = string(bencode.Encode(info.Info))
	io.WriteString(h, ben_info)
	//info_hashed  := []byte(url.QueryEscape(string(h.Sum(nil))))
	//fmt.Println("this all not hashed ", ben_info)
	hsh_info := h.Sum(nil)
	fmt.Println(hsh_info)

	request.Info_hash = make([]byte, len(hsh_info))
	copy(request.Info_hash[:], hsh_info)

	hostname, err := os.Hostname()

	check(err)
	anyb := make([]byte, 20)
	enc_name := "-" + hostname + "-"
	peerby := []byte(enc_name)
	copy(anyb, peerby)

	request.Peer_id = make([]byte, len(anyb))
	copy(request.Peer_id[:], anyb)
	request.Uploaded = 0
	request.Downloaded = 0
	var file_size int = 0
	if info.Is_multiple {
		file_list, ok := info.Info["files"].([]interface{})
		if !ok {
			panic("no files")
		}

		for _, v := range file_list {
			ele := v.(map[string]interface{})
			tmp, ok := ele["length"].(int64)
			if !ok {
				panic("no file lengths")
			}
			file_size += int(tmp)
		}

	} else {
		tmp, ok := info.Info["length"].(int64)
		if !ok {
			panic("no length")
		}
		file_size = int(tmp)
	}
	request.Left = file_size
	request.Event = "started"

	println("total file size", file_size)

}

func Send_http(request *Tracker_request, info *Torrent_info, response *Tracker_response) {

	var tmp_ip string = "no valid tracker"
	var tmp_port int = -1
	tmp_ip, tmp_port = Get_addr_set_type(info.Announce, info)
	dial_val := fmt.Sprintf("%s:%d", tmp_ip, tmp_port)
	//get a real tmp
	var conn net.Conn
	var err error
	var found_valid bool = true
	conn, err = net.DialTimeout("tcp", dial_val, time.Second)

	if os.IsTimeout(err) || err != nil {
		found_valid = false

		do_break := false

		for _, outer := range info.Announce_list {

			for _, inner := range outer.([]interface{}) {

				tmp_ip, tmp_port = Get_addr_set_type(inner.(string), info)
				dial_val := fmt.Sprintf("%s:%d", tmp_ip, tmp_port)

				conn, err = net.DialTimeout("tcp", dial_val, time.Second)

				if !(os.IsTimeout(err)) {
					found_valid = true
					do_break = true
					break

				}

			}

			if do_break {
				break
			}

		}

	}
	if !found_valid {
		panic("NO VALID CONNECTION")
	}
	var buffer bytes.Buffer
	buffer.WriteString("GET /announce")
	var tmp string
	stinga := url.QueryEscape(string(request.Info_hash))
	tmp = fmt.Sprintf("?info_hash=%s", stinga)

	buffer.WriteString(tmp)
	tado := url.QueryEscape(string(request.Peer_id))
	tmp = fmt.Sprintf("&peer_id=%s", strings.TrimRight(tado[:], "\x00"))
	buffer.WriteString(tmp)
	tmp = fmt.Sprintf("&port=%d", request.Port)
	buffer.WriteString(tmp)
	tmp = fmt.Sprintf("&uploaded=%d", request.Uploaded)
	buffer.WriteString(tmp)
	tmp = fmt.Sprintf("&left=%d", request.Left)
	buffer.WriteString(tmp)
	tmp = fmt.Sprintf("&compact=%d", request.Compact)
	buffer.WriteString(tmp)
	tmp = fmt.Sprintf("&no_peer_id=%d", request.No_peer_id)
	buffer.WriteString(tmp)
	tmp = fmt.Sprintf("&event=%s", request.Event)
	buffer.WriteString(tmp)
	if strings.Compare("", request.Ip) != 0 {
		tmp = fmt.Sprintf("&ip=%s", request.Ip)
		buffer.WriteString(tmp)
	}
	if request.Numwant != 0 {
		tmp = fmt.Sprintf("&numwant=%d", request.Numwant)
		buffer.WriteString(tmp)
	}
	if request.Key != "" {
		tmp = fmt.Sprintf("&key=%s", request.Key)
		buffer.WriteString(tmp)
	}
	if request.Trackerid != 0 {
		tmp = fmt.Sprintf("&trackerid=%d", request.Trackerid)
		buffer.WriteString(tmp)
	}
	buffer.WriteString(" HTTP/1.1\r\n")

	if tmp_port != 80 {
		tmp = fmt.Sprintf("Host: %s:%d\r\n", tmp_ip, tmp_port)
	} else {
		tmp = fmt.Sprintf("Host: %s\r\n", tmp_ip)
	}
	buffer.WriteString(tmp)
	buffer.WriteString("Connection: close\r\n")
	buffer.WriteString("\r\n")
	_, err = conn.Write(buffer.Bytes())
	fmt.Println("I SHOULD GET PASST")
	check(err)
	bun := make([]byte, 1024)
	conn.Read(bun)
	conn.Close()

	var got_string string = string(bun[:])

	tokend := strings.Split(got_string, "\r\n")

	tmp_data := strings.NewReader(tokend[len(tokend)-1])
	ben_data, err := bencode.Decode(tmp_data)
	check(err)

	interval, ok := ben_data["interval"].(int64)
	if ok {
		response.Interval = int(interval)
	} else {
		fmt.Println("no interval")
	}

	peer_string, ok := ben_data["peers"].(string)

	if ok {

		for i := 0; i < len(peer_string); i += 6 {
			cutoff := i + 6
			if cutoff > len(peer_string) {
				cutoff = len(peer_string)
			}
			ip_port := peer_string[i:cutoff]
			ip := net.IP(ip_port[0:4])

			port := binary.BigEndian.Uint16([]byte(ip_port[4:6]))

			var tmp_peer Peer
			tmp_peer.Ip = ip
			tmp_peer.Port = int(port)
			*response.Peers = append(*response.Peers, tmp_peer)
		}

	}
	peer_dict, ok := ben_data["peers"].([]map[string]interface{})
	if ok {

		for _, v := range peer_dict {

			var tmp_peer Peer
			p_id, ok := v["peer id"].(string)
			if ok {
				tmp_peer.Peer_id = p_id

			}
			p_ip, ok := v["ip"].(string)
			if ok {
				tmp_peer.Ip = net.ParseIP(p_ip)

			}
			p_port, ok := v["port"].(int64)
			if ok {
				tmp_peer.Port = int(p_port)

			}
			*response.Peers = append(*response.Peers, tmp_peer)
		}

		print(peer_dict)
	} else {
		fmt.Println("NO DATA FROM TRACKER")
	}

}
func Send_udp(request *Tracker_request, info *Torrent_info, response *Tracker_response) {

	var tmp_ip string = "no valid tracker"
	var tmp_port int = -1
	var dial_val string
	tmp_ip, tmp_port = Get_addr_set_type(info.Announce, info)
	dial_val = fmt.Sprintf("%s:%d", tmp_ip, tmp_port)
	//get a real tmp
	var conn net.Conn
	var err error

	conn, err = net.Dial("udp", dial_val)

	if err != nil {
		panic(err)
	}
	var buffer *bytes.Buffer = new(bytes.Buffer)
	connection_id := int64(0x41727101980)
	binary.Write(buffer, binary.BigEndian, connection_id)
	action := int32(0)
	binary.Write(buffer, binary.BigEndian, action)
	transaction_id := rand.Int31()
	binary.Write(buffer, binary.BigEndian, transaction_id)
	var got_connection bool = false
	conn.Write(buffer.Bytes())
	var serv_reply []byte = make([]byte, 16)

	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))

	_, err = conn.Read(serv_reply)
	if err != nil {

		do_break := false

		for _, outer := range info.Announce_list {

			for _, inner := range outer.([]interface{}) {

				tmp_ip, tmp_port = Get_addr_set_type(inner.(string), info)
				dial_val = fmt.Sprintf("%s:%d", tmp_ip, tmp_port)

				conn, err = net.Dial("udp", dial_val)
				check(err)
				conn.Write(buffer.Bytes())

				conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))

				_, err = conn.Read(serv_reply)

				if err == nil {
					do_break = true
					fmt.Println("do we get here")
					got_connection = true
					break

				}

			}

			if do_break {
				break
			}

		}

	} else {
		got_connection = true

	}

	if !got_connection {
		panic("no avalible tracker")
	}
	var gotten_transaction_id int32

	r := bytes.NewReader(serv_reply)
	var temp_byte = make([]byte, 4)
	r.Read(temp_byte)
	action = int32(binary.BigEndian.Uint32(temp_byte))
	temp_byte = make([]byte, 4)
	r.Read(temp_byte)
	gotten_transaction_id = int32(binary.BigEndian.Uint32(temp_byte))
	temp_byte = make([]byte, 8)
	r.Read(temp_byte)
	connection_id = int64(binary.BigEndian.Uint64(temp_byte))
	fmt.Println("waddado")
	fmt.Println(transaction_id, gotten_transaction_id)
	fmt.Println("waddado")
	buffer = new(bytes.Buffer)

	binary.Write(buffer, binary.BigEndian, connection_id)
	binary.Write(buffer, binary.BigEndian, int32(1))
	binary.Write(buffer, binary.BigEndian, gotten_transaction_id)
	binary.Write(buffer, binary.BigEndian, request.Info_hash)
	fmt.Println(request.Info_hash)
	binary.Write(buffer, binary.BigEndian, request.Peer_id)
	binary.Write(buffer, binary.BigEndian, int64(request.Downloaded))
	binary.Write(buffer, binary.BigEndian, int64(request.Left))
	binary.Write(buffer, binary.BigEndian, int64(request.Uploaded))

	//binary.Write(buffer, binary.BigEndian, Event_to_num(request.Event))
	binary.Write(buffer, binary.BigEndian, int32(2))
	binary.Write(buffer, binary.BigEndian, uint32(0))
	binary.Write(buffer, binary.BigEndian, uint32(55))
	to_want := request.Numwant
	if to_want == 0 {
		to_want = -1
	}
	binary.Write(buffer, binary.BigEndian, int32(to_want))
	binary.Write(buffer, binary.BigEndian, uint16(request.Port))

	conn.Write(buffer.Bytes())
	fmt.Println("wowza send")
	var ann_reply []byte = make([]byte, 1024)

	conn.Read(ann_reply)
	//action
	ann_reader := bytes.NewReader(ann_reply)
	var srv_data int32
	var srv_bd = make([]byte, 4)
	ann_reader.Read(srv_bd)
	srv_data = int32(binary.BigEndian.Uint32(srv_bd))
	if srv_data != 1 {
		fmt.Println("ERROR SEND")
	}
	//transaction_id
	srv_bd = make([]byte, 4)
	ann_reader.Read(srv_bd)
	srv_data = int32(binary.BigEndian.Uint32(srv_bd))
	//interval
	srv_bd = make([]byte, 4)
	ann_reader.Read(srv_bd)
	srv_data = int32(binary.BigEndian.Uint32(srv_bd))
	response.Interval = int(srv_data)
	//leechers
	srv_bd = make([]byte, 4)
	ann_reader.Read(srv_bd)
	srv_data = int32(binary.BigEndian.Uint32(srv_bd))
	//seeders
	srv_bd = make([]byte, 4)
	ann_reader.Read(srv_bd)
	srv_data = int32(binary.BigEndian.Uint32(srv_bd))

	for true {

		peer_data := make([]byte, 6)

		ann_reader.Read(peer_data)
		chk := binary.BigEndian.Uint32(peer_data[0:4])
		if chk == 0 {
			break
		} else {
			ip_addr := net.IP(peer_data[0:4])
			port := binary.BigEndian.Uint16(peer_data[4:6])
			var tmp_peer Peer
			tmp_peer.Ip = ip_addr
			tmp_peer.Port = int(port)
			*response.Peers = append(*response.Peers, tmp_peer)
		}

	}

}

func Send_get(request *Tracker_request, info *Torrent_info, response *Tracker_response) {
	info.Tracker_Type = UDP
	if info.Tracker_Type == HTTP {
		Send_http(request, info, response)

	} else if info.Tracker_Type == UDP {
		Send_udp(request, info, response)

	}

}
