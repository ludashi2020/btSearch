package common

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
)

type esData struct {
	Title      string `json:"title"`
	HashId     string `json:"hash_id"`
	Length     int64  `json:"length"`
	CreateTime int64  `json:"create_time"`
	FileType   string `json:"file_type"`
	Hot        int    `json:"hot"`
}

func EsPut(url string, data []byte) (err error) {
	client := http.Client{}
	req, err := http.NewRequest("PUT", url, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	io.Copy(ioutil.Discard, resp.Body)
	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		resp.Body.Close()
		return errors.New(fmt.Sprint("error code", resp.StatusCode))
	}
	resp.Body.Close()
	return nil
}
