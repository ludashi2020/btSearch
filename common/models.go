package common

import (
	"github.com/Unknwon/goconfig"
	mapset "github.com/deckarep/golang-set"
	"github.com/gin-gonic/gin"
	mgo "gopkg.in/mgo.v2"
)

type dbData struct {
	Title   string
	ID      string
	Rate    string
	Summary string
	Cover   string
}
type mvData struct {
	ID          string
	SlateURL    string
	SlateImgURL string
	Data        map[string]dbData
}
type hotSearchData struct {
	Flag string
	Data []mvData
}

type webServer struct {
	mon          *mgo.Session
	Router       *gin.Engine
	hotSearchSet mapset.Set
	hotSearch    []hotSearchData
	total        int
}

type torrentInfo struct {
	Name        string
	InfoHash    string
	thunderLink string
	ObjectID    string
	CreateTime  string
	Length      float32
	LengthType  string
	Category    string
}

type fileCommon struct {
	FilePath     string
	FileSize     float32
	FileSizeType string
}

var (
	mongoAddr            = ""
	mongoUsername        = ""
	mongoPassWord        = ""
	esUsername           = ""
	esPassWord           = ""
	hotSearchOnePageSize = 6
	hotSearchPageSize    = 3
	dataBase             = ""
	authDataBase         = ""
	collection           = ""
	esURL                = ""
	WebServerAddr        = ""
	cfg                  *goconfig.ConfigFile
)
