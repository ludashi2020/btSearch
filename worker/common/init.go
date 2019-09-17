package common

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/Unknwon/goconfig"

	mapset "github.com/deckarep/golang-set"
)

func checkErr(err error) {
	if err != nil {
		panic(err.Error())
	}
}
func init() {
	confPath := flag.String("c", "config/server.conf", "worker config file")

	flag.Parse()

	config, err := goconfig.LoadConfigFile(*confPath)
	if err != nil {
		fmt.Println("Config file not exist")
		os.Exit(-1)
	}
	cfg = config
	listenerAddr, err = cfg.GetValue("worker", "listenerAddr")
	checkErr(err)
	maxNodeQsizeTmp, err := cfg.GetValue("worker", "maxNodeQsize")
	checkErr(err)
	maxNodeQsize, err = strconv.Atoi(maxNodeQsizeTmp)
	checkErr(err)
	nodeChanSizeTmp, err := cfg.GetValue("worker", "nodeChanSize")
	checkErr(err)
	nodeChanSize, err = strconv.Atoi(nodeChanSizeTmp)
	checkErr(err)
	udpPortTmp, err := cfg.GetValue("worker", "udpPort")
	checkErr(err)
	udpPort, err = strconv.Atoi(udpPortTmp)
	checkErr(err)
	verifyPassord, err = cfg.GetValue("worker", "verifyPassord")
	checkErr(err)

}

func NewServer() *wkServer {
	tcplistener, err := net.Listen("tcp", listenerAddr)
	if err != nil {
		panic(err.Error())
	}
	udpAddr, err := net.ResolveUDPAddr("udp4", ":"+strconv.Itoa(udpPort))
	if err != nil {
		panic(err.Error())
	}

	udplistener, err := net.ListenUDP("udp4", udpAddr)
	if err != nil {
		panic(err.Error())
	}
	return &wkServer{
		tcpListener: tcplistener,
		num:         0,
		revNum:      0,
		dropNum:     0,
		findNodeNum: 0,
		udpListener: udplistener,
		localID:     string(randBytes(20)),
		// nodeChan:    make(chan *node, nodeChanSize),
		node:      mapset.NewSet(),
		nodes:     "",
		kbucket:   []*node{},
		printChan: make(chan string, 5),
		dataChan:  make(chan tdata, 5),
	}
}
