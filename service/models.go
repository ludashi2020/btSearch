package service

import (
	"github.com/Unknwon/goconfig"
	"github.com/caarlos0/env/v6"
	mapset "github.com/deckarep/golang-set"
	"github.com/gin-gonic/gin"
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
	Router       *gin.Engine
	hotSearchSet mapset.Set
	hotSearch    []hotSearchData
	total        int64
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

type Config struct {
	EsUsername           string `env:"EsUsername"`
	EsPassWord           string `env:"EsPassWord"`
	HotSearchOnePageSize int    `env:"HotSearchOnePageSize" envDefault:"6"`
	HotSearchPageSize    int    `env:"HotSearchPageSize" envDefault:"3"`
	AuthDataBase         string `env:"AuthDataBase"`
	EnableElasticsearch  bool   `env:"EnableElasticsearch"`
	EsURL                string `env:"EsURL"`
	EsUrlBase            string `env:"EsUrlBase"`
	WebServerAddr        string `env:"WebServerAddr"`
}

var (
	cfg        *goconfig.ConfigFile
	ConfigData *Config
)

func InitConfig() {
	var config Config
	if err := env.Parse(&config); err != nil {
		panic(err)
	}
	ConfigData = &config
	return
}
