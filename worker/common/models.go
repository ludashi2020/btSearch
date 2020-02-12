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
	addr net.UDPAddr
}
type Worker struct {
	Tool        Tool
	revNum      int
	DecodeNum   int
	sussNum     int
	dropNum     int
	findNodeNum int
	udpListener *net.UDPConn
	localID     string
	// node        mapset.Set
	nodeChan    chan *node
	kbucket     []*node
	nodes       string
	printChan   chan string
	messageChan chan *message
	dataChan    chan tdata
}
