package main

//TODO: validate handshake we receive (sane info_hash and peer_id)
//TODO: Currently we send one request to a single peer for each block. We have to handle the case that the peer fails or doesn't send a piece back

import (
	"client/comms"
	"client/data"
	"client/filemanager"
	"client/logic"
	"client/peer"
	"client/state"
	"encoding/binary"
	"fmt"
	"io"
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

	infoHashStr := torrent_file.Info["pieces"].(string)
	f := filemanager.Open(torrent_file.Info["name"].(string), []byte(infoHashStr))
	fmt.Printf("Saving to %s\n", f.Filename)

	// --------
	// Initialize State
	// --------

	// Initialize the IP address
	localState.IP = net.ParseIP("127.0.0.1")

	pieceLength := torrent_file.Info["piece length"].(int64)
	fmt.Printf("piece length: %d\n", pieceLength)
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
	stateMsgCh := make(chan comms.Message, 100)

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
					//Bitfield size can differ a little from numberOfPieces since
					//numberOfPieces may not be a multiple of 8, so last byte in Bitfield may have spare bits
					bitfieldSizeInBits := int(math.Ceil((float64(numberOfpieces) / float64(8)))) * 8
					if bitfieldSizeInBits != (int(msg.MsgLength-1) * 8) {
						//TODO: disconnect peer.
						return
					}
					peer.Bitmap_lock.Lock()
					for i, b := range msg.Payload {
						for kth := 0; kth < 8; kth++ {
							if (i*8)+kth >= numberOfpieces {
								break
							}
							//AND Masking kth_bit
							if (b & (1 << (8 - (kth + 1)))) != 0 {
								peer.Bitmap.MarkPiece((i * 8) + kth)
							}
						}
					}
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
					peer.Port = int(binary.BigEndian.Uint16(msg.Payload))
				case comms.KeepAlive:
				case comms.Have:
					piece_index := msg.Payload[0]
					//casting may be problematic if # of pieces exceed INT_MAX
					peer.Bitmap_lock.Lock()
					peer.Bitmap.MarkPiece(int(piece_index))
					peer.Bitmap_lock.Unlock()
				}
			}(msg)
		}
	}(peerMsgCh)

	//State-dependent messages
	go func(msgCh chan comms.Message) {
		for msg := range stateMsgCh {
			go func(msg comms.Message) {
				pieceIdx := int(binary.BigEndian.Uint32(msg.Payload[:4]))
				pieceOffset := int(binary.BigEndian.Uint32(msg.Payload[4:8]))
				switch msg.MsgType {
				case comms.Piece:
					if localState.Bitmap.HasPiece(int(pieceIdx)) {
						return
					}
					//TODO: Update Peer's upload rate
					buf := msg.Payload[8:]
					f.WritePieceOffset(pieceIdx, int(pieceLength), pieceOffset, buf)
					if f.IsPieceDownloaded(pieceIdx, int(pieceLength)) {
						localState.Bitmap.MarkPiece(int(pieceIdx))
						//TODO: Send Have Messages to peers
					}
				case comms.Request:
					peer := localState.FindPeer(msg.PeerId)
					if !localState.Bitmap.HasPiece(pieceIdx) || peer.Am_choking {
						return
					}
					length := binary.BigEndian.Uint32(msg.Payload[8:12])
					buf, n, err := f.ReadPieceOffset(pieceIdx, int(pieceLength), pieceOffset, length)
					if err != nil && err != io.EOF {
						fmt.Println("Processing Request MSG: read piece offset:", err)
					}
					buf = buf[:n]
					if peer != nil {
						(*peer.Connection).SendPiece(uint32(pieceIdx), uint32(pieceOffset), buf, n)
					} else {
						fmt.Printf("Abnormal Error: Failed to find peer that requested pieces")
					}

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
			buf, n := localState.Bitmap.Compress()
			connectedPeer.Connection.SendBitfield(buf, n)
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
				peer.Connection = conn
				establishCh <- peer
			}
		}(p)
	}

	time.Sleep(20000000000)

	//fmt.Println(localState)
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
