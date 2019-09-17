package common

import (
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/Unknwon/goconfig"
	mapset "github.com/deckarep/golang-set"
	"github.com/go-ego/gse"
	mgo "gopkg.in/mgo.v2"
)

func checkErr(err error) {
	if err != nil {
		panic(err.Error())
	}
}
func init() {
	confPath := flag.String("c", "config/server.conf", "web server config file")

	flag.Parse()

	config, err := goconfig.LoadConfigFile(*confPath)
	if err != nil {
		fmt.Println("Config file not exist")
		os.Exit(-1)
	}
	cfg = config
	mongoAddr, err = cfg.GetValue("mongodb", "addr")
	checkErr(err)
	dataBase, err = cfg.GetValue("mongodb", "database")
	checkErr(err)
	collection, err = cfg.GetValue("mongodb", "collection")
	checkErr(err)
	mongoUsername, err = cfg.GetValue("mongodb", "musername")
	checkErr(err)
	mongoPassWord, err = cfg.GetValue("mongodb", "mpassword")
	checkErr(err)
	handshakePassword, err = cfg.GetValue("server", "handshakePassword")
	checkErr(err)
	wkServer, err = cfg.GetValue("server", "wkServer")
	checkErr(err)
	banList, err = cfg.GetValue("server", "banList")
	checkErr(err)

}
func NewSniffer() *sn {
	dialInfo := &mgo.DialInfo{
		Addrs:  []string{mongoAddr},
		Direct: false,
		//Timeout: time.Second * 1,
		Database:  dataBase,
		Source:    collection,
		Username:  mongoUsername,
		Password:  mongoPassWord,
		PoolLimit: 4096, // Session.SetPoolLimit
	}

	session, err := mgo.DialWithInfo(dialInfo)
	session.SetPoolLimit(mongoConnectLimitNum)
	session.SetMode(mgo.Monotonic, true)
	if err != nil {
		panic(err.Error)
	}

	var segmenter gse.Segmenter
	segmenter.LoadDict()

	return &sn{
		segmenter:     segmenter, //分词
		printChan:     make(chan string, 5),
		tdataChan:     make(chan tdata, tdataChanSize),
		hashList:      mapset.NewSet(),
		blackAddrList: mapset.NewSet(),
		Conn:          *new(net.Conn),
		Server:        wkServer,
		Mon:           session,
		mongoLimit:    make(chan bool, mongoConnectLimitNum),
		blackList:     loadBlackList(),
	}
}
