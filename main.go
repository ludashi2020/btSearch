package main

import (
	"runtime"

	"github.com/Bmixo/btSearch/common"
	"github.com/flosch/pongo2"
)

func main() {
	common.InitCommon()
	runtime.GOMAXPROCS(runtime.NumCPU())
	server := common.NewServer()
	pongo2.RegisterFilter("locFilter", server.FilterAddLoc)
	pongo2.RegisterFilter("keyFilter", server.FilterGetdbDataValueByKey)
	server.Router.GET("/search", server.Search)
	server.Router.GET("/movie/:id", server.Movie)
	server.Router.GET("/about", server.About)
	server.Router.GET("/", server.Index)
	server.Router.GET("/details/:objectid", server.Details)

	go server.Timer()
	go server.SyncDbHotSearchTimer()

	server.Router.Run(common.WebServerAddr)

}
