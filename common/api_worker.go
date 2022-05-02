package common

import (
	"bytes"
	"crypto/rand"
	"github.com/Bmixo/btSearch/api/api_server_1/torrent"
	"github.com/Bmixo/btSearch/pkg/metawire"
	reuse "github.com/libp2p/go-reuseport"
	"log"
	randx "math/rand"
	"strings"

	"encoding/binary"
	"encoding/hex"
	"net"
	"strconv"
	"time"

	"github.com/Bmixo/btSearch/pkg/bencode"
)

func (m *Worker) Refresh() {
	for {
		time.Sleep(time.Second * 60)
		m.blackAddrList.Clear()
		m.hashList.Clear()
	}
}

func (m *Worker) HandleMsg() {
	for i := 0; i < 20; i++ {
		go m.onMessage()
	}
	for i := 0; i < 10; i++ {

		go func() {
			for {
				buf := make([]byte, 512)
				n, addr, err := m.udpListener.ReadFrom(buf)
				if err != nil {
					m.printChan <- err.Error()
					continue
				}
				m.count[1].rate.Incr(1)
				m.messageChan <- &message{
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

func (m *Worker) AutoSendFindNode() {
	var one *node
	for {
		one = <-m.nodeChan
		m.sendFindNode(one)
		if len(m.kbucket) < 8 {
			m.kbucket = append(m.kbucket, one)
		}
	}
}

func (m *Worker) FindNode() {
	for {
		if m.count[4].rate.Rate() == 0 {
			for _, address := range bootstapNodes {
				m.printChan <- "send to: " + address
				m.sendFindNode(&node{
					addr: address,
					id:   m.localID,
				})
			}
		} else {
			time.Sleep(15 * time.Second)
			for _, address := range bootstapNodes {
				m.sendFindNode(&node{
					addr: address,
					id:   m.localID,
				})
			}
		}
		time.Sleep(5 * time.Second)
	}
}

func (m *Worker) PrintLog() {
	go m.timer()
	for {
		//log.Printf("\r")
		log.Printf("%s", <-m.printChan)
	}
}

func (m *Worker) Server() {
	m.Tool.ToolServer(&m.Tool)

}
func (m *Worker) timer() {
	for {
		m.printChan <- "Rev: " + strconv.FormatInt(m.count[1].rate.Rate(), 10) + "/sec" +
			" Decode: " + strconv.FormatInt(m.count[3].rate.Rate(), 10) + "/sec" +
			" Suss: " + strconv.FormatInt(m.count[0].rate.Rate(), 10) + "/sec" +
			" FindNode: " + strconv.FormatInt(m.count[4].rate.Rate(), 10) + "/sec" +
			" Drop: " + strconv.FormatInt(m.count[2].rate.Rate(), 10) + "/sec" +
			" HashList:" + strconv.Itoa(m.hashList.Cardinality()) + "/sec" +
			" blackAddrList:" + strconv.Itoa(m.blackAddrList.Cardinality())
		time.Sleep(time.Second * 1)
	}

}

func (m *Worker) onReply(dict *map[string]interface{}, from net.Addr) {
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
	if len(m.nodeChan) < nodeChanSize {
		for _, node := range decodeNodes(nodes) {
			if findNodeSpeedLimiter.Allow() {
				m.nodeChan <- node
			}
		}

	} else {
		m.count[2].rate.Incr(1)
	}

}

func (m *Worker) onQuery(dict *map[string]interface{}, from net.Addr) {
	q, ok := (*dict)["q"]
	if !ok {
		m.printChan <- "dict q err,788990"
		return
	}
	switch q {
	case pingType:
		m.onPing(dict, from)
	case findNodeType:
		m.onFindNode(dict, from)
	case getPeersType:
		m.onGetPeers(*dict, from)
	case announcePeerType:
		m.onAnnouncePeer(dict, from)
		// default:
		// 	m.playDead(dict, from)
	}
}

func (m *Worker) onFindNode(dict *map[string]interface{}, from net.Addr) {
	// c := 1
	tid, ok := (*dict)["t"].(string)
	if !ok {
		return
	}
	d := makeReply(tid, map[string]interface{}{
		"id":    string(randBytes(2)), //m.localID,
		"nodes": m.nodes,
	})
	m.udpListener.WriteTo(bencode.Encode(d), from)

}
func (m *Worker) onMessage() {
	var data *message
	for {
		data = <-m.messageChan
		dict := map[string]interface{}{}
		dict, err := bencode.Decode(bytes.NewBuffer(data.buf))
		if err != nil {
			// m.printChan <- ("ERR 121213:" + err.Error())
			continue
		}
		m.count[3].rate.Incr(1)
		y, ok := dict["y"].(string)
		if !ok {
			continue
		}
		switch y {
		case "q":
			m.onQuery(&dict, data.addr)
		case "r": //,
			m.onReply(&dict, data.addr)
		//case "e": //处理错误不写 爬虫没必要浪费资源
		default:
			continue
		}
	}
}
func (m *Worker) onPing(dict *map[string]interface{}, from net.Addr) {
	tid, ok := (*dict)["t"].(string)
	if !ok {
		return
	}
	d := makeReply(tid, map[string]interface{}{
		"id": m.localID,
	})

	m.udpListener.WriteTo(bencode.Encode(d), from)
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

func (m *Worker) onGetPeers(dict map[string]interface{}, from net.Addr) {

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
		"id":    string(neighborID(id, m.localID)),
		"nodes": m.nodes,
		"token": genToken(from),
	})

	m.udpListener.WriteTo(bencode.Encode(d), from)

}

func (m *Worker) onAnnouncePeer(dict *map[string]interface{}, from net.Addr) {
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
		"id": m.localID,
	})

	m.udpListener.WriteTo(bencode.Encode(d), from)
	if len(m.Tool.ToolSendChan) == cap(m.Tool.ToolSendChan) {
		<-m.Tool.ToolSendChan
		m.count[2].rate.Incr(1)
	}
	infoHash := hex.EncodeToString([]byte(infohash))
	if m.hashList.Contains(infoHash) {
		return
	}
	m.hashList.Add(infoHash)
	m.Tool.ToolSendChan <- torrent.TData{
		TDataCode: torrent.TDataCode_TDataCodeVerifyHashExist,
		Hash:      infoHash,
		Addr:      from.String(),
	}
	m.count[0].rate.Incr(1)

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

func (m *Worker) sendFindNode(one *node) {
	m.count[4].rate.Incr(1)
	msg := makeQuery(secret+one.addr[:4], "find_node", map[string]interface{}{
		"id":     neighborID(one.id, m.localID),
		"target": string(randBytes(20)),
	})
	addr, err := net.ResolveUDPAddr("udp", one.addr)
	if err != nil {
		return
	}
	randx.Seed(time.Now().Unix())
	m.udpListener.WriteTo(bencode.Encode(msg), addr)
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

func (m *Worker) GenerNodes() {
	for {
		if len(m.kbucket) >= 8 {
			m.nodes = nodeToString(m.kbucket)
			m.kbucket = []*node{}
			time.Sleep(time.Minute)
		}
		time.Sleep(5)
	}
}

func (m *Worker) Metadata() {
	if metadataNum < 1 {
		m.printChan <- "metadataNum error set defalut 10"
	}
	nla, err := net.ResolveTCPAddr("tcp4", ":9797")
	if err != nil {
		panic("resolving local addr")
	}
	go func() {
		for {
			tDataTmp := <-m.Tool.ToolRevChan
			if tDataTmp.TDataCode == torrent.TDataCode_TDataCodeNeedDownload {
				m.tdataChan <- tDataTmp
			}
		}
	}()
	dialer := net.Dialer{Control: reuse.Control, Timeout: time.Second * 1, LocalAddr: nla}
	for i := 0; i < metadataNum; i++ {
		go func() {
			for {
				tdatax := <-m.tdataChan
				infoHash, err := hex.DecodeString(tdatax.Hash)
				if err != nil {
					continue
				}
				peer := metawire.New(
					string(infoHash),
					tdatax.Addr,
					metawire.Dialer(dialer),
					metawire.Timeout(time.Second*1),
					metawire.Timeout(time.Second*3),
				)
				data, err := peer.Fetch()
				if err != nil {
					m.blackAddrList.Add(tdatax.Addr)
					continue
				}
				m.Tool.ToolSendChan <- torrent.TData{
					Hash:        tdatax.Hash,
					TDataCode:   torrent.TDataCode_TDataCodeNeedHandleTorrentData,
					TorrentData: data,
				}
				m.hashList.Remove(tdatax.Hash)
				//m.printChan <- ("------" + torrent.Name + "------" + torrent.InfoHash)
				continue
			}
		}()
	}

}
