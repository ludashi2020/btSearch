package main

import (
	"github.com/Bmixo/btSearch/server/common"
)

func main() {

	sniffer := common.NewSniffer()
	defer sniffer.Mon.Close()
	go sniffer.PrintLog()
	go sniffer.NewServerConn()
	go sniffer.Reboot()
	go sniffer.Metadata()
	sniffer.CheckSpeed()
}
