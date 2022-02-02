package common

import (
	"fmt"
	"github.com/Bmixo/btSearch/pkg/pongo2gin"
	elasticsearch "github.com/elastic/go-elasticsearch/v6"
	"log"
	"os"
	"path/filepath"

	mapset "github.com/deckarep/golang-set"
	"github.com/gin-gonic/gin"
	mgo "gopkg.in/mgo.v2"
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

func init() {
	mongoAddr = os.Getenv("mongoAddr")
	dataBase = os.Getenv("mongoDatabase")
	collection = os.Getenv("mongoCollection")
	mongoUsername = os.Getenv("mongoUsername")
	mongoPassWord = os.Getenv("mongoPassWord")
	esUsername = os.Getenv("esUsername")
	esPassWord = os.Getenv("esPassWord")
	esURL = os.Getenv("esURL")
	WebServerAddr = os.Getenv("webServerAddr")
	esUrlBase := os.Getenv("esUrlBase")
	{
		var err error
		ES, err = elasticsearch.NewClient(elasticsearch.Config{
			Addresses: []string{esUrlBase},
			Username:  esUsername,
			Password:  esPassWord,
		})
		if err != nil {
			log.Fatalf("Error creating the client: %s", err)
		}
		res, err := ES.Info()
		if err != nil {
			log.Fatalf("Error getting response: %s", err)
		}
		defer res.Body.Close()
		log.Println(res)
	}
}

func NewServer() *webServer {
	dialInfo := &mgo.DialInfo{
		Addrs:    []string{mongoAddr},
		Direct:   false,
		Database: dataBase,
		Source:   collection,
		Username: mongoUsername,
		Password: mongoPassWord,
	}

	session, err := mgo.DialWithInfo(dialInfo)
	if err != nil {
		panic(err)
	}
	session.SetPoolLimit(10)
	session.SetMode(mgo.Monotonic, true)
	fmt.Println("Mongodb load suss")

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
		mon:          session,
		Router:       router,
		hotSearchSet: mapset.NewSet(),
	}
}
