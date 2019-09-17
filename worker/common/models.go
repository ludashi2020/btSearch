package common

import (
	"net"

	"github.com/Unknwon/goconfig"
	mapset "github.com/deckarep/golang-set"
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
	maxNodeQsize  = 145000 //Nanoseconds
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

type wkServer struct {
	tcpListener net.Listener
	client      net.Conn
	revNum      float64
	num         float64
	dropNum     float64
	findNodeNum float64
	udpListener *net.UDPConn
	localID     string
	node        mapset.Set
	kbucket     []*node
	nodes       string
	printChan   chan string
	dataChan    chan tdata
}
