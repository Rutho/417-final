package main

//TODO: validate handshake we receive (sane info_hash and peer_id)
//TODO: Currently we send one request to a single peer for each block. We have to handle the case that the peer fails or doesn't send a piece back
//Idea for handling responses when we have multiple requests for the same block to different peers:
//Create a outstanding request queue (sort of like chord) and when we get a piece message,
//We check if the piece message has a corresponding request in the queue. If so, we remove
//all requests that were asking for the same thing (same piece idx, offset; len should be the same for all messages except end of piece)
//and send cancel messages to the peers that were the destination of these extra requests.
//If piece messages for these requests do still come, we just ignore them.
//The question is, do we consider these piece messages in our upload rate calculation? YES

import (
	"client/comms"
	"client/data"
	"client/filemanager"
	"client/peer"
	"client/request"
	"client/receive"
	"client/state"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"sort"
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

	// Initialize the bitmap
	localState.PieceLength = torrent_file.Info["piece length"].(int64)
	localState.FileSize = torrent_file.Info["length"].(int64)
	localState.PieceHashes = []byte(torrent_file.Info["pieces"].(string))
	localState.NumPieces = len(localState.PieceHashes) / 20
	localState.Bitmap.New(localState.PieceHashes, localState.PieceLength, localState.FileSize)

	pieceLength := torrent_file.Info["piece length"].(int64)
	//fmt.Printf("piece length: %d\n", pieceLength)
	numberOfpieces := int(math.Ceil(float64(len(torrent_file.Info["pieces"].(string))) / float64(20)))

	// Generate an ID
	localState.GenerateID()

	// Store the Info_hash
	localState.Info_hash = request_info.Info_hash

	// Store the peer list into the State struct
	// --------

	s := comms.NewServer(localState.Port, 2048, localState.ID, localState.Info_hash)
	go s.Start()

	peerMsgCh := make(chan comms.Message, 50)
	stateMsgCh := make(chan comms.Message, 50)
	establishCh := make(chan peer.Peer, 30)
	disconnectCh := &s.DisconnectCh

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
			//fmt.Printf("peer address: %s\n", msg.PeerAddress)
			peer := localState.Peers.FindPeerByID(msg.PeerId)
			switch msg.MsgType {
			case comms.Bitfield:
				//Bitfield size can differ a little from numberOfPieces since
				//numberOfPieces may not be a multiple of 8, so last byte in Bitfield may have spare bits
				bitfieldSizeInBits := int(math.Ceil((float64(numberOfpieces) / float64(8)))) * 8
				if bitfieldSizeInBits != (int(msg.MsgLength-1) * 8) {
					s.DisconnectCh <- msg.PeerId
					return
				}
				// peer.Bitmap.New(localState.PieceHashes, localState.PieceLength, localState.FileSize)
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
				extra, err := localState.Bitmap.HasExtraPieces(&peer.Bitmap)
				if err != nil {
					*disconnectCh <- peer.Peer_id
				}
				if extra && !peer.Am_interested {
					peer.UpdateInterested(true)
					peer.Connection.SendInterested()
				} else if !extra && peer.Am_interested {
					peer.UpdateInterested(false)
					peer.Connection.SendNotInterested()
				}
			case comms.Choke:
				peer.UpdatePeerChoking(true)
			case comms.Unchoke:
				peer.UpdatePeerChoking(false)
				p := localState.Peers.FindPeerByID(msg.PeerId)

				// add to Top 4
				localState.Top4Lock.RLock()
				if len(localState.Top4) < 4 && !contains(localState.Top4, p) {
					localState.Top4 = append(localState.Top4, p)
					p.UpdateChoking(false)
					p.Connection.SendUnchoke()
				}
				localState.Top4Lock.RUnlock()
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
				if !localState.Bitmap.Pieces[int(piece_index)].Complete() && !peer.Am_interested {
					peer.UpdateInterested(true)
					peer.Connection.SendInterested()
				}
			}
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
					// If we have all the pieces
					if localState.Bitmap.AllDownloaded() {
						break // ignore the piece
					}
					// If we have the piece
					if localState.Bitmap.Pieces[int(pieceIdx)].PendingComplete() {
						break // ignore the piece
					}

					//TODO: Update Peer's upload rate
					//Either peer is sending us stuff willy-nilly or piece was completed
					//between the time we sent a request message to this peer and we received the piece message from this peer
					//Latter case only happens if we're sending multiple requests for the same block to different peers
					//Should we include these messages for the calculation of peer's upload rate then? YES

					buf := msg.Payload[8:]

					p := localState.Peers.FindPeerByID(msg.PeerId)

					ok := receive.Process(&localState, pieceIdx, int64(pieceOffset), buf)
					if !ok { // something went wrong (e.g. incorrect hash)
						break
					}

					// fmt.Printf("GOT Piece %d\n", pieceIdx)

					p.Uploaded_bytes += int64(len(buf))

					// Write to the file
					f.WritePieceOffset(pieceIdx, int(pieceLength), pieceOffset, buf)

					if f.IsPieceDownloaded(pieceIdx, int(pieceLength)) {
						localState.PeersLock.Lock()
						for i := range localState.Peers.List {
							localState.Peers.List[i].Connection.SendHave(pieceIdx)
							go func(p *peer.Peer) {
								extra, _ := localState.Bitmap.HasExtraPieces(&p.Bitmap)
								if !extra && p.Peer_interested {
									p.UpdateInterested(false)
									p.Connection.SendNotInterested()
								} else if extra && !p.Peer_interested {
									p.UpdateInterested(true)
									p.Connection.SendInterested()
								}
							}(&localState.Peers.List[i])
						}
						localState.PeersLock.Unlock()
					}
					extra, _ := localState.Bitmap.HasExtraPieces(&p.Bitmap)
					if extra {
						if !p.Am_interested {
							p.UpdateInterested(true)
							p.Connection.SendInterested()
						}
					} else {
						p.UpdateInterested(false)
						p.Connection.SendNotInterested()
					}
				case comms.Request:
					peer := localState.Peers.FindPeerByID(msg.PeerId)
					if !localState.Bitmap.Pieces[int(pieceIdx)].Complete() || peer.Am_choking {
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
			select {
			case connectedPeer := <-establishCh:
				localState.PeersLock.Lock()
				localState.Peers.List = append(localState.Peers.List, connectedPeer) //deep copy
				localState.PeersLock.Unlock()
				go func() {
					err := s.ReadLoop(connectedPeer.Connection)
					if err != nil {
						*disconnectCh <- connectedPeer.Peer_id
					}
				}()
			case disconnectedPeerId := <-*disconnectCh:
				fmt.Printf("disconnected: %s\n", disconnectedPeerId)
				localState.RemovePeer(disconnectedPeerId)
				localState.PeersLock.Lock()
				localState.Top4Lock.Lock()
				SortPeersAndUpdateTop4(&localState)
				localState.PeersLock.Unlock()
				localState.Top4Lock.Unlock()
			}

			//buf, n := localState.Bitmap.Compress()
		}
	}()

	//Actively establish connections to peers
	for _, p := range *response_info.Peers {
		go func(peer peer.Peer) {
			conn := comms.NewConnectionState(peer.Ip, peer.Port, peer.Peer_id, string(request_info.Peer_id),
				request_info.Info_hash, nil)
			peerId, err := s.EstablishConnection(conn, *disconnectCh)
			if err != nil {
				return
			}
			fmt.Printf("peerID: %s\n", peerId)
			peer.New = true
			peer.Bitmap.New(localState.PieceHashes, localState.PieceLength, localState.FileSize)
			peer.Connection = conn
			peer.Peer_id = peerId
			establishCh <- peer
		}(p)
	}

	time.Sleep(20000000000)

	// Goroutine to send requests for pieces, until entire file is received
	requestTicker := time.NewTicker(100000000) // 0.1 sec
	go func() {
		for range requestTicker.C {
			if localState.SendARequest() {
				// Stop sending requests
				requestTicker.Stop()
				// Cancel outgoing requests
				request.Cancel(&localState.RequestQueue)
			}
		}
	}()

	// Goroutine to periodically clean up requests from RequestQueue
	requestQueueCleanerTicker := time.NewTicker(1 * time.Second)
	go func() {
		for range requestQueueCleanerTicker.C {
			var reqs_to_keep []int
			for i, r := range localState.RequestQueue {
				// Compare
				if time.Now().Unix() - r.Timestamp > 5 {
					localState.RequestQueue[i].Clean(&localState.Bitmap)
				} else {
					reqs_to_keep = append(reqs_to_keep, i)
				}
			}
			for i, j := range reqs_to_keep {
				localState.RequestQueue[i] = localState.RequestQueue[j]
			}

		}
	}()

	// Goroutine to periodically print download progress
	downloadProgressTicker := time.NewTicker(1 * time.Second)
	go func() {
		for range downloadProgressTicker.C {
			fmt.Println(localState.Bitmap)

		}
	}()

	fmt.Println(localState)
	fmt.Println(torrent_file.Announce)

	fmt.Println("Hello!!!")

	sendKeepAliveTicker := time.NewTicker(30 * time.Second)
	go func() {
		for {
			<-sendKeepAliveTicker.C
			fmt.Println("sendKeepAlive()")
			SendKeepAlive(&localState)
		}
	}()

	/*
		Call Optimistic Unchoking every 30 seconds.
		Calculate the upload rate of every peer every 10 seconds
		Will need to add feature to have this run continously
	*/
	// Tickers for periodic tasks
	optimisticUnchokingTicker := time.NewTicker(30 * time.Second)
	printLocalStateTicker := time.NewTicker(30 * time.Second)
	var sendTrackerRequestTicker *time.Ticker
	upcomingPanicRequest := false
	if len(localState.Peers.List) < 30 {
		sendTrackerRequestTicker = time.NewTicker(time.Duration(10) * time.Second)
		upcomingPanicRequest = true
	} else {
		sendTrackerRequestTicker = time.NewTicker(time.Duration(int(math.Min(float64(response_info.Interval), 300))) * time.Second)
		upcomingPanicRequest = false
	}

	//This periodically checks number of peers we're connected to, so that we can reset sendTrackerRequest
	//
	periodicCheckNumberOfPeers := time.NewTicker(10 * time.Second)
	go func() {
		for {
			<-periodicCheckNumberOfPeers.C
			if upcomingPanicRequest {
				continue
			}
			if len(localState.Peers.List) < 30 {
				sendTrackerRequestTicker.Reset(time.Duration(10) * time.Second)
				upcomingPanicRequest = true
			}
		}
	}()

	for {
		select {
		case <-optimisticUnchokingTicker.C:
			fmt.Println("OptimisticUnchoking()")
			// Lock and perform optimistic unchoking
			OptimisticUnchoking(&localState)

		case <-printLocalStateTicker.C:
			localState.PeersLock.Lock()
			fmt.Println(localState)
			localState.PeersLock.Unlock()
		case <-sendTrackerRequestTicker.C:
			fmt.Println("sendTrackerRequest()")
			data.Resend_get(request_info, torrent_file, response_info)
			for _, p := range *response_info.Peers {
				if localState.FindPeerByAddr(p.Ip, p.Port) != nil {
					continue
				}
				go func(peer peer.Peer) {
					conn := comms.NewConnectionState(peer.Ip, peer.Port, peer.Peer_id, string(request_info.Peer_id),
						request_info.Info_hash, nil)
					peerId, err := s.EstablishConnection(conn, *disconnectCh)
					if err != nil {
						return
					}
					peer.New = true
					peer.Bitmap.New(localState.PieceHashes, localState.PieceLength, localState.FileSize)
					peer.Connection = conn
					peer.Peer_id = peerId
					establishCh <- peer
				}(p)
			}
			if len(localState.Peers.List) < 30 {
				sendTrackerRequestTicker.Reset(time.Duration(5) * time.Second)
				upcomingPanicRequest = true
			} else {
				sendTrackerRequestTicker.Reset(time.Duration(int(math.Min(float64(response_info.Interval), 300))) * time.Second)
				upcomingPanicRequest = false
			}
		}

	}
}

func OptimisticUnchoking(localState *state.State) {
	localState.PeersLock.Lock()
	localState.Top4Lock.Lock()
	defer localState.PeersLock.Unlock()
	defer localState.Top4Lock.Unlock()

	//fmt.Println(localState)
	if len(localState.Top4) == 4 && len(localState.Peers.List) > 4 {
		SortPeersAndUpdateTop4(localState)

		// Check for any unchoked peers and choke them. Also generate a list of interested peers
		var interested_peers []*peer.Peer
		for i := range localState.Peers.List {
			p := &localState.Peers.List[i]

			if !contains(localState.Top4, p) {
				p.Am_choking_lock.RLock()
				if !p.Am_choking {
					p.Am_choking_lock.RUnlock()
					p.UpdateChoking(true)
					p.Connection.SendChoke()
				} else {
					p.Am_choking_lock.RUnlock()
				}

				p.Peer_interested_lock.RLock()
				if p.Am_interested {
					interested_peers = append(interested_peers, p)
				}
				p.Peer_interested_lock.RUnlock()
			}

		}
		if len(interested_peers) < 1 {
			fmt.Println("No interested peers to unchoke")
			return
		}

		// Randomly unchoke a interested peer
		optimisticPeer := interested_peers[weightedRandomChoice(interested_peers)]
		optimisticPeer.UpdateChoking(false)
		optimisticPeer.Connection.SendUnchoke()
		fmt.Printf("Unchoked: %s:%d\n", optimisticPeer.Ip, optimisticPeer.Port)

		// Reset the uploaded bytes for next cycle
		for i := range localState.Peers.List {
			p := &localState.Peers.List[i]
			p.Uploaded_bytes = 0
		}
	} else {
		fmt.Println("Not enough peers for optimistic unchoking")
	}
}

// sort the peers based on upload speed and update the top 4
func SortPeersAndUpdateTop4(localState *state.State) {
	// Sort the peers in descending order of upload speed. Latest unchoked peer may be a replacement
	sort.SliceStable(localState.Peers.List, func(i, j int) bool {
		return (localState.Peers.List)[i].Uploaded_bytes > (localState.Peers.List)[j].Uploaded_bytes
	})

	// Update the top 4. Top4 stores pointers to Peer
	var newTop4 []*peer.Peer
	for i := 0; i < len(localState.Peers.List); i++ {
		if len(newTop4) == 4 {
			break
		}

		localState.Peers.List[i].Am_choking_lock.RLock()
		localState.Peers.List[i].Peer_interested_lock.RLock()
		localState.Peers.List[i].Peer_choking_lock.RLock()

		if localState.Peers.List[i].Peer_interested || !localState.Peers.List[i].Peer_choking {
			if localState.Peers.List[i].Am_choking {
				localState.Peers.List[i].Am_choking_lock.RUnlock()
				localState.Peers.List[i].UpdateChoking(false)
				localState.Peers.List[i].Connection.SendUnchoke()
			} else {
				localState.Peers.List[i].Am_choking_lock.RUnlock()
			}

			newTop4 = append(newTop4, &localState.Peers.List[i])
		}
		localState.Peers.List[i].Peer_choking_lock.RUnlock()
		localState.Peers.List[i].Peer_interested_lock.RUnlock()
	}

	localState.Top4 = newTop4
	fmt.Println("After sort and update")
	localState.PrintTop4()
}

// check if the given peer is in the top 4
func contains(top_four []*peer.Peer, p *peer.Peer) bool {
	for _, peer := range top_four {
		if peer.Ip.Equal(p.Ip) && peer.Port == p.Port {
			return true
		}
	}
	return false
}

// Assigns a weight to each interested peer. New peers are 3x more likely to be chosen
func weightedRandomChoice(interested_peers []*peer.Peer) int {
	// Step 1: Assign weights
	weights := make([]float64, len(interested_peers))
	totalWeight := 0.0
	for i, peer := range interested_peers {
		if peer.New { // New peers are 3x more likely to get unchoked
			weights[i] = 3.0
			interested_peers[i].New = false
		} else {
			weights[i] = 1.0
		}
		totalWeight += weights[i]
	}

	// Step 2: Normalize weights
	for i := range weights {
		weights[i] /= totalWeight
	}

	// Step 3: Generate a random choice
	r := rand.Float64()
	cumulative := 0.0
	for i, weight := range weights {
		cumulative += weight
		if r < cumulative {
			return i
		}
	}
	return len(interested_peers) - 1 // Fallback in case of rounding errors
}

func SendKeepAlive(s *state.State) {
	s.PeersLock.RLock()
	for i := range s.Peers.List {
		s.Peers.List[i].Connection.SendKeepAlive()
	}
	s.PeersLock.RUnlock()
}
