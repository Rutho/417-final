// Package server provides a way to initialize and start a TCP server
// That listens and accepts connections, and sends received messages to the message channel
package comms

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	localAddress net.IP
	localPort    int
	ln           net.Listener
	quitCh       chan struct{}
	MsgCh        chan Message
	DisconnectCh chan string
	maxPktSize   uint32
	serverId     string
	infoHash     []byte
}

type Packet struct {
	FromAddress net.IP
	FromPort    int
	FromId      string
	Payload     []byte
}

type ConnectionState struct {
	PeerAddress net.IP
	PeerPort    int
	PeerId      string
	clientId    string
	infoHash    []byte
	Conn        *net.Conn
}

type Message struct {
	PeerAddress net.IP
	PeerPort    int
	PeerId      string
	MsgLength   uint32
	MsgType     MsgType
	Payload     []byte
}

type MsgType byte

// Note: Keep-alive and Handshake messages are different from the other message types. They don't correspond
// to the value of the type byte in the message packet since keep-alive messages don't have a type
// byte in their packets. I'm giving it a type enum value just to stay consistent with the other message types.
// Remember NOT to include '10' as the type byte in keep-alive messages!
const (
	Choke         MsgType = 0
	Unchoke       MsgType = 1
	Interested    MsgType = 2
	NotInterested MsgType = 3
	Have          MsgType = 4
	Bitfield      MsgType = 5
	Request       MsgType = 6
	Piece         MsgType = 7
	Cancel        MsgType = 8
	Port          MsgType = 9
	KeepAlive     MsgType = 10
	Handshake     MsgType = 11
)

// Initializes a server instance and returns a pointer to it
func NewServer(localPort int, maxPktSize uint32, serverId string, infoHash []byte) *Server {
	return &Server{
		localAddress: net.ParseIP("127.0.0.1"),
		localPort:    localPort,
		quitCh:       make(chan struct{}),
		MsgCh:        make(chan Message, 50),
		DisconnectCh: make(chan string, 40),
		maxPktSize:   maxPktSize,
		serverId:     serverId,
		infoHash:     infoHash,
	}
}

func NewMessage(peerAddress net.IP, peerPort int, peerId string, msgLength uint32, msgType MsgType, payload []byte) *Message {
	return &Message{
		PeerAddress: peerAddress,
		PeerPort:    peerPort,
		PeerId:      peerId,
		MsgLength:   msgLength,
		MsgType:     msgType,
		Payload:     payload,
	}
}

func NewConnectionState(peerAddress net.IP, peerPort int, peerId string, clientId string, infoHash []byte, conn *net.Conn) *ConnectionState {
	return &ConnectionState{
		PeerAddress: peerAddress,
		PeerPort:    peerPort,
		PeerId:      peerId,
		clientId:    clientId,
		infoHash:    infoHash,
	}
}

// Start starts the server instance, making it listen for and accept connections.
// It'll receive packets from the connections and write them to the channel
func (s *Server) Start() {
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.localAddress, s.localPort))
	if err != nil {
		fmt.Println("listen error:", err)
	}
	defer ln.Close()
	s.ln = ln

	fmt.Printf("Listening on %s\n", s.localAddress)

	go s.acceptLoop()

	<-s.quitCh
	close(s.MsgCh)

	return
}

// Establishes a TCP connection and initiates BitTorrent handshake (active open)
func (s *Server) EstablishConnection(conn *ConnectionState, disconnectCh chan string) (string, error) {
	c, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", conn.PeerAddress, conn.PeerPort), 7*time.Second)
	if err != nil {
		fmt.Println("EstablishConnection: dial error:", err)
		return "", err
	}

	conn.Conn = &c

	peerId, err := conn.TwoWayHandshake("BitTorrent protocol", true)
	if err != nil {
		c.Close()
		return "", err
	}
	//Check if handshake went well. Else, error stuff happens
	//If it went well, send connection state to connCh and let another goroutine handle processing that

	conn.PeerId = peerId
	return peerId, nil
}

func (s *Server) acceptLoop() {
	for {
		c, err := s.ln.Accept()
		if err != nil {
			fmt.Println("accept error:", err)
			continue
		}
		splitAddressPort := strings.Split((c.RemoteAddr().String()), ":")
		fromAddressString, fromPortString := splitAddressPort[0], splitAddressPort[1]
		fromPort, err := strconv.Atoi(fromPortString)
		if err != nil {
			fmt.Println("acceptLoop: atoi error:", err)
		}
		go func() {
			conn := NewConnectionState(net.IP(fromAddressString), fromPort, "", s.serverId, s.infoHash, &c)
			fmt.Printf("established connection with %s\n", c.RemoteAddr())
			peerId, err := conn.TwoWayHandshake("BitTorrent protocol", false)
			conn.PeerId = peerId
			//peer.Peer.
			//Handle Peers connecting to us
			if err == nil {
				go func() {
					err := s.ReadLoop(conn)
					if err != nil {
						s.DisconnectCh <- conn.PeerId
					}
				}()
			}

		}()
	}

}

// Reads packets received from the given connection
// Writes the packet + metadata (based on Packet struct) to the msg channel
func (s *Server) ReadLoop(conn *ConnectionState) error {
	c := bufio.NewReader(*conn.Conn)
	defer (*conn.Conn).Close()
	for {
		msgLenBuf := make([]byte, 4)
		msgLenN, err := c.Read(msgLenBuf)
		//fmt.Printf("%v\n", msgLenBuf)
		if msgLenN != 4 || err == io.EOF {
			fmt.Printf("ip: %s, readloop: read error: %s\n", conn.PeerAddress, err)
			return err
		}
		msgLen := binary.BigEndian.Uint32(msgLenBuf)
		var buf []byte
		if msgLen != 0 {
			payloadBuf := make([]byte, msgLen)
			n, err := io.ReadFull(c, payloadBuf)
			fmt.Printf("read %d bytes\n", n)
			if err == io.EOF {
				fmt.Println("message payload: read error:", err)
				return err
			}
			//fmt.Printf("read %d bytes from %s\n", payloadLen, conn.PeerAddress)
			buf = append(msgLenBuf, payloadBuf...)
		} else {
			buf = msgLenBuf
		}

		s.MsgCh <- *(Packet{
			FromAddress: conn.PeerAddress,
			FromPort:    conn.PeerPort,
			FromId:      conn.PeerId,
			Payload:     buf[:],
		}.PacketPayloadToMessage())
	}
}

// This assumes bittorrent payload
// with format: <msg length : 4 bytes><msg type : 1 byte><payload : 0 - n bytes>
func (p Packet) PacketPayloadToMessage() *Message {
	payload := p.Payload
	msgLength := binary.BigEndian.Uint32(payload[:4])
	var msgType MsgType
	var msgPayload []byte
	//Message with no type byte (keep-alive message)
	if msgLength == 0 {
		msgType = KeepAlive
		msgPayload = nil
	} else {
		msgType = MsgType(payload[4])
		//fmt.Printf("peer: %s, len: %d, msgType: %d\n", p.FromAddress, msgLength, msgType)
		//Message with no payload
		if len(payload) <= 5 {
			msgPayload = nil
		} else {
			msgPayload = payload[5:]
		}
	}
	return &Message{
		PeerAddress: p.FromAddress,
		PeerPort:    p.FromPort,
		PeerId:      p.FromId,
		MsgLength:   msgLength,
		MsgType:     msgType,
		Payload:     msgPayload,
	}
}

func (conn *ConnectionState) SendChoke() {
	var chokeMsg Message
	chokeMsg.MsgLength = 1
	chokeMsg.MsgType = 0
	conn.sendMessage(chokeMsg)
}

func (conn *ConnectionState) SendUnchoke() {
	var unchokeMsg Message
	fmt.Printf("abc\n")
	unchokeMsg.MsgLength = 1
	unchokeMsg.MsgType = 1
	conn.sendMessage(unchokeMsg)
}

func (conn *ConnectionState) SendInterested() {
	var interestedMsg Message
	interestedMsg.MsgLength = 1
	interestedMsg.MsgType = 2
	conn.sendMessage(interestedMsg)
}

func (conn *ConnectionState) SendNotInterested() {
	var notInterestedMsg Message
	notInterestedMsg.MsgLength = 1
	notInterestedMsg.MsgType = 3
	conn.sendMessage(notInterestedMsg)
}

func (conn *ConnectionState) SendPiece(pieceIdx uint32, start_offset uint32, block []byte, n int) {
	var pieceMsg Message
	pieceMsg.MsgLength = 9 + uint32(n)
	pieceMsg.MsgType = 7
	pieceMsg.Payload = block
	conn.sendMessage(pieceMsg)
}

func (conn *ConnectionState) SendRequest(pieceIdx uint32, start_offset uint32, n uint32) {
	var requestMsg Message
	requestMsg.MsgLength = 13
	requestMsg.MsgType = 6
	pieceIdxBuf := make([]byte, 4)
	startOffsetBuf := make([]byte, 4)
	nBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(pieceIdxBuf, pieceIdx)
	binary.BigEndian.PutUint32(startOffsetBuf, start_offset)
	binary.BigEndian.PutUint32(nBuf, n)
	requestMsg.Payload = append(pieceIdxBuf, append(startOffsetBuf, nBuf...)...)
	// fmt.Printf("request payload: %v\n", requestMsg.Payload)
	conn.sendMessage(requestMsg)
}

func (conn *ConnectionState) SendHave(pieceIdx int) {
	var haveMsg Message
	haveMsg.MsgLength = 5
	haveMsg.MsgType = 4
	haveMsg.Payload = make([]byte, 4)
	binary.BigEndian.PutUint32(haveMsg.Payload, uint32(pieceIdx))
	conn.sendMessage(haveMsg)
}

func (conn *ConnectionState) SendBitfield(compressedBitfield []byte, n int) {
	var bitfieldMsg Message
	bitfieldMsg.MsgLength = uint32(1 + n)
	bitfieldMsg.MsgType = 5
	bitfieldMsg.Payload = compressedBitfield
	conn.sendMessage(bitfieldMsg)
}

func (conn *ConnectionState) SendKeepAlive() {
	msgBytes := []byte{0, 0, 0, 0}
	_, err := (*conn.Conn).Write(msgBytes)
	if err != nil {
		fmt.Println("SendKeepAlive: Write error", err)
	}
}

func (conn *ConnectionState) SendCancel(pieceIdx uint32, start_offset uint32, n uint32) {
	var cancelMsg Message
	cancelMsg.MsgLength = 13
	cancelMsg.MsgType = 8
	pieceIdxBuf := make([]byte, 4)
	startOffsetBuf := make([]byte, 4)
	nBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(pieceIdxBuf, pieceIdx)
	binary.BigEndian.PutUint32(startOffsetBuf, start_offset)
	binary.BigEndian.PutUint32(nBuf, uint32(n))
	cancelMsg.Payload = append(pieceIdxBuf, append(startOffsetBuf, nBuf...)...)
	conn.sendMessage(cancelMsg)
}

func (conn *ConnectionState) sendMessage(msg Message) {
	msgBytes := make([]byte, 5)
	binary.BigEndian.PutUint32(msgBytes, msg.MsgLength)
	//fmt.Printf("sending msg type: %d\n", msg.MsgType)
	msgBytes[4] = byte(msg.MsgType)
	msgBytes = append(msgBytes, msg.Payload...)
	_, err := (*conn.Conn).Write(msgBytes)
	if err != nil {
		fmt.Println("sendMessage: fprintf error.")
	}
}

// Handles two-way handshake.
// If initiator is true, we send the first handshake and then wait for a handshake.
// Otherwise, if we aren't the connection initiator, we wait for a handshake from the other side
// before sending a handshake back.
func (conn *ConnectionState) TwoWayHandshake(pstr string, initiator bool) (string, error) {
	sendHandshake := func() error {
		_, err := fmt.Fprintf(*conn.Conn, "\x13BitTorrent protocol\x00\x00\x00\x00\x00\x00\x00\x00%s%s",
			conn.infoHash, conn.clientId)
		if err != nil {
			fmt.Println("sendHandshake: fprintf error:", err)
			return err
		}
		return nil
	}
	receiveHandshake := func() (string, error) {
		pstrlen := make([]byte, 1)
		_, err := (*conn.Conn).Read(pstrlen)
		if err != nil {
			//fmt.Printf("ip: %s", conn.PeerAddress)
			fmt.Printf("ip: %s, receiveHandshake: read error: %s\n", conn.PeerAddress, err)
			return "", err
		}
		//-1 since we've the read single pstrlen byte
		handshakePacketSize := 49 + uint8(pstrlen[0]) - 1
		buf := make([]byte, handshakePacketSize)
		_, err = (*conn.Conn).Read(buf)
		if err != nil {
			//fmt.Printf("ip: %s", conn.PeerAddress)
			//fmt.Printf("%v", buf)
			//fmt.Println("receiveHandshake: read error:", err)
			return "", err
		}
		//fmt.Printf("handshake bytes: %d\n", n)
		//TODO: check if info_hash is the same.
		//infoHash := buf[uint8(pstrlen[0])+8 : uint(pstrlen[0])+8+20]
		//fmt.Printf("%v\n", infoHash)
		peerId := buf[uint8(pstrlen[0])+8+20 : uint(pstrlen[0])+8+40]
		//fmt.Printf("%s\n", peerId)
		return string(peerId), nil
	}
	var peerId string
	if initiator {
		err := sendHandshake()
		if err != nil {
			return "", err
		}
		tmp, err := receiveHandshake()
		if err != nil {
			return "", err
		}
		peerId = tmp
	} else {
		tmp, err := receiveHandshake()
		if err != nil {
			return "", err
		}
		peerId = tmp
		err = sendHandshake()
		if err != nil {
			return "", err
		}
	}
	return peerId, nil
}
