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
	"os"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Bmixo/btSearch/header"

	"github.com/Bmixo/btSearch/package/bencode"
	"github.com/Bmixo/btSearch/package/metawire"
	"github.com/go-ego/gse"
	"github.com/go-redis/redis"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func (sniffer *sn) handleData() {
	for {
		s := <-sniffer.Tool.ToolPostChan

		sniffer.revNum = sniffer.revNum + 1
		if sniffer.blackAddrList.Contains(s.Addr) {
			return
		}

		//InfoHash := hex.EncodeToString(s.Hash)
		InfoHash := s.Hash

		if !sniffer.hashList.Contains(InfoHash) {
			select {
			case sniffer.mongoLimit <- true:
			default:
				sniffer.dropSpeed = sniffer.dropSpeed + 1
				return
			}
			m, err := sniffer.findHash(InfoHash)
			if err != nil && err != mgo.ErrNotFound {
				sniffer.printChan <- ("ERR:4511" + err.Error())
				return
			}

			if m != nil {
				select {
				case sniffer.mongoLimit <- true:
				default:
					sniffer.dropSpeed = sniffer.dropSpeed + 1
					return
				}
				sniffer.foundNum = sniffer.foundNum + 1
				err = sniffer.updateTimeHot(m["_id"].(bson.ObjectId))
				if err != nil {
					sniffer.printChan <- ("ERR:0025" + err.Error())
					return
				}
			} else {
				if len(sniffer.tdataChan) < 100 {
					sniffer.tdataChan <- s
					sniffer.hashList.Add(InfoHash)
				} else {
					sniffer.dropSpeed = sniffer.dropSpeed + 1
				}

			}

		}
	}
}
func (sniffer *sn) NewServerConn() {
	for _, node := range wkNodes {
		sniffer.Tool.Links = append(sniffer.Tool.Links, Link{Conn: nil, Addr: node, LinkPostChan: make(chan header.Tdata, 1000)})
	}
	go sniffer.handleData()
	sniffer.Tool.LinksServe()

}
func (sniffer *sn) Reboot() {

	for {
		time.Sleep(time.Second * 240)
		sniffer.blackAddrList.Clear()
		sniffer.hashList.Clear()
	}

}
func (sniffer *sn) PrintLog() {

	for {
		fmt.Printf("\r")
		fmt.Printf("%s", <-sniffer.printChan)
	}

}

func (sniffer *sn) CheckSpeed() {
	for {
		sniffer.printChan <- ("RevSpeed: " + strconv.FormatFloat(sniffer.revNum, 'f', 3, 64) + "/sec" + "\tDropSpeed: " + strconv.FormatFloat(sniffer.dropSpeed, 'f', 3, 64) + "\tFoundSpeed: " + strconv.FormatFloat(sniffer.foundNum, 'f', 3, 64) + "/sec" + "\tSussSpeed: " + strconv.FormatFloat(sniffer.sussNum, 'f', 3, 64) + "/sec")
		sniffer.printChan <- ("HashList:" + strconv.FormatInt(int64(sniffer.hashList.Cardinality()), 10) + "  blackAddrList:" + strconv.FormatInt(int64(sniffer.blackAddrList.Cardinality()), 10) + " \n")
		sniffer.sussNum = 0.0
		sniffer.dropSpeed = 0.0
		sniffer.foundNum = 0.0
		sniffer.revNum = 0.0
		time.Sleep(time.Second)
	}

}

func (sniffer *sn) Metadata() {
	for one := range sniffer.tdataChan {

		go func(i header.Tdata) {
			infoHash, err := hex.DecodeString(i.Hash)
			if err != nil {
				return
			}

			peer := metawire.New(
				string(infoHash),
				i.Addr,
				metawire.Timeout(15*time.Second),
			)
			data, err := peer.Fetch()
			if err != nil {
				sniffer.blackAddrList.Add(i.Addr)
				return
			}

			torrent, err := sniffer.newTorrent(data, i.Hash)
			if err != nil {
				return
			}

			segments := sniffer.segmenter.Segment([]byte(torrent.Name))
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
			// objectid, err := torrent.ID.MarshalText()
			select {
			case sniffer.mongoLimit <- true:
			default:
				sniffer.dropSpeed = sniffer.dropSpeed + 1
				return
			}
			err = sniffer.syncmongodb(torrent)

			if err != nil {
				return
			}

			sniffer.sussNum = sniffer.sussNum + 1
			sniffer.hashList.Remove(torrent.InfoHash)
			//sniffer.printChan <- ("------" + torrent.Name + "------" + torrent.InfoHash)
			return

		}(one)
	}

}

func (sniffer *sn) newTorrent(metadata []byte, InfoHash string) (torrent bitTorrent, err error) {
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

	for _, black := range sniffer.blackList {
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

func (sniffer *sn) findHash(infohash string) (m map[string]interface{}, err error) {
	if redisEnable {
		val, redisErr := sniffer.RedisClient.Get(infohash).Result()
		if redisErr == redis.Nil {
			c := sniffer.Mon.DB(dataBase).C(collection)
			selector := bson.M{"infohash": infohash}
			err = c.Find(selector).One(&m)
			if m != nil {
				sniffer.RedisClient.Set(infohash, m["_id"].(bson.ObjectId), 0)
			}
			return
		} else if redisErr != nil {
			c := sniffer.Mon.DB(dataBase).C(collection)
			selector := bson.M{"infohash": infohash}
			err = c.Find(selector).One(&m)
		} else {
			m["_id"] = bson.ObjectId(val)
		}
	} else {
		c := sniffer.Mon.DB(dataBase).C(collection)
		selector := bson.M{"infohash": infohash}
		err = c.Find(selector).One(&m)
	}
	<-sniffer.mongoLimit
	return
}

func (sniffer *sn) syncmongodb(data bitTorrent) (err error) {

	c := sniffer.Mon.DB(dataBase).C(collection)
	err = c.Insert(data)
	<-sniffer.mongoLimit
	return
}

func (sniffer *sn) updateTimeHot(objectID bson.ObjectId) (err error) {

	c := sniffer.Mon.DB(dataBase).C(collection)

	m := make(map[string]interface{})
	m["$inc"] = map[string]int{"hot": 1}
	m["$set"] = map[string]int64{"last_time": time.Now().Unix()}

	selector := bson.M{"_id": objectID}
	err = c.Update(selector, m)
	<-sniffer.mongoLimit
	return
}

func loadBlackList() (blackList []string) {
	fi, err := os.Open(banList)

	if err != nil {
		fi.Close()
		log.Panicln("Error: %s\n", err)
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
