package service

import (
	"context"
	"github.com/Bmixo/btSearch/api/api_server_1/torrent"
	mapset "github.com/deckarep/golang-set"
	"net"
	"os"
	"strconv"
	"time"

	reuse "github.com/libp2p/go-reuseport"
	"github.com/paulbellamy/ratecounter"
	"golang.org/x/time/rate"
)

func InitWorker() {

	listenerAddr = os.Getenv("listenerAddr")
	findNodeSpeedTmp := os.Getenv("findNodeSpeed")
	var err error
	findNodeSpeed, err = strconv.Atoi(findNodeSpeedTmp)
	checkErr(err)
	findNodeSpeedLimiter = rate.NewLimiter(rate.Every(time.Second/time.Duration(findNodeSpeed)), findNodeSpeed)
	nodeChanSizeTmp := os.Getenv("nodeChanSize")
	nodeChanSize, err = strconv.Atoi(nodeChanSizeTmp)
	checkErr(err)
	udpPortTmp := os.Getenv("udpPort")
	udpPort, err = strconv.Atoi(udpPortTmp)
	checkErr(err)
	verifyPassord = os.Getenv("verifyPassword")

}

func NewWorkerServer() *Worker {
	listenConfig := net.ListenConfig{
		Control: reuse.Control,
	}
	// udpAddr, err := net.ResolveUDPAddr("udp4", ":"+strconv.Itoa(udpPort))
	// if err != nil {
	// 	panic(err.Error())
	// }
	udplistener, err := listenConfig.ListenPacket(context.Background(), "udp", ":"+strconv.Itoa(udpPort)) //端口复用
	// udplistener, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		panic(err.Error())
	}
	return &Worker{
		Tool: *NewTool(),
		count: []Count{
			Count{
				name: "sussNum", rate: ratecounter.NewRateCounter(1 * time.Second),
			},
			Count{
				name: "revNum", rate: ratecounter.NewRateCounter(1 * time.Second),
			},
			Count{
				name: "dropNum", rate: ratecounter.NewRateCounter(1 * time.Second),
			},
			Count{
				name: "DecodeNum", rate: ratecounter.NewRateCounter(1 * time.Second),
			},
			Count{
				name: "findNodeNum", rate: ratecounter.NewRateCounter(1 * time.Second),
			},
		},
		udpListener: udplistener,
		localID:     string(randBytes(20)),
		// node:        mapset.NewSet(),
		nodeChan:    make(chan *node, nodeChanSize),
		nodes:       "",
		kbucket:     []*node{},
		printChan:   make(chan string, 5),
		messageChan: make(chan *message, nodeChanSize),
		dataChan:    make(chan tdata, 5),
		//server
		tdataChan:     make(chan torrent.TData, tdataChanSize),
		hashList:      mapset.NewSet(),
		blackAddrList: mapset.NewSet(),
		sussNum:       ratecounter.NewRateCounter(1 * time.Second),
	}
}
