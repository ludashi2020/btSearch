package model

import "github.com/Bmixo/btSearch/model/torrent"

var Model *MModel

type MModel struct {
	Torrent *torrent.MTorrent
}

func Init() {
	Model = &MModel{
		Torrent: &torrent.MTorrent{},
	}
}
