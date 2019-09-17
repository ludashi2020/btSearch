package main

import (
	"github.com/Bmixo/btSearch/worker/common"
)

func main() {
	wk := common.NewServer()
	go wk.HandleConn()
	go wk.PrintLog()
	go wk.FindNode()
	go wk.Server()
	go wk.GenerNodes()
	go wk.AutoSendFindNode()
	wk.HandleMsg()
}
