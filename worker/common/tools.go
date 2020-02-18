package common

import (
	"context"
	"log"
	"net"
	"sync"
	"time"

	"github.com/Bmixo/btSearch/header"
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

func NewTool() *Tool {
	return &Tool{
		ToolPostChan: make(chan header.Tdata, 1000),
		Links:        make([]Link, 0),
	}

}
func (self *Tool) Connect(i int) {
	if self.Links[i].Conn == nil {
	reconnect:
		var err error
		log.Printf("\non connect: [%v]\n", self.Links[i].Addr)
		self.Links[i].Conn, err = grpc.Dial(self.Links[i].Addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(time.Second*3))
		if err != nil || self.Links[i].Conn == nil {
			log.Printf("\nconnect fail: [%v]\n", self.Links[i].Addr)
			time.Sleep(time.Millisecond * 200)
			goto reconnect
		}
		log.Printf("\nconnect success: [%v]\n", self.Links[i].Addr)
		client := header.NewRPCClient(self.Links[i].Conn)
		ctx := context.Background()
		stream, err := client.Communite(ctx)
		if err != nil {
			self.Links[i].Conn = nil
			//self.Links[i].Conn.Close()
			goto reconnect
		}
		if err := stream.Send(&header.Verify{Password: verifyPassord}); err != nil {
			self.Links[i].Conn = nil
			goto reconnect
		}
		for {
			data, err := stream.Recv()
			if err != nil {
				goto reconnect
			}
			if len(self.ToolPostChan) != cap(self.ToolPostChan) {
				self.ToolPostChan <- *data
			}

		}

	}
}
func (self *Tool) LinksServe() {
	for i := 0; i < len(self.Links); i++ {
		go self.Connect(i)
	}
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

func NewClientCollection() *ClientCollection {
	return &ClientCollection{
		clients: make(map[uint64]*Client),
	}
}

var wsCollection = NewClientCollection()

func (self *Tool) SendData(postData header.Tdata) string {
	if self == nil {
		return ""
	}

	for i := 0; i < len(self.Links); i++ {
		if self.Links[i].LinkPostChan == nil {
			continue
		}
		self.Links[i].LinkPostChan <- postData
	}
	return ""
}

func (self *Tool) Communite(stream header.RPC_CommuniteServer) error {

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
			data := <-self.ToolPostChan
			err := stream.Send(&data)
			if err != nil {
				log.Println("\n" + ctx.Err().Error() + "\n")
				return ctx.Err()
			}

		}
	}
}
func (self *Tool) ToolServer(toolServer *Tool) {
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
