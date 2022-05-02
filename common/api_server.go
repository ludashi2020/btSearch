package common

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/Bmixo/btSearch/api/api_server_1/torrent"
	"io"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Bmixo/btSearch/pkg/bencode"
	"github.com/go-ego/gse"
	"github.com/go-redis/redis"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func (m *Server) handleData() {
	for {
		tDataTmp := <-m.Tool.ToolRevChan
		if tDataTmp.TDataCode == torrent.TDataCode_TDataCodeVerifyHashExist {
			m.revNum.Incr(1)
			infoHash := tDataTmp.Hash

			if m.hashList.Contains(infoHash) {
				continue
			}
			select {
			case m.mongoLimit <- true:
			default:
				m.dropSpeed.Incr(1)
				continue
			}
			data, err := m.findHash(infoHash)
			if err != nil && err != mgo.ErrNotFound {
				m.printChan <- "\n" + "ERR:4511" + err.Error() + "\n"
				continue
			}

			if data != nil {
				select {
				case m.mongoLimit <- true:
				default:
					m.dropSpeed.Incr(1)
					continue
				}
				err = m.updateTimeHot(data["_id"].(bson.ObjectId))
				if err != nil {
					m.printChan <- "\n" + "update time hot ERR:0025" + err.Error() + "\n"
					continue
				}
				continue
			}
			if len(m.Tool.ToolSendChan) < cap(m.Tool.ToolSendChan) {
				tDataTmp.TDataCode = torrent.TDataCode_TDataCodeNeedDownload
				m.Tool.ToolSendChan <- tDataTmp
				m.hashList.Add(infoHash)
				m.notFoundNum.Incr(1)
			} else {
				m.dropSpeed.Incr(1)
			}
		} else if tDataTmp.TDataCode == torrent.TDataCode_TDataCodeNeedHandleTorrentData {
			torrent, err := m.newTorrent(tDataTmp.TorrentData, tDataTmp.Hash)
			if err != nil {
				continue
			}

			segments := m.segmenter.Segment([]byte(torrent.Name))
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
			case m.mongoLimit <- true:
			default:
				m.dropSpeed.Incr(1)
				continue
			}
			err = m.syncmongodb(torrent)

			if err != nil {
				continue
			}

			m.sussNum.Incr(1)
			m.hashList.Remove(torrent.InfoHash)
			//m.printChan <- ("------" + torrent.Name + "------" + torrent.InfoHash)
			continue
		}
	}
}

func (m *Server) NewServerConn() {
	for _, node := range wkNodes {
		m.Tool.Links = append(m.Tool.Links, Link{Conn: nil, Addr: node, LinkPostChan: make(chan torrent.TData, 1000)})
	}
	for i := 0; i < 10; i++ {
		go m.handleData()
	}
	m.Tool.LinksServe()

}

func (m *Server) Refresh() {
	for {
		time.Sleep(time.Second * 240)
		m.blackAddrList.Clear()
		m.hashList.Clear()
	}
}

func (m *Server) PrintLog() {

	for {
		//fmt.Printf("\r")
		fmt.Printf("%s", <-m.printChan)
	}

}

func (m *Server) CheckSpeed() {
	for {

		m.printChan <- "RevSpeed: " + strconv.FormatInt(m.revNum.Rate(), 10) + "/sec" +
			" DropSpeed: " + strconv.FormatInt(m.dropSpeed.Rate(), 10) + "/sec" +
			" NotFoundSpeed: " + strconv.FormatInt(m.notFoundNum.Rate(), 10) + "/sec" +
			" SussSpeed: " + strconv.FormatInt(m.sussNum.Rate(), 10) + "/sec" +
			" HashList:" + strconv.Itoa(m.hashList.Cardinality()) +
			" blackAddrList:" + strconv.Itoa(m.blackAddrList.Cardinality()) +
			"\n"
		time.Sleep(time.Second)
	}

}

func (m *Server) newTorrent(metadata []byte, InfoHash string) (torrent bitTorrent, err error) {
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

	for _, black := range m.blackList {
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
		var biggestFile fileServer
		files := v.([]interface{})
		bt.Files = make([]fileServer, len(files))
		var TotalLength int64

		bt.FileType = "Unknow"
		for i, item := range files {
			f := item.(map[string]interface{})

			if _, ok := f["length"].(int64); !ok {
				return bitTorrent{}, errors.New("length, not int64")
			}
			TotalLength = TotalLength + f["length"].(int64)
			if f["length"].(int64) > biggestFile.Length {
				biggestFile.Length = f["length"].(int64)
				biggestFile.Path = f["path"].([]interface{})
			}
			bt.Files[i] = fileServer{
				Path:   f["path"].([]interface{}),
				Length: f["length"].(int64),
			}
		}
		bt.Length = TotalLength
		sourceName = biggestFile.Path[len(biggestFile.Path)-1].(string)

	} else if _, ok := info["length"]; ok {
		bt.Length = info["length"].(int64)
		sourceName = bt.Name
	}
	bt.Extension = path.Ext(sourceName)

findName:
	for i, one := range cats {
		tmpLength := len(one)
		for j := 0; j < tmpLength; j++ {

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

func (m *Server) findHash(infoHash string) (result map[string]interface{}, err error) {
	if redisEnable {
		val, redisErr := m.RedisClient.Get(infoHash).Result()
		if redisErr == redis.Nil {
			c := m.Mon.DB(dataBase).C(collection)
			selector := bson.M{"infohash": infoHash}
			err = c.Find(selector).One(&result)
			if result != nil {
				m.RedisClient.Set(infoHash, result["_id"].(bson.ObjectId), 0)
			}
			return
		} else if redisErr != nil {
			c := m.Mon.DB(dataBase).C(collection)
			selector := bson.M{"infohash": infoHash}
			err = c.Find(selector).One(&result)
		} else {
			result["_id"] = bson.ObjectId(val)
		}
	} else {
		c := m.Mon.DB(dataBase).C(collection)
		selector := bson.M{"infohash": infoHash}
		err = c.Find(selector).One(&result)
	}
	<-m.mongoLimit
	return
}

func (m *Server) syncmongodb(data bitTorrent) (err error) {

	c := m.Mon.DB(dataBase).C(collection)
	err = c.Insert(data)
	<-m.mongoLimit
	return
}

func (m *Server) updateTimeHot(objectID bson.ObjectId) (err error) {

	c := m.Mon.DB(dataBase).C(collection)

	data := make(map[string]interface{})
	data["$inc"] = map[string]int{"hot": 1}
	data["$set"] = map[string]int64{"last_time": time.Now().Unix()}

	selector := bson.M{"_id": objectID}
	err = c.Update(selector, data)
	<-m.mongoLimit
	return
}

func loadBlackList() (blackList []string) {
	fi, err := os.Open(banList)

	if err != nil {
		fi.Close()
		log.Panicf("\nError: %s\n\n", err)
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
