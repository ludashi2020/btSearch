package main

import (
	"testing"
)

func TestPut(t *testing.T) {
	maxThread := make(chan int, 10)
	m := newMon()
	if m.Put("http://127.0.0.1:9200/bavbt/torrent/", []byte("{}"), 0, maxThread) != nil {
		t.Errorf("本地服务器初始化失败")
	}

}
