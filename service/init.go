package service

import (
	"github.com/Bmixo/btSearch/pkg/pongo2gin"
	mapset "github.com/deckarep/golang-set"
	elasticsearch "github.com/elastic/go-elasticsearch/v6"
	"github.com/gin-gonic/gin"
	"log"
	"os"
	"path/filepath"
)

func checkErr(err error) {
	if err != nil {
		panic(err.Error())
	}
}

var ES *elasticsearch.Client

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func Init() {
	InitConfig()
	if ConfigData.EnableElasticsearch {
		InitEs()
	}
}

func NewServer() *webServer {

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.HTMLRender = pongo2gin.New(pongo2gin.RenderOptions{
		TemplateDir: "misc/templates",
		ContentType: "text/html; charset=utf-8",
	})
	wd, _ := os.Getwd()

	// fmt.Println(filepath.Join("misc", "static"))
	router.Static("static", filepath.Join(wd, "misc", "static"))
	router.Static("img", filepath.Join(wd, "misc", "img"))

	return &webServer{
		Router:       router,
		hotSearchSet: mapset.NewSet(),
	}
}
