// Package server provides a way to initialize and start a TCP server
// That listens and accepts connections, and sends received messages to the message channel
package server

import (
	"fmt"
	"math"
	"net"
)

type Server struct {
	localAddress string
	ln           net.Listener
	quitch       chan struct{}
	MsgCh        chan Packet
}

type Packet struct {
	FromAddress string
	Payload     []byte
}

// Initializes a server instance and returns a pointer to it
func New(localAddress string) *Server {
	return &Server{
		localAddress: localAddress,
		quitch:       make(chan struct{}),
		MsgCh:        make(chan Packet, 10),
	}
}

// Start starts the server instance, making it listen for and accept connections.
// It'll receive packets from the connections and write them to the channel
func (s *Server) Start() {
	ln, err := net.Listen("tcp", "")
	if err != nil {
		fmt.Println("listen error:", err)
	}
	defer ln.Close()
	s.ln = ln

	fmt.Printf("Listening on %s\n", s.localAddress)

	go s.acceptLoop()

	<-s.quitch
	close(s.MsgCh)

	return
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			fmt.Println("accept error:", err)
			continue
		}

		fmt.Printf("established connection with %s\n", conn.RemoteAddr())

		go s.readLoop(conn)
	}

}

// Reads packets received from the given connection
// Writes the packet + metadata (based on Packet struct) to the msg channel
func (s *Server) readLoop(conn net.Conn) {
	buf := make([]byte, int32(math.Pow(2, 15)))
	for {
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Println("read error:", err)
			continue
		}
		s.MsgCh <- Packet{
			FromAddress: conn.RemoteAddr().String(),
			Payload:     buf[:n],
		}
	}
}
