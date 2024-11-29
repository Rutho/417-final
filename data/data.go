package data

import (
	"bytes"
	"client/bencode"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
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
}

func Get_http_addr(announce string) (string, int) {

	regstr := `(\A[a-z][a-z0-9+\-.]*:\/\/)([a-z0-9\-._~%!$&'()*+,;=]+@?[a-z0-9\-._~%]+):*(\d{4})*`
	reg := regexp.MustCompile(regstr)
	m_list := reg.FindStringSubmatch(announce)
	var port int
	var err error

	ip := m_list[2]
	if m_list[3] != "" {
		port, err = strconv.Atoi(m_list[3])
		check(err)
	} else {
		port = 80
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
		panic("INVALID TORRENT FILE")
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
}
func Set_tracker_request(request *Tracker_request, info *Torrent_info) {

	h := sha1.New()
	var ben_info string = string(bencode.Encode(info.Info))
	io.WriteString(h, ben_info)
	//info_hashed  := []byte(url.QueryEscape(string(h.Sum(nil))))
	//fmt.Println("this all not hashed ", ben_info)
	hsh_info := h.Sum(nil)
	fmt.Println(hsh_info)
	pinga := string(hsh_info[:])

	shh := url.QueryEscape(string(hsh_info[:]))
	fmt.Println(pinga)
	request.Info_hash = make([]byte, len(shh))
	copy(request.Info_hash[:], []byte(shh))

	hostname, err := os.Hostname()

	check(err)
	anyb := make([]byte, 20)
	enc_name := "-" + hostname + "-"
	peerby := []byte(enc_name)
	copy(anyb, peerby)
	fin20 := url.QueryEscape(string(anyb[:]))
	request.Peer_id = make([]byte, len(fin20))
	copy(request.Peer_id[:], fin20)
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
func Send_get(request *Tracker_request, info *Torrent_info, response *Tracker_response) {

	var buffer bytes.Buffer
	buffer.WriteString("GET /announce")
	var tmp string
	tmp = fmt.Sprintf("?info_hash=%s", request.Info_hash)
	buffer.WriteString(tmp)
	tmp = fmt.Sprintf("&peer_id=%s", strings.TrimRight(string(request.Peer_id[:]), "\x00"))
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

	var tmp_ip string = "no valid tracker"
	var tmp_port int = -1
	tmp_ip, tmp_port = Get_http_addr(info.Announce)
	dial_val := fmt.Sprintf("%s:%d", tmp_ip, tmp_port)
	//get a real tmp
	var conn net.Conn
	var err error
	conn, err = net.DialTimeout("tcp", dial_val, time.Second)

	if os.IsTimeout(err) {
		do_break := false

		for _, outer := range info.Announce_list {

			for _, inner := range outer.([]string) {

				tmp_ip, tmp_port = Get_http_addr(inner)
				dial_val := fmt.Sprintf("%s:%d", tmp_ip, tmp_port)

				conn, err = net.DialTimeout("tcp", dial_val, time.Second)

				if !(os.IsTimeout(err)) {
					do_break = true
					break

				}

			}

			if do_break {
				break
			}

		}

	}

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
	}

}
