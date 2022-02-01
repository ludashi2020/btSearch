package common

import (
	"github.com/Bmixo/btSearch/model"
	"github.com/Unknwon/goconfig"
	"github.com/go-redis/redis"
	"github.com/paulbellamy/ratecounter"

	mapset "github.com/deckarep/golang-set"

	"github.com/go-ego/gse"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	listenerAddr         = "0.0.0.0:9898"
	tdataChanSize        = 100
	mongoConnectLimitNum = 100
	metadataNum          = 10
	mongoAddr            = ""
	wkNodes              = []string{}
	verifyPassord        = ""
	banList              = "banList.txt"
	mongoUsername        = ""
	mongoPassWord        = ""
	dataBase             = ""
	collection           = ""
	redisEnable          = false
	cfg                  *goconfig.ConfigFile
	redisAddr            = ""
	redisPassword        = ""
	redisDB              = 0
)

var (
	typeVideo    = []string{".avi", ".mp4", ".rmvb", ".m2ts", ".wmv", ".mkv", ".flv", ".qmv", ".rm", ".mov", ".vob", ".asf", ".3gp", ".mpg", ".mpeg", ".m4v", ".f4v"}
	typeImage    = []string{".jpg", ".bmp", ".jpeg", ".png", ".gif", ".tiff"}
	typeDocument = []string{".pdf", ".isz", ".chm", ".txt", ".epub", ".bc!", ".doc", ".ppt", ".mobi", ".awz", "rtf", "fb2"}
	typeMusic    = []string{".mp3", ".ape", ".wav", ".dts", ".mdf", ".flac", ".m4a"}
	typePackage  = []string{".zip", ".rar", ".7z", ".tar.gz", ".iso", ".dmg", ".pkg"}
	typeSoftware = []string{".exe", ".app", ".msi", ".apk"}
	cats         = [][]string{typeVideo, typeImage, typeDocument, typeMusic, typePackage, typeSoftware}
)

type file struct {
	Path   []interface{} `bson:"path"`
	Length int64         `bson:"length"`
}

type bitTorrent struct {
	ID         bson.ObjectId `bson:"_id"`
	InfoHash   string        `bson:"infohash"`
	Name       string        `bson:"name"`
	Extension  string        `bson:"extension"`
	Files      []file        `bson:"files"`
	Length     int64         `bson:"length"`
	CreateTime int64         `bson:"create_time"`
	LastTime   int64         `bson:"last_time"`
	Hot        int           `bson:"hot"`
	FileType   string        `bson:"category"`
	KeyWord    []string      `bson:"key_word"`
}
type Server struct {
	segmenter     gse.Segmenter
	hashList      mapset.Set
	blackAddrList mapset.Set
	Nodes         []string
	Tool          Tool
	Mon           *mgo.Session
	RedisClient   *redis.Client
	collection    *mgo.Collection
	revNum        *ratecounter.RateCounter
	dropSpeed     *ratecounter.RateCounter
	sussNum       *ratecounter.RateCounter
	notFoundNum   *ratecounter.RateCounter
	blackList     []string
	mongoLimit    chan bool
	printChan     chan string
	tdataChan     chan header.Tdata
}
