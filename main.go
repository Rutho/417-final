package main

//TODO: validate handshake we receive (sane info_hash and peer_id)

import (
	"client/comms"
	"client/data"
	"client/peer"
	"client/state"
	"client/logic"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"net"
	"time"
)

func main() {
	// Seed the random number generator
	rand.Seed(time.Now().UTC().UnixNano())

	// Create a State object
	var localState state.State

	// Parse command line arguments
	localState.ParseClargs()

	// --------
	// Get data from the tracker
	// --------
	var torrent_file *data.Torrent_info = new(data.Torrent_info)
	var request_info *data.Tracker_request = new(data.Tracker_request)
	var response_info *data.Tracker_response = new(data.Tracker_response)
	response_info.Peers = new([]peer.Peer)
	data.Set_torrent_info(localState.TorrentFile, torrent_file)
	data.Set_tracker_request(request_info, torrent_file)
	fmt.Println(torrent_file.Announce)
	request_info.Port = localState.Port
	request_info.Compact = 1
	request_info.Numwant = 30
	request_info.Key = "wow"
	torrent_file.Tracker_Type = data.HTTP
	data.Send_get(request_info, torrent_file, response_info)

	// --------
	// Initialize State
	// --------

	// Initialize the IP address
	localState.IP = net.ParseIP("127.0.0.1")

	// Initialize the bitmap
	numberOfpieces := int(math.Ceil(float64(len(torrent_file.Info["pieces"].(string))) / float64(20)))

	localState.Bitmap.New(numberOfpieces)

	// Generate an ID
	localState.GenerateID()

	// Store the peer list into the State struct
	// --------

	s := comms.NewServer(localState.Port, 2048, string(request_info.Peer_id), request_info.Info_hash)
	go s.Start()

	establishCh := make(chan peer.Peer, 60)

	peerMsgCh := make(chan comms.Message, 60)
	stateMsgCh := make(chan comms.Message, 60)

	//Message Filtering
	//Filters peer-specific messages to peerMsgCh and
	//messages that require local state to stateMsgCh
	go func(msgCh chan comms.Message) {
		for msg := range msgCh {
			switch msg.MsgType {
			case comms.Bitfield, comms.Choke, comms.Unchoke,
				comms.Interested, comms.NotInterested, comms.Have,
				comms.KeepAlive:
				peerMsgCh <- msg
			case comms.Piece, comms.Request, comms.Cancel:
				stateMsgCh <- msg
			}
		}
	}(s.MsgCh)

	//Peer-specific Message Processing
	go func(msgCh chan comms.Message) {
		for msg := range msgCh {
			go func(msg comms.Message) {
				peer := localState.FindPeer(msg.PeerId)
				switch msg.MsgType {
				case comms.Bitfield:
					//Bitfield given is not expected.
					if numberOfpieces != int(msg.MsgLength-1) {
						//TODO: disconnect peer.
						return
					}
					peer.Bitmap_lock.Lock()
					//TODO: Parse bitmap message and update peer's bitmap
					peer.Bitmap_lock.Unlock()
				case comms.Choke:
					peer.UpdatePeerChoking(true)
				case comms.Unchoke:
					peer.UpdatePeerChoking(false)
				case comms.Interested:
					peer.UpdatePeerInterested(true)
				case comms.NotInterested:
					peer.UpdatePeerInterested(false)
				case comms.Port:
				case comms.KeepAlive:
				case comms.Request:
				case comms.Have:
					piece_index := binary.BigEndian.Uint32(msg.Payload)
					//casting may be problematic if # of pieces exceed INT_MAX
					peer.Bitmap.MarkPiece(int(piece_index))
				}
			}(msg)
		}
	}(peerMsgCh)

	//State-dependent messages
	go func(msgCh chan comms.Message) {
		for msg := range stateMsgCh {
			go func(msg comms.Message) {
				switch msg.MsgType {
				case comms.Piece:
				case comms.Request:
				case comms.Cancel:
				}
			}(msg)
		}
	}(stateMsgCh)

	go func() {
		for {
			connectedPeer := <-establishCh
			localState.PeersLock.Lock()
			localState.Peers = append(localState.Peers, connectedPeer) //deep copy
			localState.PeersLock.Unlock()
		}
	}()
	//Actively establish connections to peers

	for _, p := range *response_info.Peers {
		go func(peer peer.Peer) {
			conn := comms.NewConnectionState(peer.Ip, peer.Port, peer.Peer_id, string(request_info.Peer_id),
				request_info.Info_hash, nil)
			err := s.EstablishConnection(conn)
			if err == nil {
				peer.Bitmap.New(numberOfpieces)
				establishCh <- peer
			}
		}(p)
	}

	time.Sleep(10000000000)

	fmt.Println(torrent_file.Announce)

	fmt.Println("Hello!!!")

	// Every 10 seconds, refresh Top4
	time.AfterFunc(10*time.Second, func() {
		localState.PeersLock.Lock()
		logic.ChokeAlgorithm(&localState)
		localState.PeersLock.Unlock()
	})

	fmt.Println("Execution reached the end of main().")
}

