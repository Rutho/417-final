// Package server provides a way to initialize and start a TCP server
// That listens and accepts connections, and sends received messages to the message channel
package comms

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
)

type Server struct {
	localAddress net.IP
	localPort    int
	ln           net.Listener
	quitCh       chan struct{}
	MsgCh        chan Message
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
		MsgCh:        make(chan Message, 100),
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
func (s *Server) EstablishConnection(conn *ConnectionState) error {
	c, err := net.Dial("tcp", fmt.Sprintf("%s:%d", conn.PeerAddress, conn.PeerPort))
	if err != nil {
		fmt.Println("EstablishConnection: dial error:", err)
		return err
	}

	conn.Conn = &c

	conn.TwoWayHandshake("BitTorrent protocol", true)
	//Check if handshake went well. Else, error stuff happens
	//If it went well, send connection state to connCh and let another goroutine handle processing that
	go func() {
		s.readLoop(conn)
	}()
	return nil
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
			s.readLoop(conn)
		}()
	}

}

// Reads packets received from the given connection
// Writes the packet + metadata (based on Packet struct) to the msg channel
func (s *Server) readLoop(conn *ConnectionState) error {
	for {
		buf := make([]byte, int32(math.Pow(2, 14)))
		n, err := (*conn.Conn).Read(buf)
		if err != nil {
			fmt.Printf("ip: %s, readloop: read error: %s\n", conn.PeerAddress, err)
			(*conn.Conn).Close()
			return err
		}

		s.MsgCh <- *(Packet{
			FromAddress: conn.PeerAddress,
			FromPort:    conn.PeerPort,
			FromId:      conn.PeerId,
			Payload:     buf[:n],
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
	if len(payload) <= 4 {
		msgType = KeepAlive
		msgPayload = nil
	} else {
		msgType = MsgType(payload[4])
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

func (conn *ConnectionState) SendRequest(pieceIdx uint32, start_offset uint32, n int) {
	var requestMsg Message
	requestMsg.MsgLength = 13
	requestMsg.MsgType = 6
	requestMsg.Payload = make([]byte, 4)
	binary.BigEndian.PutUint32(requestMsg.Payload, uint32(n))
	conn.sendMessage(requestMsg)
}

func (conn *ConnectionState) SendHave(pieceIdx byte) {
	var haveMsg Message
	haveMsg.MsgLength = 5
	haveMsg.MsgType = 4
	haveMsg.Payload = make([]byte, 1)
	haveMsg.Payload[0] = pieceIdx
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

func (conn *ConnectionState) SendCancel(pieceIdx uint32, start_offset uint32, n int) {
	var cancelMsg Message
	cancelMsg.MsgLength = 13
	cancelMsg.MsgType = 8
	cancelMsg.Payload = make([]byte, 4)
	binary.BigEndian.PutUint32(cancelMsg.Payload, uint32(n))
	conn.sendMessage(cancelMsg)
}

func (conn *ConnectionState) sendMessage(msg Message) {
	msgBytes := make([]byte, 4+msg.MsgLength)
	binary.BigEndian.PutUint32(msgBytes, msg.MsgLength)
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
func (conn *ConnectionState) TwoWayHandshake(pstr string, initiator bool) error {
	sendHandshake := func() error {
		_, err := fmt.Fprintf(*conn.Conn, "\x13BitTorrent protocol\x00\x00\x00\x00\x00\x00\x00\x00%s%s",
			conn.infoHash, conn.clientId)
		if err != nil {
			fmt.Println("sendHandshake: fprintf error:", err)
			return err
		}
		return nil
	}
	receiveHandshake := func() error {
		pstrlen := make([]byte, 1)
		_, err := (*conn.Conn).Read(pstrlen)
		if err != nil {
			fmt.Printf("ip: %s", conn.PeerAddress)
			fmt.Println("receiveHandshake: read error:", err)
			return err
		}
		//-1 since we've the read single pstrlen byte
		handshakePacketSize := 49 + uint8(pstrlen[0]) - 1
		buf := make([]byte, handshakePacketSize)
		_, err = (*conn.Conn).Read(buf)
		//TODO: check if info_hash is the same.

		if err != nil {
			fmt.Printf("ip: %s", conn.PeerAddress)
			fmt.Printf("%v", buf)
			fmt.Println("receiveHandshake: read error:", err)
			return err
		}
		return nil
	}
	if initiator {
		err := sendHandshake()
		if err != nil {
			return err
		}
		err = receiveHandshake()
		if err != nil {
			return err
		}
	} else {
		err := receiveHandshake()
		if err != nil {
			return err
		}
		err = sendHandshake()
		if err != nil {
			return err
		}
	}
	return nil
}
