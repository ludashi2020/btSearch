package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

func init() {
	go http.ListenAndServe("0.0.0.0:6060", nil)
}

var perDataSize = 50
var maxThreadNum = 5

var (
	esURL           = ""
	mongoAddr       = ""
	mongoDataBase   = ""
	mongoCollection = ""
	mongoUsername   = ""
	mongoPassword   = ""
)

type monServer struct {
	printChan chan string
	Client    http.Client
	Session   *mgo.Session
	Data      chan []map[string]interface{}
	wg        *sync.WaitGroup
	queue     chan int
}

type esData struct {
	Title      string `json:"title"`
	HashId     string `json:"hash_id"`
	Length     int64  `json:"length"`
	CreateTime int64  `json:"create_time"`
	FileType   string `json:"file_type"`
	Hot        int    `json:"hot"`
}

func newMon() *monServer {
	client := http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        20,
			MaxIdleConnsPerHost: 20,
			// Dial: func(netw, addr string) (net.Conn, error) {
			// 	deadline := time.Now().Add(1 * time.Second)
			// 	c, err := net.DialTimeout(netw, addr, time.Second*1)
			// 	if err != nil {
			// 		return nil, err
			// 	}
			// 	c.SetDeadline(deadline)
			// 	return c, nil
			// },
			//DisableKeepAlives: false,
		},
	}
	dialInfo := &mgo.DialInfo{
		Addrs:  []string{mongoAddr},
		Direct: false,
		//Timeout: time.Second * 1,
		Database: mongoDataBase,
		Source:   mongoCollection,
		Username: mongoUsername,
		Password: mongoPassword,
		//PoolLimit: 4096, // Session.SetPoolLimit
	}

	session, err := mgo.DialWithInfo(dialInfo)
	if err != nil {
		panic(err.Error())
	}
	session.SetPoolLimit(40)
	session.SetMode(mgo.Monotonic, true)
	return &monServer{
		printChan: make(chan string, 5),
		Client:    client,
		Session:   session,
		Data:      make(chan []map[string]interface{}, 1000),
		wg:        &sync.WaitGroup{},
		queue:     make(chan int, 20),
	}
}

func (m *monServer) getdata(objectid bson.ObjectId) {

	//data := bson.M{"hot": 100}
	c := m.Session.DB(mongoCollection).C("torrent")
	for {

		data := make([]map[string]interface{}, perDataSize)
		selector := bson.M{"_id": map[string]bson.ObjectId{"$gt": objectid}}
		c.Find(selector).Limit(perDataSize).All(&data)
		// for _, i := range data {
		// 	m.printChan <- (i["_id"])

		// }
		m.Data <- data
		//m.printChan <- (len(data))
		if size := len(data); size == perDataSize {
			objectid = data[size-1]["_id"].(bson.ObjectId)
		} else {
			m.printChan <- "Done!!!"
			break
		}

	}

}

func (m *monServer) Add(delta int) {
	for i := 0; i < delta; i++ {
		m.queue <- 1
	}
	for i := 0; i > delta; i-- {
		<-m.queue
	}
	m.wg.Add(delta)
}

func (m *monServer) Done() {
	<-m.queue
	m.wg.Done()
}

func (m *monServer) Wait() {
	m.wg.Wait()
}
func (m *monServer) Put(url string, data []byte, pid int, maxThread chan int) (err error) {
	m.Add(1)
	req, err := http.NewRequest("PUT", url, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	resp, err := m.Client.Do(req)
	if err != nil {
		m.Done()
		m.printChan <- "Try Again Error:" + err.Error()
		return m.Put(url, data, pid, maxThread)
	}
	io.Copy(ioutil.Discard, resp.Body)
	//m.printChan <- (string(body))
	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		resp.Body.Close()
		m.Done()
		// handle error
		m.printChan <- fmt.Sprint(resp.StatusCode) + "Try Again"

		return m.Put(url, data, pid, maxThread)
	}
	resp.Body.Close()
	//m.printChan <- (string(body))
	maxThread <- pid
	m.Done()
	return err

}

func (m *monServer) sync() {

	//var num int64 = 0

	maxThread := make(chan int, maxThreadNum)

	for i := 1; i <= maxThreadNum; i++ {
		maxThread <- i

	}

	for i := range m.Data {
		m.parserData(i, maxThread)
		m.printChan <- "-----------------------------------------------"
	}
}

func (m *monServer) parserData(data []map[string]interface{}, maxThread chan int) {
	for _, one := range data {

		if _, ok := one["name"]; !ok {
			m.printChan <- "Name Err"
			return
		}
		if _, ok := one["_id"]; !ok {
			m.printChan <- "_id Err"
			return
		}

		if _, ok := one["length"]; !ok {
			m.printChan <- "length Err"
			return
		}
		if _, ok := one["create_time"]; !ok {
			m.printChan <- "create_time Err"
			return
		}
		if _, ok := one["category"]; !ok {
			m.printChan <- "Category Err"
			return
		}
		if _, ok := one["hot"]; !ok {
			m.printChan <- "Hot Err"
			return
		}
		objectId := one["_id"].(bson.ObjectId).Hex()
		syncdata, err := json.Marshal(esData{
			Title:      one["name"].(string),
			HashId:     one["infohash"].(string),
			Length:     one["length"].(int64),
			CreateTime: one["create_time"].(int64),
			FileType:   strings.ToLower(one["category"].(string)),
			Hot:        one["hot"].(int),
		})
		if err != nil {
			return
		}
		pid := <-maxThread
		m.printChan <- "PID:" + strconv.Itoa(pid) + "----" + "---" + one["name"].(string) + "------" + one["_id"].(bson.ObjectId).Hex()
		m.Put(esURL+objectId, syncdata, pid, maxThread)
	}
}

func (m *monServer) PrintLog() {

	for {
		fmt.Println(<-m.printChan)
	}

}

func (m *monServer) run(objectId ...string) (data map[string]interface{}) {
	m.printChan <- "Runing..."
	c := m.Session.DB(mongoCollection).C("torrent")
	//selector := bson.M{} //从0开始
	if len(objectId) == 1 {
		selector := bson.M{"_id": bson.ObjectIdHex(objectId[0])}
		c.Find(selector).Sort("_id").Limit(1).One(&data)
	} else if len(objectId) == 0 {
		c.Find(nil).Limit(1).One(&data)
	} else {
		panic("objectId error")
	}
	if data == nil {
		log.Fatalln("have no such ObjectID")
		return
	}
	log.Println("ObjectID Start with", data["_id"])
	maxThread := make(chan int, 1)
	maxThread <- 1
	m.parserData([]map[string]interface{}{data}, maxThread)
	return data
}

func main() {
	objectID := flag.String("id", "", "start with object id")
	esURLTmp := flag.String("esURL", "", "esURL")                //"http://127.0.0.1:9200/bavbt/torrent/"
	mongoAddrTmp := flag.String("mongoAddr", "", "mongoAddr")    //"127.0.0.1:27017"
	mongoDataBaseTmp := flag.String("mongoDataBase", "", "")     //"bavbt"
	mongoCollectionTmp := flag.String("mongoCollection", "", "") //"torrent"
	mongoUsernameTmp := flag.String("mongoUsername", "", "")
	mongoPasswordTmp := flag.String("mongoPassword", "", "")
	flag.Parse()
	esURL = *esURLTmp
	mongoAddr = *mongoAddrTmp
	mongoDataBase = *mongoDataBaseTmp
	mongoCollection = *mongoCollectionTmp
	mongoUsername = *mongoUsernameTmp
	mongoPassword = *mongoPasswordTmp
	fmt.Println("objectID", objectID)
	fmt.Println("esURL", esURL)
	fmt.Println("mongoAddr", mongoAddr)
	fmt.Println("mongoDataBase", mongoDataBase)
	fmt.Println("mongoCollection", mongoCollection)
	fmt.Println("mongoUsername", mongoUsername)
	fmt.Println("mongoPassword", mongoPassword)
	runtime.GOMAXPROCS(runtime.NumCPU())
	m := newMon()
	defer m.Session.Close()
	go m.PrintLog()
	entry := map[string]interface{}{}
	if *objectID == "" {
		entry = m.run()
	} else {
		entry = m.run(*objectID)
	}
	go m.getdata(entry["_id"].(bson.ObjectId))
	m.sync()

}
