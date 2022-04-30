package common

import (
	"fmt"
	"github.com/Bmixo/btSearch/pkg/pongo2gin"
	mapset "github.com/deckarep/golang-set"
	elasticsearch "github.com/elastic/go-elasticsearch/v6"
	"github.com/gin-gonic/gin"
	mgo "gopkg.in/mgo.v2"
	"log"
	"os"
	"path/filepath"
	"time"
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
	for {
		time.Sleep(time.Second)
		log.Println("trying to connect es")
		var err error
		ES, err = elasticsearch.NewClient(elasticsearch.Config{
			Addresses: []string{esUrlBase},
			Username:  esUsername,
			Password:  esPassWord,
		})
		if err != nil {
			log.Printf("Error creating the client: %s", err)
			continue
		}
		res, err := ES.Info()
		if err != nil {
			log.Printf("Error getting response: %s", err)
			continue
		}
		res.Body.Close()
		log.Println("connect es suss")
		log.Println(res)
		break
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
