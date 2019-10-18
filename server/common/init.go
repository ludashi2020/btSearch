package common

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/Unknwon/goconfig"
	mapset "github.com/deckarep/golang-set"
	"github.com/go-ego/gse"
	"github.com/go-redis/redis"
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
	tmp, err := cfg.GetValue("server", "wkNodes")
	checkErr(err)
	wkNodes = strings.Split(tmp, ",")
	for _, j := range wkNodes {

		if _, _, err := net.SplitHostPort(j); err != nil {
			panic("wkNodes set error")
		}
	}
	banList, err = cfg.GetValue("server", "banList")
	checkErr(err)

	if tmp, err = cfg.GetValue("redis", "redisEnable"); tmp == "true" {
		redisEnable = true
		redisAddr, err = cfg.GetValue("redis", "redisAddr")
		checkErr(err)
		redisPassword, err = cfg.GetValue("redis", "redisPassword")
		checkErr(err)
		tmp, err = cfg.GetValue("redis", "redisDB")
		checkErr(err)
		redisDB, err = strconv.Atoi(tmp)
		checkErr(err)
	}
	checkErr(err)

}

//NewSniffer :NewSniffer
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
	var redisClient *redis.Client
	if redisEnable {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     redisAddr,
			Password: redisPassword,
			DB:       redisDB,
		})
		_, err := redisClient.Ping().Result()
		if err != nil {
			panic(err.Error())
		}
	}

	var segmenter gse.Segmenter
	segmenter.LoadDict()

	return &sn{
		segmenter:     segmenter, //分词
		printChan:     make(chan string, 5),
		tdataChan:     make(chan tdata, tdataChanSize),
		hashList:      mapset.NewSet(),
		blackAddrList: mapset.NewSet(),
		Conn:          map[int]net.Conn{},
		Nodes:         wkNodes,
		Mon:           session,
		RedisClient:   redisClient,
		mongoLimit:    make(chan bool, mongoConnectLimitNum),
		blackList:     loadBlackList(),
	}
}
