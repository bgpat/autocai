package main

//package resolve

import (
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
	"database/sql"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	_ "github.com/mattn/go-sqlite3"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
	"bufio"
	"os"
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
			query := href[i+9:]
			url = append(url, query)
		}
	})
	return url
}

func resolve(exe_dir string, db *sql.DB, name, id, email string, t int) {
	time.Sleep(time.Duration(t) * time.Millisecond * 100)
	row := db.QueryRow("SELECT `tmp` FROM `process` WHERE `exe_dir` = ?", exe_dir)
	var tmp string
	row.Scan(&tmp)
	res, _ := http.PostForm("http://shirodanuki.cs.shinshu-u.ac.jp/cgi-bin/olts/sys/exercise.cgi",
		url.Values{
			"name":    {name},
			"id":      {id},
			"email":   {email},
			"exe_dir": {exe_dir},
			"chapter": {""},
			"url":     {"http://webmizar.cs.shinshu-u.ac.jp/learn/infomath/"},
			"tmp":     {tmp},
		})
	defer res.Body.Close()
	utf8 := euc2utf(res.Body)
	doc, _ := goquery.NewDocumentFromReader(utf8)
	html, _ := doc.Find("blockquote").Html()
	question := strings.TrimSpace(html)
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
		stmt, _ := db.Prepare("SELECT `answer` FROM `cai` WHERE `exe_dir` = ? AND `question` = ? AND `answer` in (?, ?, ?, ?)")
		defer stmt.Close()
		a1, _ := answers.Attr("value")
		a2, _ := answers.Next().Next().Attr("value")
		a3, _ := answers.Next().Next().Next().Next().Attr("value")
		a4, _ := answers.Next().Next().Next().Next().Next().Next().Attr("value")
		row := stmt.QueryRow(exe_dir, question, a1, a2, a3, a4)
		row.Scan(&answer)
	}
	answer, _ = utf2euc(answer)
	res, _ = http.PostForm("http://shirodanuki.cs.shinshu-u.ac.jp/cgi-bin/olts/sys/answer.cgi",
		url.Values{
			"answer":  {answer},
			"subject": {""},
			"chapter": {""},
			"url":     {"http://webmizar.cs.shinshu-u.ac.jp/learn/infomath/"},
			"tmp":     {tmp},
		})
	defer res.Body.Close()
	utf8 = euc2utf(res.Body)
	doc, _ = goquery.NewDocumentFromReader(utf8)
	tmp, _ = doc.Find("input[name=tmp]").Attr("value")
	db.Exec("REPLACE INTO `process` (`exe_dir`, `tmp`) VALUES (?, ?)", exe_dir, tmp)
	if strings.Contains(doc.Text(), "おめでとうございます") {
		println(exe_dir)
	} else {
		resolve(exe_dir, db, name, id, email, t)
	}
}

func main() {
	db, _ := sql.Open("sqlite3", "./cai.db")
	defer db.Close()
	stdin := bufio.NewScanner(os.Stdin)
	fmt.Printf("name > ")
	stdin.Scan()
	name := stdin.Text()
	fmt.Printf("id > ")
	stdin.Scan()
	id := stdin.Text()
	fmt.Printf("email > ")
	stdin.Scan()
	email := stdin.Text()
	list := getList()
	for i := range list {
		fmt.Printf("[%d] %s\n", i, list[i])
	}
	var n, t int
	fmt.Printf("chapter > ")
	fmt.Scanf("%d", &n)
	fmt.Printf("time[sec] > ")
	fmt.Scanf("%d", &t)
	resolve(list[n], db, name, id, email, t)
}
