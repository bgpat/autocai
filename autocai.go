package main

import (
	"io"
	//"fmt"
	"strings"
	//"strconv"
	"net/http"
	"net/url"
	//"io/ioutil"
	"github.com/PuerkitoBio/goquery"
	"code.google.com/p/go.text/encoding/japanese"
	"code.google.com/p/go.text/transform"
)

func euc2utf8(src io.Reader) io.Reader {
	return transform.NewReader(src, japanese.EUCJP.NewDecoder())
}

func getList() []string {
	res, _ := http.PostForm(
		"http://webmizar.cs.shinshu-u.ac.jp/learn/infomath/",
		url.Values{},
	)
	defer res.Body.Close()
	doc, _ := goquery.NewDocumentFromReader(res.Body)
	url := []string{}
	doc.Find("table").Find("a").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		i := strings.Index(href, "?exe_dir=")
		if i != -1 {
			query := href[i:]
			url = append(url, "http://shirodanuki.cs.shinshu-u.ac.jp/cgi-bin/olts/sys/top.cgi" + query)
		}
	})
	return url
}

func main() {
	urls := getList()
	println(urls[0])
	res, _ := http.Get(urls[0])
	defer res.Body.Close()
	utf8 := euc2utf8(res.Body)
	doc, _ := goquery.NewDocumentFromReader(utf8)
	println(doc.Text())
	/*
	for i, url := range(urls) {
		println(strconv.Itoa(i) + ": " + url)
	}
	*/
}
