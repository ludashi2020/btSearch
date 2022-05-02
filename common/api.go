package common

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/buger/jsonparser"
	mapset "github.com/deckarep/golang-set"
	"github.com/flosch/pongo2"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func (server *webServer) Timer() {
	for {
		total, err := server.getStatus()
		if err != nil {
			continue
		}
		server.total = total
		time.Sleep(time.Minute)
	}

}

func locVerify(loc string, language string) (result string) {

	switch loc {
	case "zh":
		return "zh"
	case "en":
		return "en"
	default:
		if strings.Contains(language, "zh") {
			return "zh"
		}
		return "en"

	}
}

func (server *webServer) findMoiveByID(id string) (mvData, error) {

	for _, j := range server.hotSearch {
		for _, z := range j.Data {
			if z.ID == id {
				return z, nil
			}
		}

	}
	return mvData{}, errors.New("MoiveID Not Found")

}
func (server *webServer) Movie(c *gin.Context) {
	language := c.Request.Header.Get("Accept-Language")
	loc := "en"
	if strings.Contains(language, "zh") {
		loc = "zh"
	}
	id := c.Param("id")
	if !imdbIDverify(id) {
		c.HTML(http.StatusBadRequest, "404.html", pongo2.Context{})
		return
	}
	start := c.DefaultQuery("start", "1") //可设置默认值
	data, err := server.findMoiveByID(id)
	if err != nil { //过期热词 返回搜索列表 避免死链
		title := c.DefaultQuery("title", "")

		torrentList, begin, end, total, took, err := searchES(title, "video", start)
		if err != nil {
			c.HTML(http.StatusBadRequest, "404.html", pongo2.Context{})
			return
		}
		c.HTML(http.StatusOK, "search.html", pongo2.Context{
			"torrent_list": torrentList,
			"key_word":     title,
			"begin":        begin,
			"end":          end,
			"total":        total,
			"loc":          loc,
			"took":         float32(took) / 1000,
		})
		return
	}

	mvdata := data.Data[loc]
	kw := mvdata.Title
	torrentList, begin, end, total, took, err := searchES(kw, "video", start)
	if err != nil {
		c.HTML(http.StatusBadRequest, "404.html", pongo2.Context{})
		return
	}
	c.HTML(http.StatusOK, "movie.html", pongo2.Context{
		"torrent_list":  torrentList,
		"key_word":      kw,
		"loc":           loc,
		"movieID":       id,
		"begin":         begin,
		"end":           end,
		"mvdata":        mvdata,
		"total":         total,
		"trailerImgUrl": data.SlateImgURL,
		"trailerUrl":    data.SlateURL,
		"took":          float32(took) / 1000,
	})

}
func (server *webServer) About(c *gin.Context) {
	loc := "en"
	if strings.Contains(c.Request.Header.Get("Accept-Language"), "zh") {
		loc = "zh"
	}
	c.HTML(http.StatusOK, "about.html", pongo2.Context{
		"loc": loc,
	})
	return
}
func (server *webServer) Search(c *gin.Context) {
	kw := c.DefaultQuery("kw", "")
	loc := locVerify(c.DefaultQuery("loc", ""), c.Request.Header.Get("Accept-Language")) //可设置默认值
	category := c.DefaultQuery("category", "all")
	start := c.DefaultQuery("start", "1") //可设置默认值
	torrentList, begin, end, total, took, err := searchES(kw, category, start)
	if err != nil {
		c.HTML(http.StatusBadRequest, "404.html", pongo2.Context{})
		return
	}

	c.HTML(http.StatusOK, "search.html", pongo2.Context{
		"torrent_list": torrentList,
		"key_word":     kw,
		"begin":        begin,
		"end":          end,
		// "sort":         sort,
		"total": total,
		"loc":   loc,
		"took":  float32(took) / 1000,
	})
}
func searchES(kw, category, start string) (torrentList []torrentInfo, begin int, end int, total int, took float64, err error) {
	{
		begin, err = strconv.Atoi(start)
		if err != nil || begin <= 0 {
			return nil, begin, end, total, took, errors.New("start err 9990990")
		}
	}
	var resBytes []byte
	{

		var buf bytes.Buffer

		if true {
			mod := `[
		{"term": {"file_type": "` + category + `"}},
				  {"match": {
		  "title" : {
			"query":"` + kw + `",
			"operator":"and",
			"minimum_should_match": "50%"
					}
		  }
		}
	  ]`
			switch category {
			case "video":
			case "document":
			case "music":
			case "all":
				mod = `{
			"match": {
				"title": {
					"query": "` + kw + `",
					"operator": "and",
					"minimum_should_match": "50%"
				}
			}
		}`
			default:
				return nil, begin, end, total, took, errors.New("category err 980909")

			}

			sample := []byte(`{
		"query": {
			"bool": {
				"must": {} 
			}
		},
		"from": 0,
		"size": 15,
		"sort":{
			"_score":{"order":"desc"},
			"length":{"order":"desc"},
			"create_time":{"order":"desc"}
			}
	}`)
			//"hot":{"order":"desc"},

			dataTmp, _ := jsonparser.Set(sample, []byte(mod), "query", "bool", "must")
			dataTmp, _ = jsonparser.Set(dataTmp, []byte(strconv.Itoa((begin-1)*15)), "from")

			//result, err := get(esURL+"_search", dataTmp)
			//
			//if err != nil {
			//	return nil, begin, end, total, took, errors.New("ES search err 78798")
			//}

			buf.Write(dataTmp)
		}
		//TODO 使用go-es封装查询
		if false {
			query := map[string]interface{}{
				"query": map[string]interface{}{
					"match": map[string]interface{}{
						"title": kw,
					},
				},
				//"highlight": map[string]interface{}{
				//	"pre_tags" : []string{"<font color='red'>"},
				//	"post_tags" : []string{"</font>"},
				//	"fields" : map[string]interface{}{
				//		"title" : map[string]interface{}{},
				//	},
				//},
			}
			if err := json.NewEncoder(&buf).Encode(query); err != nil {
				return nil, begin, end, total, took, err
			}
		}
		// Perform the search request.
		res, err := ES.Search(
			ES.Search.WithContext(context.Background()),
			ES.Search.WithIndex("bavbt/torrent"),
			ES.Search.WithBody(&buf),
			ES.Search.WithPretty(),
		)
		if err != nil {
			return nil, begin, end, total, took, err
		}
		defer res.Body.Close()
		resBytes, err = ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, begin, end, total, took, err
		}
		fmt.Println(string(resBytes))
	}

	searchData := map[string]interface{}{}

	err = json.Unmarshal(resBytes, &searchData)
	if err != nil {
		return
	}
	took = searchData["took"].(float64)
	var left int
	total = int(searchData["hits"].(map[string]interface{})["total"].(float64))

	if begin <= 0 {
		return
	} else if total == 0 {
		end = -1
		left = 0
	} else if total >= 15 {
		if total%15 == 0 {
			end = total % 15
		} else {
			end = total/15 + 1
		}
		if total-begin*15 >= 15 {
			left = 15

		} else {
			left = total - (begin-1)*15
		}
	} else if total > 0 && total < 15 && (total-(begin-1)*15) > 0 {
		end = 0
		left = total

	}

	if left > 15 {
		left = 15
	}
	if end > 666 { //最大页数es决定
		end = 666
	}

	torrentSet := mapset.NewSet()

	for _, data := range searchData["hits"].(map[string]interface{})["hits"].([]interface{}) {
		one := data.(map[string]interface{})["_source"].(map[string]interface{})
		name := one["title"].(string)
		if torrentSet.Contains(name) {
			continue
		}
		createTime := time.Unix(int64(one["create_time"].(float64)), 0).Format("2006-01-02")
		length, lengthType := getsize(int64(one["length"].(float64)))
		hashLink := one["hash_id"].(string) + "&dn=" + name
		a := torrentInfo{
			Name:        name,
			thunderLink: magnet2Thunder("magnet:?xt=urn:btih:" + hashLink),
			InfoHash:    hashLink,
			ObjectID:    data.(map[string]interface{})["_id"].(string),
			CreateTime:  createTime,
			Length:      length,
			LengthType:  lengthType,
			Category:    one["file_type"].(string),
		}
		torrentList = append(torrentList, a)
		torrentSet.Add(a.Name)

	}
	//fmt.Println("Left:", left, "Begin:", begin, "End:", end, "Int_Total:", total, "::: Total:", total, "::: left:", left, "\n")

	return torrentList, begin, end, total, took, err

}

func (server *webServer) FilterGetdbDataValueByKey(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	m := param.Interface().(mvData)
	s := strings.Split(in.String(), "-")
	switch s[1] {
	case "Title":
		return pongo2.AsValue(m.Data[s[0]].Title), nil
	case "ID":
		return pongo2.AsValue(m.Data[s[0]].ID), nil
	case "Rate":
		return pongo2.AsValue(m.Data[s[0]].Rate), nil
	case "Summary":
		return pongo2.AsValue(m.Data[s[0]].Summary), nil
	case "Cover":
		return pongo2.AsValue(m.Data[s[0]].Cover), nil
	}
	return pongo2.AsValue(""), nil
}

func (server *webServer) FilterAddLoc(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return pongo2.AsValue(in.String() + param.String()), nil
}

func (server *webServer) Index(c *gin.Context) {
	loc := locVerify(c.DefaultQuery("loc", ""), c.Request.Header.Get("Accept-Language"))

	c.HTML(http.StatusOK, "index.html", pongo2.Context{
		"total":     server.total,
		"hotSearch": server.hotSearch,
		"loc":       loc,
	})
}

func (server *webServer) Details(c *gin.Context) {

	objectid := strings.Replace(c.Param("objectid"), ".html", "", -1)
	loc := locVerify(c.DefaultQuery("loc", ""), c.Request.Header.Get("Accept-Language"))

	if len(objectid) == 24 {
		torrentData, err := server.find(objectid)
		if err != nil {
			c.HTML(http.StatusNotFound, "404.html", pongo2.Context{})
		} else {

			var tmKeyword []string
			for _, keyword := range torrentData["key_word"].([]interface{}) {
				tmKeyword = append(tmKeyword, keyword.(string))
			}

			var files []fileCommon
			var filenum int

			for _, one := range torrentData["files"].([]interface{}) {

				var fileTmp fileCommon
				var ignore bool

				for i, path := range one.(map[string]interface{})["path"].([]interface{}) {

					if strings.Contains(path.(string), "如果您看到此文件，请升级到BitComet(比特彗星)0.85或以上版本") {
						ignore = true
						continue
					}

					if i == 0 {
						fileTmp.FilePath = path.(string)

					} else {

						fileTmp.FilePath = fileTmp.FilePath + "/" + path.(string)
					}

					var length int64

					if lens, ok := one.(map[string]interface{})["length"].(int); ok {
						length = int64(lens)
					} else {
						length = one.(map[string]interface{})["length"].(int64)
					}

					fileTmp.FileSize, fileTmp.FileSizeType = getsize(length)
					filenum = filenum + 1
				}

				if !ignore {
					files = append(files, fileTmp)
				}

			}

			totalLengh, lengthType := getsize(torrentData["length"].(int64))

			infohash := torrentData["infohash"].(string)
			name := torrentData["name"].(string)
			c.HTML(http.StatusOK, "details.html", pongo2.Context{
				"Name":            name,
				"Infohash":        infohash,
				"thunderLink":     magnet2Thunder("magnet:?xt=urn:btih:" + infohash + "&dn=" + name),
				"Hot":             torrentData["hot"].(int),
				"CreateTime":      time.Unix(torrentData["create_time"].(int64), 0).Format("2006-01-02"),
				"LastTime":        time.Unix(torrentData["last_time"].(int64), 0).Format("2006-01-02"),
				"TotalLength":     totalLengh,
				"TotalLengthType": lengthType,
				"FileNum":         filenum,
				"Files":           files,
				"Tag":             tmKeyword,
				"loc":             loc,
				"Category":        torrentData["category"].(string),
			})

		}

	} else {
		c.Redirect(http.StatusMovedPermanently, "http://bt.bmixo.com")
	}
}

func (server *webServer) find(id string) (data map[string]interface{}, err error) {
	for _, j := range id {
		if !((48 <= j && j <= 57) || (65 <= j && j <= 90) || (97 <= j && j <= 122)) {
			return nil, errors.New("database inject")
		}
	}

	session := server.mon.Clone()
	c := session.DB(dataBase).C(collection)
	selector := bson.M{"_id": bson.ObjectIdHex(id)}
	//data := bson.M{"hot": 100}
	err = c.Find(selector).One(&data)
	if err != nil {
		return nil, errors.New("Mongodb Find ERROR")
	}
	session.Close()
	return
}

func get(url string, indata []byte) (data []byte, err error) {
	client := &http.Client{}

	post := bytes.NewBuffer(indata)
	req, err := http.NewRequest("GET", url, post)
	if err != nil {
		return []byte{}, err
	}

	req.SetBasicAuth(esUsername, esPassWord)

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return []byte{}, err
	}
	defer resp.Body.Close()

	data, err = ioutil.ReadAll(resp.Body)
	return data, err
}
func getsize(i int64) (float32, string) {

	if i >= 1073741824 {
		return float32(i>>30) + float32((i%1073741824>>20))*0.001, "GB"
	} else if i >= 1048576 {
		return float32(i >> 20), "MB"
	} else if i >= 1024 {
		return float32(i >> 10), "KB"
	} else {
		return float32(i), "B"
	}

}

// func hotWords() (words []string, err error) {
// 	rev, err := http.Get("https://movie.douban.com/j/search_subjects?type=movie&tag=%E7%83%AD%E9%97%A8&sort=recommend&page_limit=20&page_start=0")

// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rev.Body.Close()

// 	body, err := ioutil.ReadAll(rev.Body)
// 	if err != nil {
// 		return nil, err
// 	}
// 	data := map[string]interface{}{}
// 	err = json.Unmarshal(body, &data)

// 	for _, hotWord := range data["subjects"].([]interface{}) {

// 		words = append(words, hotWord.(map[string]interface{})["title"].(string))

// 	}
// 	return

// }

func dbSpider(n int) (result []mvData, err error) {
	rev, err := http.Get("https://movie.douban.com/j/search_subjects?type=movie&tag=%E7%83%AD%E9%97%A8&sort=recommend&page_limit=20&page_start=" + strconv.Itoa(n))
	if err != nil {
		return nil, err
	}
	defer rev.Body.Close()

	body, err := ioutil.ReadAll(rev.Body)

	if err != nil {
		return nil, err
	}
	data := map[string]interface{}{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}
	for _, one := range data["subjects"].([]interface{}) {

		dbsummary, imdbID, err := dbDetail(one.(map[string]interface{})["id"].(string))
		if err != nil {
			return nil, err
		}

		imdata, TrailerURL, trailerImg, err := imdbDeatil(imdbID)
		if err != nil {
			return nil, err
		}

		result = append(result, mvData{
			ID:          imdbID,
			SlateURL:    TrailerURL,
			SlateImgURL: trailerImg,
			Data: map[string]dbData{
				"en": imdata,
				"zh": dbData{
					Title:   fixHotWord(one.(map[string]interface{})["title"].(string)),
					ID:      one.(map[string]interface{})["id"].(string),
					Rate:    one.(map[string]interface{})["rate"].(string),
					Cover:   one.(map[string]interface{})["cover"].(string),
					Summary: dbsummary,
				},
			},
		})

	}
	return
}

func imdbPhotoFix(url string) (result string) {

	re, err := regexp.Compile(`(.*?)@`)
	if err != nil {
		return url
	}
	result = re.FindString(url)
	if result == "" {
		return url
	}
	return result
}
func imdbDeatil(id string) (imdata dbData, TrailerURL, trailerImg string, err error) {

	doc, err := goquery.NewDocument("https://www.imdb.com/title/tt" + id + "/")
	if err != nil {
		return dbData{}, "", "", errors.New("76899")
	}
	summary := doc.Find("div.summary_text").Text()
	slate := doc.Find("div.slate")
	TrailerURL, _ = slate.Find("a").Attr("href")
	trailerImg, _ = slate.Find("img").Attr("src")

	cover, exists := doc.Find("div.poster").Find("img").Attr("src")
	if exists {
		cover = imdbPhotoFix(cover)
	}

	imdata = dbData{
		Title:   doc.Find("div.title_wrapper").Find("h1").Text(),
		ID:      id,
		Rate:    doc.Find("div.ratingValue").Find("strong").Find("span").Text(),
		Summary: summary,
		Cover:   cover,
	}
	return
}

func dbDetail(id string) (summary string, imdbID string, err error) {

	doc, err := goquery.NewDocument("https://movie.douban.com/subject/" + id + "/")
	if err != nil {
		return "", "", errors.New("899090")
	}
	summary = doc.Find("#link-report").Find("span").Text()
	summary = strings.TrimSpace(strings.Replace(summary, "©豆瓣", "", -1))
	c := doc.Find("#info").Find("[rel='nofollow']").Text()
	re, err := regexp.Compile(`tt(\d*)`)
	if err != nil {
		return "", "", errors.New("890090")
	}
	imdbID = re.FindString(c)
	if len(imdbID) < 2 {
		return "", "", errors.New("78899")
	}
	if !imdbIDverify(imdbID[2:]) {
		return "", "", errors.New("67789")
	}
	return summary, imdbID[2:], nil
}
func imdbIDverify(id string) bool {

	if len(id) > 10 {
		return false
	}
	for _, j := range id {
		if j < 48 || j > 57 {
			return false
		}
	}
	return true
}

// type hotWordsJSON struct {
// 	TableName string
// 	HotWords  []string
// }

func fixHotWord(hotword string) (words string) {

	if strings.Contains(hotword, "：") {
		return hotword[:strings.LastIndex(hotword, "：")]

	}
	return hotword

}

// func (server *webServer) getHotWord() (words []string) {
// 	session := server.mon.Clone()
// 	c := session.DB(dataBase).C("etc")
// 	selector := bson.M{"tablename": "HotWords"}
// 	var m map[string]interface{}

// 	err := c.Find(selector).One(&m)
// 	if err != nil {
// 		return []string{}
// 	}
// 	total := 0
// 	for _, j := range m["hotwords"].([]interface{}) {

// 		word := j.(string)
// 		total = total + len([]rune(word))
// 		if total > 50 {
// 			break
// 		}
// 		words = append(words, word)
// 	}
// 	return

// }
// func (server *webServer) updateHotWords() {

// 	for {
// 		hotwords, err := hotWords()
// 		if err != nil {
// 			continue
// 		}
// 		server.hotWords = hotwords
// 		time.Sleep(time.Hour)

// 	}
// }

// func (server *webServer) updateHotWords() {
// 	selector := bson.M{"tablename": "HotWords"}
// 	for {
// 		session := server.mon.Clone()
// 		c := session.DB(dataBase).C("etc")

// 		var m map[string]interface{}

// 		err := c.Find(selector).One(&m)
// 		if err == mgo.ErrNotFound {
// 			var words hotWordsJSON
// 			words.TableName = "HotWords"
// 			words.HotWords, err = hotWords()
// 			fmt.Println("Create collection: hotWords")
// 			c.Insert(words)

// 		} else if err != nil {
// 			fmt.Println(err, "1232311")
// 			session.Close()
// 			continue
// 		} else {
// 			m["hotwords"], err = hotWords()
// 			if err != nil {
// 				session.Close()
// 				continue
// 			}
// 			fmt.Println("Update Hot Words")
// 			c.Update(selector, m)
// 		}
// 		session.Close()
// 		time.Sleep(time.Hour)
// 	}
// }
func (server *webServer) syncHotSearch() (result []mvData) {
	server.hotSearchSet.Clear()

	for n := 0; ; n += 20 {
		data, err := dbSpider(n)
		if err != nil {
			fmt.Println("IP ERR ,sleep 2 min " + err.Error())
			time.Sleep(2 * time.Minute)
			continue
		}
		for _, j := range data {
			if server.hotSearchSet.Cardinality() >= hotSearchPageSize*hotSearchOnePageSize {
				return
			}
			if !server.hotSearchSet.Contains(j.ID) {
				//搜索有记录
				_, _, _, totalen, _, err := searchES(j.Data["en"].Title, "video", "1")
				_, _, _, totalzh, _, err := searchES(j.Data["zh"].Title, "video", "1")
				if err == nil {
					if totalen != 0 && totalzh != 0 {
						server.hotSearchSet.Add(j.ID)
						result = append(result, j)
					}
				}
			}

		}
	}
}
func (server *webServer) SyncDbHotSearchTimer() {

	for {
		data := server.syncHotSearch()
		server.hotSearch = []hotSearchData{}
		num := 0
		for i := 0; i < hotSearchPageSize; i++ {
			fg := ""
			if i == 0 {
				fg = "active"
			}
			server.hotSearch = append(server.hotSearch, hotSearchData{
				Flag: fg,
				Data: data[i*hotSearchOnePageSize : (i+1)*hotSearchOnePageSize],
			})
			num++
		}
		fmt.Println("Collect MvInfo Suss.......")
		time.Sleep(2 * time.Hour)
	}

}
func (server *webServer) getStatus() (result int, err error) {
	session := server.mon.Clone()
	data := bson.M{}
	if err := session.DB(dataBase).Run("dbstats", &data); err != nil {
		return 0, errors.New("Mongodb ERR")
	}
	session.Close()
	return data["objects"].(int), nil

}

func magnet2Thunder(murl string) string {
	return "thunder://" + base64.StdEncoding.EncodeToString([]byte("AA"+murl+"ZZ"))
}
