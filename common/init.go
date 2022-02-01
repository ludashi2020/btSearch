package common

import (
	"flag"
	"fmt"
	"github.com/Bmixo/btSearch/pkg/pongo2gin"
	"os"
	"path/filepath"

	"github.com/Unknwon/goconfig"
	mapset "github.com/deckarep/golang-set"
	"github.com/gin-gonic/gin"
	mgo "gopkg.in/mgo.v2"
)

func checkErr(err error) {
	if err != nil {
		panic(err.Error())
	}
}
func init() {
	confPath := flag.String("c", "config/server.conf", "web server config file")

	flag.Parse()

	config, err := goconfig.LoadConfigFile(*confPath)
	if err != nil {
		fmt.Println("Config file not exist")
		os.Exit(-1)
	}
	cfg = config
	mongoAddr, err = cfg.GetValue("mongodb", "addr")
	checkErr(err)
	dataBase, err = cfg.GetValue("mongodb", "database")
	checkErr(err)
	collection, err = cfg.GetValue("mongodb", "collection")
	checkErr(err)
	mongoUsername, err = cfg.GetValue("mongodb", "musername")
	checkErr(err)
	mongoPassWord, err = cfg.GetValue("mongodb", "mpassword")
	checkErr(err)
	esUsername, err = cfg.GetValue("elasticsearch", "eusername")
	checkErr(err)
	esPassWord, err = cfg.GetValue("elasticsearch", "epassword")
	checkErr(err)
	esURL, err = cfg.GetValue("elasticsearch", "url")
	checkErr(err)
	WebServerAddr, err = cfg.GetValue("webServer", "webServerAddr")
	checkErr(err)
}
func NewServer() *webServer {
	dialInfo := &mgo.DialInfo{
		Addrs:    []string{mongoAddr},
		Direct:   false,
		Database: dataBase,
		Source:   collection,
		Username: mongoUsername,
		Password: mongoUsername,
	}

	session, err := mgo.DialWithInfo(dialInfo)
	session.SetPoolLimit(10)
	session.SetMode(mgo.Monotonic, true)

	if err != nil {
		panic(err.Error)
	}
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
