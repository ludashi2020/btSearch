package common

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/Unknwon/goconfig"
	reuse "github.com/libp2p/go-reuseport"
	"github.com/paulbellamy/ratecounter"
	"golang.org/x/time/rate"
)

func checkErr(err error) {
	if err != nil {
		panic(err.Error())
	}
}
func init() {
	confPath := flag.String("c", "config/worker.conf", "worker config file")

	flag.Parse()

	config, err := goconfig.LoadConfigFile(*confPath)
	if err != nil {
		fmt.Println("Config file not exist")
		os.Exit(-1)
	}
	cfg = config
	listenerAddr, err = cfg.GetValue("worker", "listenerAddr")
	checkErr(err)
	findNodeSpeedTmp, err := cfg.GetValue("worker", "findNodeSpeed")
	checkErr(err)
	findNodeSpeed, err = strconv.Atoi(findNodeSpeedTmp)
	checkErr(err)
	findNodeSpeedLimiter = rate.NewLimiter(rate.Every(time.Second/time.Duration(findNodeSpeed)), findNodeSpeed)
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

func NewServer() *Worker {
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
	}
}
