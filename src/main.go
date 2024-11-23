package src

type tracker_request struct {
	info_hash  [20]byte
	peer_id    [20]byte
	port       uint
	uploaded   uint
	downloaded uint
	left       uint
	compact    int
	no_peer_id int
	event      string
	ip         string
	numwant    int
	key        int
	trackerid  int
}

type tracker_response struct {
	failure_reason  string
	warning_message string
	interval        int
	min_interval    int
	tracker_id      int
	complete        int
	incomplete      int
	peers           []map[string]interface{}
	b_peers         string
}

type torrent_info struct {
	is_multiple   bool
	info          map[string]interface{}
	announce      string
	announce_list []string
	creation_date int
	comment       string
	created_by    string
	encoding      string
}
