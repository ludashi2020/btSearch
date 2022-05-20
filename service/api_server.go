package service

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Bmixo/btSearch/api/api_server_1/torrent"
	"github.com/Bmixo/btSearch/model"
	tt "github.com/Bmixo/btSearch/model/torrent"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

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
			data, err := model.Model.Torrent.FindTorrentByHash(infoHash)
			if err != nil && err != mongo.ErrNoDocuments {
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
				err = model.Model.Torrent.AddTorrentHot(data.ID.Hex())
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
			torrent.ID = primitive.NewObjectID()
			err = model.Model.Torrent.InsertTorrent(torrent)
			if err != nil {
				continue
			}
			objectId := torrent.ID.Hex()
			syncdata, err := json.Marshal(esData{
				Title:      torrent.Name,
				HashId:     torrent.InfoHash,
				Length:     torrent.Length,
				CreateTime: torrent.CreateTime,
				FileType:   strings.ToLower(torrent.FileType),
				Hot:        torrent.Hot,
			})
			err = EsPut(esURL+objectId, syncdata)
			if err != nil && strings.Contains(err.Error(), "500") {
				for i := 0; i < 20; i++ {
					//es第一次运行没有初始化时候可能出错
					err = EsPut(esURL+objectId, syncdata)
					if err != nil && strings.Contains(err.Error(), "500") {
						m.printChan <- fmt.Sprintln("update es error code 500,try again\n", err)
						continue
					}
					break
				}
			}
			if err != nil {
				m.printChan <- fmt.Sprintln("update es error,", err)
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
		m.hashList.Clear()
	}
}

func (m *Server) PrintLog() {
	disableLog := os.Getenv("disableLog") == "true"
	for {
		//fmt.Printf("\r")
		if disableLog {
			<-m.printChan
		} else {
			fmt.Printf("%s", <-m.printChan)
		}
	}

}

func (m *Server) CheckSpeed() {
	for {

		m.printChan <- "RevSpeed: " + strconv.FormatInt(m.revNum.Rate(), 10) + "/sec" +
			" DropSpeed: " + strconv.FormatInt(m.dropSpeed.Rate(), 10) + "/sec" +
			" NotFoundSpeed: " + strconv.FormatInt(m.notFoundNum.Rate(), 10) + "/sec" +
			" SussSpeed: " + strconv.FormatInt(m.sussNum.Rate(), 10) + "/sec" +
			" HashList:" + strconv.Itoa(m.hashList.Cardinality()) +
			"\n"
		time.Sleep(time.Second)
	}

}

func (m *Server) newTorrent(metadata []byte, InfoHash string) (torrent tt.BitTorrent, err error) {
	info, err := bencode.Decode(bytes.NewBuffer(metadata))
	if err != nil {
		return tt.BitTorrent{}, err
	}
	timestamp := time.Now().Unix()
	if _, ok := info["name"]; !ok {
		return tt.BitTorrent{}, errors.New("Metadata Name is Empty")
	}
	if t, ok := info["name"].(string); ok {
		if !utf8.Valid([]byte(t)) {
			return tt.BitTorrent{}, errors.New("Metadata Name is not Encode by utf-8")
		}
	} else {
		return tt.BitTorrent{}, errors.New("interface conversion: interface {} is int64, not string,90099")
	}

	for _, black := range m.blackList {
		if strings.Contains(info["name"].(string), black) {

			return tt.BitTorrent{}, errors.New("Metadata Name is in Blacklist")
		}
	}

	bt := tt.BitTorrent{
		ID:         primitive.NewObjectID(),
		InfoHash:   InfoHash,
		Name:       info["name"].(string),
		CreateTime: timestamp,
		LastTime:   timestamp,
	}

	var sourceName string
	if v, ok := info["files"]; ok {
		var biggestFile tt.FileServer
		files := v.([]interface{})
		bt.Files = make([]tt.FileServer, len(files))
		var TotalLength int64

		bt.FileType = "Unknow"
		for i, item := range files {
			f := item.(map[string]interface{})

			if _, ok := f["length"].(int64); !ok {
				return tt.BitTorrent{}, errors.New("length, not int64")
			}
			TotalLength = TotalLength + f["length"].(int64)
			if f["length"].(int64) > biggestFile.Length {
				biggestFile.Length = f["length"].(int64)
				biggestFile.Path = f["path"].([]interface{})
			}
			bt.Files[i] = tt.FileServer{
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
