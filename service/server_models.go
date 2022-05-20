package service

import (
	"github.com/go-redis/redis"
	"github.com/paulbellamy/ratecounter"

	mapset "github.com/deckarep/golang-set"

	"github.com/go-ego/gse"
)

var (
	listenerAddr         = "0.0.0.0:9898"
	tdataChanSize        = 100
	mongoConnectLimitNum = 100
	metadataNum          = 10
	wkNodes              []string
	verifyPassord        = ""
	banList              = "banList.txt"

	redisEnable   = false
	redisAddr     = ""
	redisPassword = ""
	redisDB       = 0
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

type Server struct {
	segmenter   gse.Segmenter
	hashList    mapset.Set
	Nodes       []string
	Tool        Tool
	RedisClient *redis.Client
	revNum      *ratecounter.RateCounter
	dropSpeed   *ratecounter.RateCounter
	sussNum     *ratecounter.RateCounter
	notFoundNum *ratecounter.RateCounter
	blackList   []string
	mongoLimit  chan bool
	printChan   chan string
}
