package common

import (
	"context"
	"log"
	"net"
	"sync"
	"time"

	"github.com/Bmixo/btSearch/model"
	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

type Link struct {
	Conn         *grpc.ClientConn
	Addr         string
	LinkPostChan chan header.Tdata
}
type Tool struct {
	Links        []Link
	ToolPostChan chan header.Tdata
}
type Client struct {
	uid     uint64
	conn    *websocket.Conn
	dataCh  chan []byte
	onClose func()
	closed  bool
}

type ClientCollection struct {
	sync.Mutex
	clients map[uint64]*Client
}

func NewTool() *Tool {
	return &Tool{
		ToolPostChan: make(chan header.Tdata, 1000),
		Links:        make([]Link, 0),
	}

}
func (m *Tool) Connect(i int) {
	if m.Links[i].Conn == nil {
	reconnect:
		var err error
		log.Printf("\non connect: [%v]\n", m.Links[i].Addr)
		m.Links[i].Conn, err = grpc.Dial(m.Links[i].Addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(time.Second*3))
		if err != nil || m.Links[i].Conn == nil {
			log.Printf("\nconnect fail: [%v]\n", m.Links[i].Addr)
			time.Sleep(time.Millisecond * 200)
			goto reconnect
		}
		log.Printf("\nconnect success: [%v]\n", m.Links[i].Addr)
		client := header.NewRPCClient(m.Links[i].Conn)
		ctx := context.Background()
		stream, err := client.Communite(ctx)
		if err != nil {
			m.Links[i].Conn = nil
			//m.Links[i].Conn.Close()
			goto reconnect
		}
		if err := stream.Send(&header.Verify{Password: verifyPassord}); err != nil {
			m.Links[i].Conn = nil
			goto reconnect
		}
		for {
			data, err := stream.Recv()
			if err != nil {
				goto reconnect
			}
			if len(m.ToolPostChan) != cap(m.ToolPostChan) {
				m.ToolPostChan <- *data
			}

		}

	}
}

func (m *Tool) LinksServe() {
	for i := 0; i < len(m.Links); i++ {
		go m.Connect(i)
	}
}

func NewClientCollection() *ClientCollection {
	return &ClientCollection{
		clients: make(map[uint64]*Client),
	}
}

var wsCollection = NewClientCollection()

func (m *Tool) SendData(postData header.Tdata) string {
	for i := 0; i < len(m.Links); i++ {
		if m.Links[i].LinkPostChan == nil {
			continue
		}
		m.Links[i].LinkPostChan <- postData
	}
	return ""
}

func (m *Tool) Communite(stream header.RPC_CommuniteServer) error {

	ctx := stream.Context()

	select {
	case <-ctx.Done():
		log.Println("\nWs Exit\n")
		return ctx.Err()
	default:
		verify, err := stream.Recv()
		if err != nil {
			log.Println("\nrev error\n")
			return ctx.Err()
		}
		if verify.Password != verifyPassord {
			log.Println("\npassword error\n")
			return ctx.Err()
		}
		pr, ok := peer.FromContext(ctx)
		if !ok {
			log.Println("\nctxfailed\n")
			return ctx.Err()
		}
		if pr.Addr == net.Addr(nil) {
			log.Println("\nip is nil\n")
			return ctx.Err()
		}
		for {
			data := <-m.ToolPostChan
			err := stream.Send(&data)
			if err != nil {
				log.Println("\n" + ctx.Err().Error() + "\n")
				return ctx.Err()
			}

		}
	}
}
func (m *Tool) ToolServer(toolServer *Tool) {
	server := grpc.NewServer()
	header.RegisterRPCServer(server, toolServer)
	address, err := net.Listen("tcp", listenerAddr)
	if err != nil {
		log.Println("\n" + (err.Error()) + "\n")
		return
	}
	if err := server.Serve(address); err != nil {
		log.Println("\n" + (err.Error()) + "\n")
		return
	}
}
