package common

import (
	"context"
	"github.com/Bmixo/btSearch/api/api_server_1/torrent"
	"log"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

type Link struct {
	Conn         *grpc.ClientConn
	Addr         string
	LinkPostChan chan torrent.TData
}

type Tool struct {
	torrent.RPCServer
	Links        []Link
	ToolSendChan chan torrent.TData
	ToolRevChan  chan torrent.TData
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
		ToolSendChan: make(chan torrent.TData, 1000),
		ToolRevChan:  make(chan torrent.TData, 1000),
		Links:        make([]Link, 0),
	}
}

func (m *Tool) Connect(i int) {
	if m.Links[i].Conn == nil {
	reconnect:
		var err error
		log.Printf("on connect: [%v]\n", m.Links[i].Addr)
		m.Links[i].Conn, err = grpc.Dial(m.Links[i].Addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(time.Second*5))
		if err != nil || m.Links[i].Conn == nil {
			log.Printf("connect fail: [%v]\n", m.Links[i].Addr)
			time.Sleep(time.Millisecond * 200)
			goto reconnect
		}
		log.Printf("connect success: [%v]\n", m.Links[i].Addr)
		client := torrent.NewRPCClient(m.Links[i].Conn)
		ctx := context.Background()
		stream, err := client.MessageStream(ctx)
		if err != nil {
			m.Links[i].Conn = nil
			//m.Links[i].Conn.Close()
			goto reconnect
		}
		if err := stream.Send(&torrent.Message{
			Code:   torrent.Code_CodeVerifyPassWord,
			Verify: &torrent.Verify{Password: verifyPassord},
		}); err != nil {
			m.Links[i].Conn = nil
			goto reconnect
		}
		go func() {
			for {
				tDataTmp := <-m.ToolSendChan
				if stream == nil {
					return
				}
				err := stream.Send(&torrent.Message{
					Code:  torrent.Code_CodeMessageTData,
					TData: &tDataTmp,
				})
				if err != nil {
					stream = nil
				}
			}
		}()
		for {
			if stream == nil {
				goto reconnect
			}
			data, err := stream.Recv()
			if err != nil {
				stream = nil
				goto reconnect
			}
			if len(m.ToolRevChan) != cap(m.ToolRevChan) {
				m.ToolRevChan <- *data.TData
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

func (m *Tool) SendData(postData torrent.TData) string {
	for i := 0; i < len(m.Links); i++ {
		if m.Links[i].LinkPostChan == nil {
			continue
		}
		m.Links[i].LinkPostChan <- postData
	}
	return ""
}

func (m *Tool) MessageStream(stream torrent.RPC_MessageStreamServer) error {

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
		if verify.Code != torrent.Code_CodeVerifyPassWord {
			log.Println("\npassword error\n")
			return ctx.Err()
		}
		if verify.Verify == nil || verify.Verify.Password != verifyPassord {
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

		go func() {
			for {
				if stream == nil {
					log.Println("exit recv")
					return
				}
				messageRev, err := stream.Recv()
				if err != nil {
					log.Println("\n" + ctx.Err().Error() + "\n")
					return
				}
				if messageRev.Code != torrent.Code_CodeMessageTData || messageRev.TData == nil {
					log.Println("message error")
					continue
				}
				m.ToolRevChan <- *messageRev.TData
			}
		}()
		for {
			data := <-m.ToolSendChan
			err := stream.Send(&torrent.Message{
				Code:  torrent.Code_CodeMessageTData,
				TData: &data,
			})
			if err != nil {
				stream = nil
				log.Println("\n" + ctx.Err().Error() + "\n")
				return ctx.Err()
			}
		}
	}
}

func (m *Tool) ToolServer(toolServer *Tool) {
	server := grpc.NewServer()
	torrent.RegisterRPCServer(server, toolServer)
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
