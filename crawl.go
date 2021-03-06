package main

import (
	"io"
	"fmt"
	"strings"
	"net/http"
	"net/url"
	"database/sql"
	"github.com/PuerkitoBio/goquery"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
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
			query := href[i + 9:]
			url = append(url, query)
		}
	})
	return url
}

func crawl(exe_dir string, db *sql.DB) {
	res, _ := http.PostForm("http://shirodanuki.cs.shinshu-u.ac.jp/cgi-bin/olts/sys/exercise.cgi",
		url.Values{
			"name": {"hoge"},
			"id": {"hogehoge"},
			"email": {""},
			"exe_dir": {exe_dir},
			"chapter": {""},
			"url": {"http://webmizar.cs.shinshu-u.ac.jp/learn/infomath/"},
		},
	)
	defer res.Body.Close()
	utf8 := euc2utf8(res.Body)
	doc, _ := goquery.NewDocumentFromReader(utf8)
	html, _ := doc.Find("blockquote").Html()
	question := strings.TrimSpace(html)
	tmp, _ := doc.Find("input[name=tmp]").Attr("value")
	res, _ = http.PostForm("http://shirodanuki.cs.shinshu-u.ac.jp/cgi-bin/olts/sys/answer.cgi",
		url.Values{
			"answer": {""},
			"subject": {""},
			"chapter": {""},
			"url": {"http://webmizar.cs.shinshu-u.ac.jp/learn/infomath/"},
			"tmp": {tmp},
		},
	)
	defer res.Body.Close()
	utf8 = euc2utf8(res.Body)
	doc, _ = goquery.NewDocumentFromReader(utf8)
	answer := strings.TrimSpace(doc.Find("blockquote tt b").Text())
	stmt, _ := db.Prepare("INSERT INTO `cai` (`exe_dir`, `question`, `answer`) VALUES (?, ?, ?)")
	stmt.Exec(exe_dir, question, answer)
}

func main() {
	db, _ := sql.Open("sqlite3", "./cai.db")
	defer db.Close()
	db.Exec("CREATE TABLE `cai` (`id` integer PRIMARY KEY AUTOINCREMENT, `exe_dir` text, `question` text, `answer` text, UNIQUE (`exe_dir`, `question`, `answer`))")
	db.Exec("CREATE TABLE `process` (`exe_dir` text PRIMARY KEY, `tmp` text)")
	list := getList()
	for i := range list {
		fmt.Printf("[%d] %s\n", i, list[i])
	}
	var c int
	n := 100
	fmt.Printf("chapter [all scan] > ")
	once, _ := fmt.Scanf("%d", &c)
	fmt.Printf("how many times [100] > ")
	fmt.Scanf("%d", &n)
	for i := range(list) {
		if (once == 0 || c == i) {
			for j := 0; j < n; j++ {
				crawl(list[i], db)
			}
			println(list[i])
		}
	}
}
