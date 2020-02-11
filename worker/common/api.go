package common

import (
	"bytes"
	"crypto/rand"

	randx "math/rand"

	"github.com/Bmixo/btSearch/header"

	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/Bmixo/btSearch/package/bencode"
)

func (wk *wkServer) HandleMsg() {
	for i := 0; i < 20; i++ {
		go wk.onMessage()
	}
	for i := 0; i < 10; i++ {

		go func() {
			for {
				buf := make([]byte, 512)
				n, addr, err := wk.udpListener.ReadFromUDP(buf)
				if err != nil {
					wk.printChan <- (err.Error())
					continue
				}
				wk.revNum = wk.revNum + 1
				wk.messageChan <- &message{
					buf:  buf[:n],
					addr: *addr,
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

func (wk *wkServer) AutoSendFindNode() {
	var one *node
	for {
		one = <-wk.nodeChan
		wk.sendFindNode(one)
		if len(wk.kbucket) < 8 {
			wk.kbucket = append(wk.kbucket, one)
		}
	}
}

func (wk *wkServer) FindNode() {
	for {
		if wk.findNodeNum == 0 {
			for _, address := range bootstapNodes {
				wk.printChan <- ("send to: " + address)
				wk.sendFindNode(&node{
					addr: address,
					id:   wk.localID,
				})
			}
		}
		time.Sleep(5 * time.Second)
	}
}
func (wk *wkServer) PrintLog() {
	go wk.timer()
	for {
		fmt.Printf("\r")
		fmt.Printf("%s", <-wk.printChan)
	}
}

func (wk *wkServer) Server() {
	wk.Tool.ToolServer(&wk.Tool)

}
func (wk *wkServer) timer() {
	findNodeNumOld := 0
	sussNumOld := 0
	dropNumOld := 0
	revNumOld := 0
	decodeNumOld := 0
	for {
		wk.findNodeNum -= findNodeNumOld
		wk.DecodeNum -= decodeNumOld
		wk.sussNum -= sussNumOld
		wk.dropNum -= dropNumOld
		wk.revNum -= revNumOld
		wk.printChan <- ("Rev: " + strconv.Itoa(wk.revNum) + "r/sec" +
			" Decode: " + strconv.Itoa(wk.DecodeNum) + "r/sec" +
			" Suss: " + strconv.Itoa(wk.sussNum) + "p/sec" + " FindNode: " +
			strconv.Itoa(wk.findNodeNum) + "p/sec" + " Drop: " +
			strconv.Itoa(wk.dropNum) + "r/sec")
		findNodeNumOld = wk.findNodeNum
		sussNumOld = wk.sussNum
		dropNumOld = wk.dropNum
		revNumOld = wk.revNum
		decodeNumOld = wk.DecodeNum

		time.Sleep(time.Second * 1)
	}

}

func (wk *wkServer) onReply(dict *map[string]interface{}, from *net.UDPAddr) {
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
	if len(wk.nodeChan) < nodeChanSize && wk.findNodeNum < findNodeSpeed {
		for _, node := range decodeNodes(nodes) {
			wk.nodeChan <- node
		}

	} else {
		wk.dropNum = wk.dropNum + 1
	}

}

func (wk *wkServer) onQuery(dict *map[string]interface{}, from *net.UDPAddr) {
	q, ok := (*dict)["q"]
	if !ok {
		wk.printChan <- ("dict q err,788990")
		return
	}
	switch q {
	case pingType:
		wk.onPing(dict, from)
	case findNodeType:
		wk.onFindNode(dict, from)
	case getPeersType:
		wk.onGetPeers(*dict, from)
	case announcePeerType:
		wk.onAnnouncePeer(dict, from)
		// default:
		// 	wk.playDead(dict, from)
	}
}

func (wk *wkServer) onFindNode(dict *map[string]interface{}, from *net.UDPAddr) {
	// c := 1
	tid, ok := (*dict)["t"].(string)
	if !ok {
		return
	}
	d := makeReply(tid, map[string]interface{}{
		"id":    string(randBytes(2)), //wk.localID,
		"nodes": wk.nodes,
	})
	wk.udpListener.WriteTo(bencode.Encode(d), from)

}
func (wk *wkServer) onMessage() {
	var data *message
	for {
		data = <-wk.messageChan
		dict := map[string]interface{}{}
		dict, err := bencode.Decode(bytes.NewBuffer(data.buf))
		if err != nil {
			// wk.printChan <- ("ERR 121213:" + err.Error())
			continue
		}
		wk.DecodeNum++
		y, ok := dict["y"].(string)
		if !ok {
			continue
		}
		switch y {
		case "q":
			wk.onQuery(&dict, &data.addr)
		case "r": //,
			wk.onReply(&dict, &data.addr)
		//case "e": //处理错误不写 爬虫没必要浪费资源
		default:
			continue
		}
	}
}
func (wk *wkServer) onPing(dict *map[string]interface{}, from *net.UDPAddr) {
	tid, ok := (*dict)["t"].(string)
	if !ok {
		return
	}
	d := makeReply(tid, map[string]interface{}{
		"id": wk.localID,
	})

	wk.udpListener.WriteTo(bencode.Encode(d), from)
}
func makeReply(tid string, r map[string]interface{}) map[string]interface{} {
	dict := map[string]interface{}{
		"t": tid,
		"y": "r",
		"r": r,
	}
	return dict
}
func genToken(from *net.UDPAddr) string {
	return secret + from.String()[:8]
	// sha1 := sha1.New()
	// sha1.Write(from.IP)
	// sha1.Write([]byte(secret))
	// return string(sha1.Sum(nil))
}

func (wk *wkServer) onGetPeers(dict map[string]interface{}, from *net.UDPAddr) {

	t := dict["t"].(string)
	a, ok := dict["a"].(map[string]interface{})
	if !ok {
		return
	}
	id, ok := a["id"].(string)
	if !ok {
		return
	}
	d := makeReply(t, map[string]interface{}{
		"id":    string(neighborID(id, wk.localID)),
		"nodes": wk.nodes,
		"token": genToken(from),
	})

	wk.udpListener.WriteTo(bencode.Encode(d), from)

}

func (wk *wkServer) onAnnouncePeer(dict *map[string]interface{}, from *net.UDPAddr) {
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
		from.Port = int(fromPort)
	}
	d := makeReply(tid, map[string]interface{}{
		"id": wk.localID,
	})

	wk.udpListener.WriteTo(bencode.Encode(d), from)
	wk.sussNum = wk.sussNum + 1
	if len(wk.Tool.ToolPostChan) == cap(wk.Tool.ToolPostChan) {
		<-wk.Tool.ToolPostChan
	}
	wk.Tool.ToolPostChan <- header.Tdata{
		Hash: hex.EncodeToString([]byte(infohash)),
		Addr: from.String(),
	}

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

func (wk *wkServer) sendFindNode(one *node) {
	wk.findNodeNum = wk.findNodeNum + 1
	msg := makeQuery(secret+one.addr[:4], "find_node", map[string]interface{}{
		"id":     neighborID(one.id, wk.localID),
		"target": string(randBytes(20)),
	})
	addr, err := net.ResolveUDPAddr("udp4", one.addr)
	if err != nil {
		return
	}
	randx.Seed(time.Now().Unix())
	wk.udpListener.WriteTo(bencode.Encode(msg), addr)
}

func nodeToString(nodes []*node) (result string) {
	for _, j := range nodes {
		addr, err := net.ResolveUDPAddr("udp4", j.addr)
		if err != nil {
			continue
		}
		port := uint16(addr.Port)
		result = result + j.id + string(addr.IP.To4()) + string([]byte{byte(port >> 8), byte(port)})
	}
	return result
}

func (wk *wkServer) GenerNodes() {
	for {
		if len(wk.kbucket) >= 8 {
			wk.nodes = nodeToString(wk.kbucket)
			wk.kbucket = []*node{}
			time.Sleep(time.Minute)
		}
		time.Sleep(5)
	}

}
