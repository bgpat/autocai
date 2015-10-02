package main
//package resolve

import (
	"io"
	"strings"
	"io/ioutil"
	"net/http"
	"net/url"
	"database/sql"
	"github.com/PuerkitoBio/goquery"
	_ "github.com/mattn/go-sqlite3"
	"code.google.com/p/go.text/encoding/japanese"
	"code.google.com/p/go.text/transform"
)

func euc2utf(src io.Reader) io.Reader {
	return transform.NewReader(src, japanese.EUCJP.NewDecoder())
}

func utf2euc(str string) (string, error) {
	ret, err := ioutil.ReadAll(transform.NewReader(strings.NewReader(str), japanese.EUCJP.NewEncoder()))
	if err != nil {
		return "", err
	}
	return string(ret), err
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
	row := db.QueryRow("SELECT `tmp` FROM `process` WHERE `exe_dir` = ?", exe_dir)
	var tmp string
	row.Scan(&tmp)
	res, _ := http.PostForm("http://shirodanuki.cs.shinshu-u.ac.jp/cgi-bin/olts/sys/exercise.cgi",
	url.Values{
		"name": {"<>"},
		"id": {"<>"},
		"email": {""},
		"exe_dir": {exe_dir},
		"chapter": {""},
		"url": {"http://webmizar.cs.shinshu-u.ac.jp/learn/infomath/"},
		"tmp": {tmp},
	},)
	defer res.Body.Close()
	utf8 := euc2utf(res.Body)
	doc, _ := goquery.NewDocumentFromReader(utf8)
	question := strings.TrimSpace(doc.Find("blockquote").Text())
	tmp, _ = doc.Find("input[name=tmp]").Attr("value")
	answers := doc.Find("input[name=answer]")
	println(doc.Find("body > u > i").Text())
	var answer string
	if answers.Length() == 1 {
		stmt, _ := db.Prepare("SELECT `answer` FROM `cai` WHERE `exe_dir` = ? AND `question` = ?")
		defer stmt.Close()
		row := stmt.QueryRow(exe_dir, question)
		row.Scan(&answer)
	} else {
		stmt, _ := db.Prepare("SELECT `answer` FROM `cai` WHERE `exe_dir` = ? AND `question` = ? AND `answer` in (?, ?, ?)")
		defer stmt.Close()
		a1, _ := answers.Attr("value")
		a2, _ := answers.Next().Attr("value")
		a3, _ := answers.Next().Next().Attr("value")
		row := stmt.QueryRow(exe_dir, question, a1, a2, a3)
		row.Scan(&answer)
	}
	answer, _ = utf2euc(answer)
	res, _ = http.PostForm("http://shirodanuki.cs.shinshu-u.ac.jp/cgi-bin/olts/sys/answer.cgi",
	url.Values{
		"answer": {answer},
		"subject": {""},
		"chapter": {""},
		"url": {"http://webmizar.cs.shinshu-u.ac.jp/learn/infomath/"},
		"tmp": {tmp},
	},)
	defer res.Body.Close()
	utf8 = euc2utf(res.Body)
	doc, _ = goquery.NewDocumentFromReader(utf8)
	tmp, _ = doc.Find("input[name=tmp]").Attr("value")
	db.Exec("REPLACE INTO `process` (`exe_dir`, `tmp`) VALUES (?, ?)", exe_dir, tmp)
	if strings.Contains(doc.Text(), "おめでとうございます") {
		println(exe_dir)
	} else {
		crawl(exe_dir, db)
	}
}

func main() {
	db, _ := sql.Open("sqlite3", "./cai.db")
	defer db.Close()
	list := getList()
	for i := range(list) {
		crawl(list[i], db)
	}
}
