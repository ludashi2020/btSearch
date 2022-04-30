package common

import (
	"bytes"
	"crypto/rand"
	"strings"

	randx "math/rand"

	"github.com/Bmixo/btSearch/model"

	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/Bmixo/btSearch/pkg/bencode"
)

func (self *Worker) HandleMsg() {
	for i := 0; i < 20; i++ {
		go self.onMessage()
	}
	for i := 0; i < 10; i++ {

		go func() {
			for {
				buf := make([]byte, 512)
				n, addr, err := self.udpListener.ReadFrom(buf)
				if err != nil {
					self.printChan <- err.Error()
					continue
				}
				self.count[1].rate.Incr(1)
				self.messageChan <- &message{
					buf:  buf[:n],
					addr: addr,
				}
			}
		}()
	}

}

func decodeNodes(s string) (nodes []*node) {
	length := len(s)
	if length%26 != 0 {
		return
	}
	for i := 0; i < length; i += 26 {
		id := s[i : i+20]
		ip := net.IP([]byte(s[i+20 : i+24])).String()
		port := binary.BigEndian.Uint16([]byte(s[i+24 : i+26]))
		nodes = append(nodes, &node{id: id, addr: ip + ":" + strconv.Itoa(int(port))})
	}
	return
}

func (self *Worker) AutoSendFindNode() {
	var one *node
	for {
		one = <-self.nodeChan
		self.sendFindNode(one)
		if len(self.kbucket) < 8 {
			self.kbucket = append(self.kbucket, one)
		}
	}
}

func (self *Worker) FindNode() {
	for {
		if self.count[4].rate.Rate() == 0 {
			for _, address := range bootstapNodes {
				self.printChan <- "send to: " + address
				self.sendFindNode(&node{
					addr: address,
					id:   self.localID,
				})
			}
		} else {
			time.Sleep(15 * time.Second)
			for _, address := range bootstapNodes {
				self.sendFindNode(&node{
					addr: address,
					id:   self.localID,
				})
			}
		}
		time.Sleep(5 * time.Second)
	}
}
func (self *Worker) PrintLog() {
	go self.timer()
	for {
		fmt.Printf("\r")
		fmt.Printf("%s", <-self.printChan)
	}
}

func (self *Worker) Server() {
	self.Tool.ToolServer(&self.Tool)

}
func (self *Worker) timer() {
	for {
		self.printChan <- "Rev: " + strconv.FormatInt(self.count[1].rate.Rate(), 10) + "r/sec" +
			" Decode: " + strconv.FormatInt(self.count[3].rate.Rate(), 10) + "r/sec" +
			" Suss: " + strconv.FormatInt(self.count[0].rate.Rate(), 10) + "p/sec" + " FindNode: " +
			strconv.FormatInt(self.count[4].rate.Rate(), 10) + "p/sec" + " Drop: " +
			strconv.FormatInt(self.count[2].rate.Rate(), 10) + "r/sec"
		time.Sleep(time.Second * 1)
	}

}

func (self *Worker) onReply(dict *map[string]interface{}, from net.Addr) {
	// tid, ok := (*dict)["t"].(string)
	// if !ok {
	// 	return
	// }
	r, ok := (*dict)["r"].(map[string]interface{})
	if !ok {
		return
	}
	nodes, ok := r["nodes"].(string)
	if !ok {
		return
	}
	if len(self.nodeChan) < nodeChanSize {
		for _, node := range decodeNodes(nodes) {
			if findNodeSpeedLimiter.Allow() {
				self.nodeChan <- node
			}
		}

	} else {
		self.count[2].rate.Incr(1)
	}

}

func (self *Worker) onQuery(dict *map[string]interface{}, from net.Addr) {
	q, ok := (*dict)["q"]
	if !ok {
		self.printChan <- "dict q err,788990"
		return
	}
	switch q {
	case pingType:
		self.onPing(dict, from)
	case findNodeType:
		self.onFindNode(dict, from)
	case getPeersType:
		self.onGetPeers(*dict, from)
	case announcePeerType:
		self.onAnnouncePeer(dict, from)
		// default:
		// 	self.playDead(dict, from)
	}
}

func (self *Worker) onFindNode(dict *map[string]interface{}, from net.Addr) {
	// c := 1
	tid, ok := (*dict)["t"].(string)
	if !ok {
		return
	}
	d := makeReply(tid, map[string]interface{}{
		"id":    string(randBytes(2)), //self.localID,
		"nodes": self.nodes,
	})
	self.udpListener.WriteTo(bencode.Encode(d), from)

}
func (self *Worker) onMessage() {
	var data *message
	for {
		data = <-self.messageChan
		dict := map[string]interface{}{}
		dict, err := bencode.Decode(bytes.NewBuffer(data.buf))
		if err != nil {
			// self.printChan <- ("ERR 121213:" + err.Error())
			continue
		}
		self.count[3].rate.Incr(1)
		y, ok := dict["y"].(string)
		if !ok {
			continue
		}
		switch y {
		case "q":
			self.onQuery(&dict, data.addr)
		case "r": //,
			self.onReply(&dict, data.addr)
		//case "e": //处理错误不写 爬虫没必要浪费资源
		default:
			continue
		}
	}
}
func (self *Worker) onPing(dict *map[string]interface{}, from net.Addr) {
	tid, ok := (*dict)["t"].(string)
	if !ok {
		return
	}
	d := makeReply(tid, map[string]interface{}{
		"id": self.localID,
	})

	self.udpListener.WriteTo(bencode.Encode(d), from)
}
func makeReply(tid string, r map[string]interface{}) map[string]interface{} {
	dict := map[string]interface{}{
		"t": tid,
		"y": "r",
		"r": r,
	}
	return dict
}
func genToken(from net.Addr) string {
	return secret + from.String()[:8]
	// sha1 := sha1.New()
	// sha1.Write(from.IP)
	// sha1.Write([]byte(secret))
	// return string(sha1.Sum(nil))
}

func (self *Worker) onGetPeers(dict map[string]interface{}, from net.Addr) {

	t, ok := dict["t"].(string)
	if !ok {
		return
	}
	a, ok := dict["a"].(map[string]interface{})
	if !ok {
		return
	}
	id, ok := a["id"].(string)
	if !ok {
		return
	}
	d := makeReply(t, map[string]interface{}{
		"id":    string(neighborID(id, self.localID)),
		"nodes": self.nodes,
		"token": genToken(from),
	})

	self.udpListener.WriteTo(bencode.Encode(d), from)

}

func (self *Worker) onAnnouncePeer(dict *map[string]interface{}, from net.Addr) {
	tid, ok := (*dict)["t"].(string)
	if !ok {
		return
	}
	a, ok := (*dict)["a"].(map[string]interface{})
	if !ok {
		return
	}
	token, ok := a["token"].(string)
	if !ok || token != genToken(from) {
		return
	}

	infohash, ok := a["info_hash"].(string)
	if !ok {
		return
	}
	impliedPort, ok := a["implied_port"].(int64)
	if !ok {
		return
	}
	if impliedPort == 1 {
		fromPort, ok := a["port"].(int64)
		if !ok {
			return
		}
		from1, err := net.ResolveIPAddr("ip", from.String()[:strings.LastIndex(from.String(), ":")]+strconv.FormatInt(fromPort, 10))
		if err == nil {
			from = from1
		}
	}
	d := makeReply(tid, map[string]interface{}{
		"id": self.localID,
	})

	self.udpListener.WriteTo(bencode.Encode(d), from)
	if len(self.Tool.ToolPostChan) == cap(self.Tool.ToolPostChan) {
		<-self.Tool.ToolPostChan
		self.count[2].rate.Incr(1)
	}
	self.Tool.ToolPostChan <- header.Tdata{
		Hash: hex.EncodeToString([]byte(infohash)),
		Addr: from.String(),
	}
	self.count[0].rate.Incr(1)

}
func generEmptyString(length int) (result string) {

	for i := 0; i < length; i++ {
		result = result + " "
	}

	return
}
func checkError(err error) {
	if err != nil {
		panic(err.Error())
	}

}
func randBytes(length int) []byte {
	b := make([]byte, length)
	rand.Read(b)
	return b
}

func neighborID(target string, local string) string {
	return target[:10] + local[10:]
}

func makeQuery(tid string, q string, a map[string]interface{}) map[string]interface{} {
	dict := map[string]interface{}{
		"t": tid,
		"y": "q",
		"q": q,
		"a": a,
	}
	return dict
}

func (self *Worker) sendFindNode(one *node) {
	self.count[4].rate.Incr(1)
	msg := makeQuery(secret+one.addr[:4], "find_node", map[string]interface{}{
		"id":     neighborID(one.id, self.localID),
		"target": string(randBytes(20)),
	})
	addr, err := net.ResolveUDPAddr("udp", one.addr)
	if err != nil {
		return
	}
	randx.Seed(time.Now().Unix())
	self.udpListener.WriteTo(bencode.Encode(msg), addr)
}

func nodeToString(nodes []*node) (result string) {
	for _, j := range nodes {
		addr, err := net.ResolveUDPAddr("udp", j.addr)
		if err != nil {
			continue
		}
		port := uint16(addr.Port)
		result = result + j.id + string(addr.IP.To4()) + string([]byte{byte(port >> 8), byte(port)})
	}
	return result
}

func (self *Worker) GenerNodes() {
	for {
		if len(self.kbucket) >= 8 {
			self.nodes = nodeToString(self.kbucket)
			self.kbucket = []*node{}
			time.Sleep(time.Minute)
		}
		time.Sleep(5)
	}

}
