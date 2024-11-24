package data

import (
	"bytes"
	"client/bencode"
	"crypto/sha1"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type Tracker_request struct {
	Info_hash  []byte
	Peer_id    []byte
	Port       uint
	Uploaded   uint
	Downloaded uint
	Left       uint
	Compact    int
	No_peer_id int
	Event      string
	Ip         string
	Numwant    int
	Key        int
	Trackerid  int
}

type Tracker_response struct {
	Failure_reason  string
	Warning_message string
	Interval        int
	Min_interval    int
	Tracker_id      int
	Complete        int
	Incomplete      int
	Peers           []map[string]interface{}
	B_peers         string
}

type Torrent_info struct {
	Is_multiple   bool
	Info          map[string]interface{}
	Announce      string
	Announce_list []string
	Creation_date int
	Comment       string
	Created_by    string
	Encoding      string
	Track_host    string
	Track_port    int
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func Set_torrent_info(file_path string, info *Torrent_info) {

	data, err := os.Open(file_path)
	check(err)
	decoded, err := bencode.Decode(data)
	check(err)

	t_url, ok := decoded["announce"].(string)
	if ok {
		info.Announce = t_url
		regstr := `(\A[a-z][a-z0-9+\-.]*:\/\/)([a-z0-9\-._~%!$&'()*+,;=]+@?[a-z0-9\-._~%]+):(\d{4})*`
		reg := regexp.MustCompile(regstr)
		m_list := reg.FindStringSubmatch(info.Announce)
		info.Track_host = m_list[2]
		if m_list[3] != "" {
			info.Track_port, err = strconv.Atoi(m_list[3])
			check(err)
		} else {
			info.Track_port = 80
		}

	} else {
		panic("INVALID TORRENT FILE")
	}
	an_list, ok := decoded["announce-list"].([]string)
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
	fmt.Println("this not url", string(hsh_info[:]))
	fmt.Println("this all url and hashed ", string(request.Info_hash[:]))
	hostname, err := os.Hostname()
	fmt.Println("host", hostname)
	check(err)
	anyb := make([]byte, 20)
	enc_name := "-" + hostname + "-"
	peerby := []byte(enc_name)
	copy(anyb, peerby)
	fin20 := url.QueryEscape(string(anyb[:]))
	request.Peer_id = make([]byte, len(fin20))
	copy(request.Peer_id[:], fin20)
	fmt.Println("encoded host 20 byte", request.Peer_id)
	request.Uploaded = 0
	request.Downloaded = 0
	var file_size uint = 0
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
			file_size += uint(tmp)
		}

	} else {
		tmp, ok := info.Info["length"].(int64)
		if !ok {
			panic("no length")
		}
		file_size = uint(tmp)
	}
	request.Left = file_size
	request.Event = "started"

	println("total file size", file_size)

}
func Send_get(request *Tracker_request, info *Torrent_info) {

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
	if request.Key != 0 {
		tmp = fmt.Sprintf("&key=%d", request.Key)
		buffer.WriteString(tmp)
	}
	if request.Trackerid != 0 {
		tmp = fmt.Sprintf("&trackerid=%d", request.Trackerid)
		buffer.WriteString(tmp)
	}
	buffer.WriteString(" HTTP/1.1\r\n")

	tmp = fmt.Sprintf("Host: %s:%d\r\n", info.Track_host, info.Track_port)
	buffer.WriteString(tmp)
	buffer.WriteString("\r\n")
	fmt.Println(buffer.String())
	tmp = fmt.Sprintf("%s:%d", info.Track_host, info.Track_port)
	conn, err := net.Dial("tcp", tmp)
	check(err)
	_, err = conn.Write(buffer.Bytes())
	check(err)
	bun := make([]byte, 1024)
	conn.Read(bun)
	conn.Close()
	fmt.Println(bun)
	var got_string string = string(bun[:])
	ben_start := strings.Index(got_string, "d8")
	coded := got_string[ben_start:]
	fmt.Println(coded)
	into_read := strings.NewReader(coded)

	track_dec, err := bencode.Decode(into_read)
	print(track_dec)

	/*listener, err := net.Listen("tcp", "localhost:8080")
	check(err)
	conn, err = listener.Accept()
	check(err)
	_, err = conn.Read(bun)
	check(err)
	fmt.Println("mung")
	fmt.Println(bun) */

}
