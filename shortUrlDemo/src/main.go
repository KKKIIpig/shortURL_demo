package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"net/http"
)

var elements = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
var db *sql.DB

type returnData struct {
	ShortUrl string
	LongUrl  string
}

//这里首字母一定要大写
//json数据与struct字段相匹配时，会查找与json中key相同的可导出字段(首字母大写)
type postData struct {
	LongUrl string
}

func main() {
	Init() //连接数据库
	//HandleFunc 的第一个参数指的是请求路径，第二个参数是一个函数类型，表示这个请求需要处理的事情。
	http.HandleFunc("/", simpleRoute)        //HandleFunc注册一个处理器函数handler和对应的模式pattern（把自定义处理业务的函数进行路由注册）。
	err := http.ListenAndServe(":8080", nil) //第一个参数是监听的端口、第二个参数是根页面的处理函数，可以为空。
	if err != nil {
		panic(err)
	}
}

//数据库
func Init() {
	db, _ = sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/shortener")
	//defer db.Close()
}

//ID 十进制转换成62进制
func base10ToBase62(n int64) string {
	var str string
	for n != 0 {
		str += string(elements[n%62])
		n /= 62
	}

	for len(str) != 5 {
		str += "0"
	}
	return str
}

//returnData
func buildResponse(p postData) returnData {
	stmtIns, err := db.Prepare("INSERT INTO urls VALUES (null, '', ?, 0)")
	if err != nil {
		panic(err.Error())
	}
	defer stmtIns.Close()

	result, err := stmtIns.Exec(p.LongUrl)
	if err != nil {
		panic(err.Error())
	}
	id, _ := result.LastInsertId()
	shorter := base10ToBase62(id) //根据id的值会转换为62进制

	stmtUps, err := db.Prepare("UPDATE urls SET short_url=? WHERE id=?")
	if err != nil {
		panic(err.Error())
	}
	defer stmtUps.Close()

	stmtUps.Exec(shorter, id)

	var returns returnData
	returns.LongUrl = p.LongUrl
	returns.ShortUrl = shorter
	return returns
}

//longUrl -> shortUrl
func shortUrl(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close() //defer 表示对紧跟其后的 xxx() 函数延迟到 defer 语句所在的当前函数返回时再进行调用
	var p postData
	data, _ := ioutil.ReadAll(r.Body)
	/*1.Unmarshal的第一个参数是json字符串，第二个参数是接受json解析的数据结构。
	  第二个参数必须是指针，否则无法接收解析的数据
	  2.可以直接p:=new(postData),此时的p自身就是指针*/
	json.Unmarshal(data, &p)
	returnData := buildResponse(p) //获取shortUrl
	returnData.ShortUrl = "http://" + r.Host + "/" + returnData.ShortUrl
	jsonData, err := json.Marshal(returnData) //将数据编码成json字符串
	if err != nil {
		fmt.Println("json error:", err)
	}
	w.Write([]byte(jsonData))
}

//处理器
func simpleRoute(w http.ResponseWriter, r *http.Request) {
	urlPath := r.URL.Path
	if len(urlPath) == 6 {
		redirectUrl(w, r)
	} else if urlPath == "/shortener" {
		shortUrl(w, r)
	} else {
		w.WriteHeader(404)
	}
}

//shortUrl -> longUrl
func redirectUrl(w http.ResponseWriter, r *http.Request) {
	urlPath := r.URL.Path
	rs := []rune(urlPath)
	urlPath = string(rs[1:])
	stmtOut, err := db.Prepare("SELECT long_url FROM urls WHERE short_url=?") //查找与之对应的longurl
	if err != nil {
		panic(err.Error())
	}
	defer stmtOut.Close()

	var url string
	err = stmtOut.QueryRow(urlPath).Scan(&url)
	if err != nil {
		panic(err.Error())
	}
	http.Redirect(w, r, "http://"+url, 301)
}
