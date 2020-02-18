package common

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Bmixo/btSearch/header"
	reuse "github.com/libp2p/go-reuseport"

	"github.com/Bmixo/btSearch/package/bencode"
	"github.com/Bmixo/btSearch/package/metawire"
	"github.com/go-ego/gse"
	"github.com/go-redis/redis"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func (self *Server) handleData() {
	tdataChanCap := cap(self.tdataChan)
	for {
		s := <-self.Tool.ToolPostChan

		self.revNum++
		if self.blackAddrList.Contains(s.Addr) {
			continue
		}

		InfoHash := s.Hash

		if self.hashList.Contains(InfoHash) {
			continue
		}
		select {
		case self.mongoLimit <- true:
		default:
			self.dropSpeed++
			continue
		}
		m, err := self.findHash(InfoHash)
		if err != nil && err != mgo.ErrNotFound {
			self.printChan <- ("\n" + "ERR:4511" + err.Error() + "\n")
			continue
		}

		if m != nil {
			select {
			case self.mongoLimit <- true:
			default:
				self.dropSpeed++
				continue
			}
			self.foundNum++
			err = self.updateTimeHot(m["_id"].(bson.ObjectId))
			if err != nil {
				self.printChan <- ("\n" + "update time hot ERR:0025" + err.Error() + "\n")
				continue
			}
			continue
		}
		if len(self.tdataChan) < tdataChanCap {
			self.tdataChan <- s
			self.hashList.Add(InfoHash)
		} else {
			self.dropSpeed++
		}

	}
}
func (self *Server) NewServerConn() {
	for _, node := range wkNodes {
		self.Tool.Links = append(self.Tool.Links, Link{Conn: nil, Addr: node, LinkPostChan: make(chan header.Tdata, 1000)})
	}
	for i := 0; i < 10; i++ {
		go self.handleData()
	}
	self.Tool.LinksServe()

}
func (self *Server) Reboot() {

	for {
		time.Sleep(time.Second * 240)
		self.blackAddrList.Clear()
		self.hashList.Clear()
	}

}
func (self *Server) PrintLog() {

	for {
		fmt.Printf("\r")
		fmt.Printf("%s", <-self.printChan)
	}

}

func (self *Server) CheckSpeed() {
	sussNum := 0
	dropSpeed := 0
	foundNum := 0
	revNum := 0
	for {
		self.sussNum -= sussNum
		self.dropSpeed -= dropSpeed
		self.foundNum -= foundNum
		self.revNum -= revNum
		self.printChan <- ("RevSpeed: " + strconv.Itoa(self.revNum) + "/sec" +
			" DropSpeed: " + strconv.Itoa(self.dropSpeed) + "/sec" +
			" FoundSpeed: " + strconv.Itoa(self.foundNum) + "/sec" +
			" SussSpeed: " + strconv.Itoa(self.sussNum) + "/sec" +
			" HashList:" + strconv.Itoa(self.hashList.Cardinality()) +
			" blackAddrList:" + strconv.Itoa(self.blackAddrList.Cardinality()))
		sussNum = self.sussNum
		dropSpeed = self.dropSpeed
		foundNum = self.foundNum
		revNum = self.revNum
		time.Sleep(time.Second)
	}

}

func (self *Server) Metadata() {
	if metadataNum < 1 {
		self.printChan <- ("metadataNum error set defalut 10")
	}
	nla, err := net.ResolveTCPAddr("tcp4", ":9797")
	if err != nil {
		panic("resolving local addr")
	}
	dialer := net.Dialer{Control: reuse.Control, Timeout: time.Second * 1, LocalAddr: nla}
	for i := 0; i < metadataNum; i++ {
		go func() {
			for {
				tdata := <-self.tdataChan
				infoHash, err := hex.DecodeString(tdata.Hash)
				if err != nil {
					continue
				}

				peer := metawire.New(
					string(infoHash),
					tdata.Addr,
					metawire.Dialer(dialer),
					metawire.Timeout(time.Second*1),
					metawire.Timeout(time.Second*3),
				)
				data, err := peer.Fetch()
				if err != nil {
					self.blackAddrList.Add(tdata.Addr)
					continue
				}

				torrent, err := self.newTorrent(data, tdata.Hash)
				if err != nil {
					continue
				}

				segments := self.segmenter.Segment([]byte(torrent.Name))
				for _, j := range gse.ToSlice(segments, false) {
					if utf8.RuneCountInString(j) < 2 || utf8.RuneCountInString(j) > 15 {
						continue
					} else if len(torrent.KeyWord) > 10 {
						break
					} else {
						if _, error := strconv.Atoi(j); error != nil {
							torrent.KeyWord = append(torrent.KeyWord, j)
						}
					}
				}
				select {
				case self.mongoLimit <- true:
				default:
					self.dropSpeed++
					continue
				}
				err = self.syncmongodb(torrent)

				if err != nil {
					continue
				}

				self.sussNum++
				self.hashList.Remove(torrent.InfoHash)
				//self.printChan <- ("------" + torrent.Name + "------" + torrent.InfoHash)
				continue
			}
		}()
	}

}

func (self *Server) newTorrent(metadata []byte, InfoHash string) (torrent bitTorrent, err error) {
	info, err := bencode.Decode(bytes.NewBuffer(metadata))
	if err != nil {
		return bitTorrent{}, err
	}
	timestamp := time.Now().Unix()
	if _, ok := info["name"]; !ok {
		return bitTorrent{}, errors.New("Metadata Name is Empty")
	}
	if t, ok := info["name"].(string); ok {
		if !utf8.Valid([]byte(t)) {
			return bitTorrent{}, errors.New("Metadata Name is not Encode by utf-8")
		}
	} else {
		return bitTorrent{}, errors.New("interface conversion: interface {} is int64, not string,90099")
	}

	for _, black := range self.blackList {
		if strings.Contains(info["name"].(string), black) {

			return bitTorrent{}, errors.New("Metadata Name is in Blacklist")
		}
	}

	bt := bitTorrent{
		ID:         bson.NewObjectId(),
		InfoHash:   InfoHash,
		Name:       info["name"].(string),
		CreateTime: timestamp,
		LastTime:   timestamp,
	}

	var sourceName string
	if v, ok := info["files"]; ok {
		var biggestfile file
		files := v.([]interface{})
		bt.Files = make([]file, len(files))
		var TotalLength int64

		bt.FileType = "Unknow"
		for i, item := range files {
			f := item.(map[string]interface{})

			if _, ok := f["length"].(int64); !ok {
				return bitTorrent{}, errors.New("length, not int64")
			}
			TotalLength = TotalLength + f["length"].(int64)
			if f["length"].(int64) > biggestfile.Length {
				biggestfile.Length = f["length"].(int64)
				biggestfile.Path = f["path"].([]interface{})
			}
			bt.Files[i] = file{
				Path:   f["path"].([]interface{}),
				Length: f["length"].(int64),
			}
		}
		bt.Length = TotalLength
		sourceName = biggestfile.Path[len(biggestfile.Path)-1].(string)

	} else if _, ok := info["length"]; ok {
		bt.Length = info["length"].(int64)
		sourceName = bt.Name
	}
	bt.Extension = path.Ext(sourceName)

findName:
	for i, one := range cats {
		tmpLegth := len(one)
		for j := 0; j < tmpLegth; j++ {

			if strings.HasSuffix(sourceName, one[j]) {

				switch i {
				case 0:
					bt.FileType = "Video"
				case 1:
					bt.FileType = "Image"
				case 2:
					bt.FileType = "Document"
				case 3:
					bt.FileType = "Music"
				case 4:
					bt.FileType = "Package"
				case 5:
					bt.FileType = "Software"
				default:
					bt.FileType = "Unknow"
				}
				break findName
			}

		}
	}

	return bt, nil

}

func (self *Server) findHash(infohash string) (m map[string]interface{}, err error) {
	if redisEnable {
		val, redisErr := self.RedisClient.Get(infohash).Result()
		if redisErr == redis.Nil {
			c := self.Mon.DB(dataBase).C(collection)
			selector := bson.M{"infohash": infohash}
			err = c.Find(selector).One(&m)
			if m != nil {
				self.RedisClient.Set(infohash, m["_id"].(bson.ObjectId), 0)
			}
			return
		} else if redisErr != nil {
			c := self.Mon.DB(dataBase).C(collection)
			selector := bson.M{"infohash": infohash}
			err = c.Find(selector).One(&m)
		} else {
			m["_id"] = bson.ObjectId(val)
		}
	} else {
		c := self.Mon.DB(dataBase).C(collection)
		selector := bson.M{"infohash": infohash}
		err = c.Find(selector).One(&m)
	}
	<-self.mongoLimit
	return
}

func (self *Server) syncmongodb(data bitTorrent) (err error) {

	c := self.Mon.DB(dataBase).C(collection)
	err = c.Insert(data)
	<-self.mongoLimit
	return
}

func (self *Server) updateTimeHot(objectID bson.ObjectId) (err error) {

	c := self.Mon.DB(dataBase).C(collection)

	m := make(map[string]interface{})
	m["$inc"] = map[string]int{"hot": 1}
	m["$set"] = map[string]int64{"last_time": time.Now().Unix()}

	selector := bson.M{"_id": objectID}
	err = c.Update(selector, m)
	<-self.mongoLimit
	return
}

func loadBlackList() (blackList []string) {
	fi, err := os.Open(banList)

	if err != nil {
		fi.Close()
		log.Panicln("\nError: %s\n", err)
		return []string{}
	}
	defer fi.Close()
	br := bufio.NewReader(fi)
	for {
		a, _, c := br.ReadLine()
		if c == io.EOF {
			break
		}
		blackList = append(blackList, string(a))

	}
	fi.Close()
	return []string{}
}

func exists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func writeFile(filename string, data []byte) error {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	n, err := f.Write(data)
	if err == nil && n < len(data) {
		err = io.ErrShortWrite
	}
	if err1 := f.Close(); err == nil {
		err = err1
	}
	return err
}

func intToBytes(i int) []byte {
	var buf = make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(i))
	return buf
}

func bytesToInt(buf []byte) int {
	return int(binary.BigEndian.Uint32(buf))
}
