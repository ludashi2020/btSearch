package common

import (
	"net"

	"github.com/Unknwon/goconfig"
)

type tdata struct {
	Hash   string
	Addr   string
	Offset string
}
type node struct {
	addr string
	id   string
}

var bootstapNodes = []string{
	"9.rarbg.to:2710",
	"9.rarbg.me:2710",
	"open.demonii.com:1337",
	"tracker.opentrackr.org:1337",
	"p4p.arenabg.com:1337",
	"router.utorrent.com:6881",
	"router.bittorrent.com:6881",
	"dht.transmissionbt.com:6881",
}

var (
	listenerAddr  = "0.0.0.0:9898"
	findNodeSpeed = 10000
	nodeChanSize  = 10000
	udpPort       = 6999
	verifyPassord = ""
	cfg           *goconfig.ConfigFile
)

const (
	pingType         = "ping"      //没必要
	findNodeType     = "find_node" //没必要
	getPeersType     = "get_peers"
	announcePeerType = "announce_peer"
	secret           = "IYHJFR%^&IO"
)

type message struct {
	buf  []byte
	addr net.Addr
}
type Worker struct {
	Tool        Tool
	revNum      int
	DecodeNum   int
	sussNum     int
	dropNum     int
	findNodeNum int
	udpListener net.PacketConn
	localID     string
	// node        mapset.Set
	nodeChan    chan *node
	kbucket     []*node
	nodes       string
	printChan   chan string
	messageChan chan *message
	dataChan    chan tdata
}
