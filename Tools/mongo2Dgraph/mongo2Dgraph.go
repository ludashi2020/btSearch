package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/dgraph-io/dgo/v2"
	"github.com/dgraph-io/dgo/v2/protos/api"
	"github.com/golang/glog"
	"go.opencensus.io/plugin/ocgrpc"
	"google.golang.org/grpc"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	perDataSize        = 20
	instancesNum       = 15
	maxCallRecvMsgSize = 9999999
	maxCallSendMsgSize = 9999999
	dgoAddr            = "127.0.0.1:9080"
	mongoAddr          = "127.0.0.1:27017"
	dataBase           = "bavbt"
	collection         = "torrent"
)

type monServer struct {
	printChan    chan string
	Session      *mgo.Session
	Data         chan []map[string]interface{}
	wg           *sync.WaitGroup
	queue        chan int
	Count        uint64
	InfoHash     string
	CountChan    chan bool
	InfoHashChan chan string
	loader       *loader
	exit         chan bool
}
type CancelFunc func()

func newMon() *monServer {

	dialInfo := &mgo.DialInfo{
		Addrs:  []string{mongoAddr},
		Direct: false,
		//Timeout: time.Second * 1,
		Database: dataBase,
		Source:   collection,
		// Username:  "root",
		// Password:  "root",
		//PoolLimit: 4096, // Session.SetPoolLimit
	}
	session, err := mgo.DialWithInfo(dialInfo)
	if err != nil {
		panic(err.Error())
	}
	session.SetPoolLimit(40)
	session.SetMode(mgo.Monotonic, true)
	m := monServer{
		printChan:    make(chan string, 5),
		Session:      session,
		Data:         make(chan []map[string]interface{}, 1000),
		wg:           &sync.WaitGroup{},
		queue:        make(chan int, 20),
		Count:        0,
		InfoHash:     "",
		CountChan:    make(chan bool, 1000),
		InfoHashChan: make(chan string, 1000),
		exit:         make(chan bool, 10),
	}
	m.loader = &loader{
		reqs: make(chan api.Request, instancesNum),
	}
	m.loader.requestsWg.Add(instancesNum)
	return &m
}

func (m *monServer) DgChan() {
	ctx := context.Background()

	bmOpts := batchMutationOptions{
		Ctx: ctx,
	}
	m.loader.opts = bmOpts
	dg, closeFunc := GetDgraphClient()
	defer closeFunc()
	m.loader.dc = dg

	for i := 0; i < instancesNum; i++ {
		go m.loader.makeRequests()
	}
	<-m.exit
	close(m.loader.reqs)
	// jsonx, err := json.Marshal(map[string]string{"hahhah": "hahahhshs"})
	// if err != nil {
	// 	panic(err)
	// }
	// mu := request{Mutation: &api.Mutation{SetJson: jsonx}}
	// l.reqs <- mu

	// fmt.Println("done")
	// time.Sleep(10 * time.Second)
	//
	// l.requestsWg.Wait()
}

func SetupConnection(host string) (*grpc.ClientConn, error) {
	callOpts := append([]grpc.CallOption{},
		grpc.MaxCallRecvMsgSize(maxCallRecvMsgSize),
		grpc.MaxCallSendMsgSize(maxCallSendMsgSize))

	dialOpts := append([]grpc.DialOption{},
		grpc.WithStatsHandler(&ocgrpc.ClientHandler{}),
		grpc.WithDefaultCallOptions(callOpts...),
		grpc.WithBlock())

	dialOpts = append(dialOpts, grpc.WithInsecure())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return grpc.DialContext(ctx, host, dialOpts...)
}

type CloseFunc func()

func GetDgraphClient() (*dgo.Dgraph, CloseFunc) {
	var conns []*grpc.ClientConn
	var clients []api.DgraphClient

	for i := 0; i < 10; i++ {
		var conn *grpc.ClientConn
		var err error
		conn, err = SetupConnection("127.0.0.1:9080")
		if err != nil {
			panic(err)
		}
		if conn == nil {
			fmt.Println("Could not setup connection after %d retries")
		}

		conns = append(conns, conn)
		dc := api.NewDgraphClient(conn)
		clients = append(clients, dc)
	}

	dg := dgo.NewDgraphClient(clients...)
	closeFunc := func() {
		for _, c := range conns {
			if err := c.Close(); err != nil {
				glog.Warningf("Error closing connection to Dgraph client: %v", err)
			}
		}
	}
	return dg, closeFunc
}

type batchMutationOptions struct {
	Ctx context.Context
}
type loader struct {
	opts       batchMutationOptions
	dc         *dgo.Dgraph
	requestsWg sync.WaitGroup
	reqs       chan api.Request
}

func (l *loader) makeRequests() {

	for req := range l.reqs {

		txn := l.dc.NewTxn()
		req.CommitNow = true
		_, err := txn.Do(l.opts.Ctx, &req)
		if err != nil {
			log.Println("retry Mutate...", err.Error())
			time.Sleep(time.Second * 3)
			l.reqs <- req
		}

	}

}

func (m *monServer) getdata(objectid bson.ObjectId) {

	//data := bson.M{"hot": 100}
	c := m.Session.DB("bavbt").C("torrent")
	for {

		data := make([]map[string]interface{}, perDataSize)
		selector := bson.M{"_id": map[string]bson.ObjectId{"$gt": objectid}}
		c.Find(selector).Limit(perDataSize).All(&data)
		m.Data <- data
		if size := len(data); size == perDataSize {
			objectid = data[size-1]["_id"].(bson.ObjectId)
		} else {
			m.printChan <- ("Done!!!")
			break
		}

	}

}
func (m *monServer) sync() {

	wg := sync.WaitGroup{}
	for {
		for i := 0; i < instancesNum; i++ {

			data := <-m.Data
			wg.Add(1)
			go func(xd []map[string]interface{}) {
				m.parserData(xd)
				wg.Done()
			}(data)
		}
		wg.Wait()
	}

}

func (m *monServer) parserData(data []map[string]interface{}) {
	// dg, cancel := getDgraphClient()
	// defer cancel()
	// txn := dg.NewTxn()
	// txn := m.dgraphClient.NewTxn()
	// ctx := context.Background()
	mus := []*api.Mutation{}
	for _, one := range data {

		if _, ok := one["name"]; !ok {
			m.printChan <- ("Name Err")
			continue
		}
		if _, ok := one["_id"]; !ok {
			m.printChan <- ("_id Err")
			continue
		}

		if _, ok := one["length"]; !ok {
			m.printChan <- ("length Err")
			continue
		}
		if _, ok := one["create_time"]; !ok {
			m.printChan <- ("create_time Err")
			continue
		}
		if _, ok := one["category"]; !ok {
			m.printChan <- ("Category Err")
			continue
		}
		if _, ok := one["hot"]; !ok {
			m.printChan <- ("Hot Err")
			continue
		}
		id := one["_id"]
		delete(one, "_id")
		pb, err := json.Marshal(one)
		one["_id"] = id
		if err != nil {
			m.printChan <- ("11" + err.Error())
			continue
		}
		mu := &api.Mutation{
			SetJson: pb,
		}
		m.CountChan <- true
		m.InfoHashChan <- one["infohash"].(string)
		// m.loader.reqs <- request{Mutation: mu}
		mus = append(mus, mu)

	}
	m.loader.reqs <- api.Request{Mutations: mus}

	// 	req := &api.Request{CommitNow: true, Mutations: mus}
	// tag:
	// 	_, err := txn.Do(ctx, req)
	// 	if err != nil {
	// 		time.Sleep(time.Second)
	// 		m.printChan <- ("\nretry " + err.Error())
	// 		m.dgraphClient, err = getDgraphClient()
	// 		if err != nil {
	// 			goto tag
	// 		}
	// 		txn = m.dgraphClient.NewTxn()
	// 		ctx = context.Background()
	// 		goto tag
	// 	}
}
func (m *monServer) PrintLog() {
	go func() {
		for <-m.CountChan {
			m.Count++
		}
	}()
	go func() {
		for {
			m.InfoHash = <-m.InfoHashChan
		}
	}()

	go func() {
		tmp := uint64(1)
		for {
			time.Sleep(time.Second)
			if m.Count > 1 {
				tmp++
			}
			fmt.Printf("\r")
			fmt.Printf("Num %v Speed:%v /s Current Hash:%v", m.Count, (m.Count-tmp)/tmp, m.InfoHash)

		}
	}()

	for {
		fmt.Println(<-m.printChan)
	}

}

func (m *monServer) run(objectId ...string) (data map[string]interface{}) {
	m.printChan <- ("Runing...")
	c := m.Session.DB("bavbt").C("torrent")
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
	//中断开始
	m.parserData([]map[string]interface{}{data})
	return data
}

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	objectID := flag.String("id", "", "start with object id")
	flag.Parse()
	m := newMon()
	go m.DgChan()
	defer m.Session.Close()
	go m.PrintLog()
	entry := map[string]interface{}{}
	if *objectID == "" {
		entry = m.run()
	} else {
		entry = m.run(*objectID)
	}
	go m.getdata(entry["_id"].(bson.ObjectId))
	go m.sync()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		fmt.Println()
		fmt.Println("exit num ", m.Count, "exit hash:", m.InfoHash)
		for {
			m.exit <- true
		}
	}()
	<-m.exit

}
