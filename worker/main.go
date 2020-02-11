package main

import (
	"github.com/Bmixo/btSearch/worker/common"
)

func main() {
	wk := common.NewServer()
	go wk.PrintLog()
	go wk.FindNode()
	go wk.GenerNodes()
	go wk.AutoSendFindNode()
	go wk.HandleMsg()
	wk.Server()
}
