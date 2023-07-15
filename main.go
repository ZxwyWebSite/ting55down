package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// 程序版本号
const version string = `dev`

// 初始化常量 (抓取参数)
const (
	rooturl   string = `https://m.ting55.com/book/`  // PC版播放器有样式不好搞，故从手机版页面抓取
	listdom   string = `div.plist a.f`               // 取href
	playdom   string = `section.h-play audio#player` // 取src
	coverdom  string = `div.bimg img`                // 取src
	useragent string = `Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Mobile Safari/537.36 Edg/114.0.1823.79`
	title     string = `div.binfo h1`               // 书名
	tag       string = `div.binfo p:nth-child(2)`   // 类型
	author    string = `div.binfo p:nth-child(3)`   // 作者
	voice     string = `div.binfo p:nth-child(4) a` // 播音
	update    string = `div.binfo p:nth-child(5)`   // 时间
	status    string = `div.binfo p:nth-child(6)`   // 状态
	intro     string = `div.intro p`                // 简介
)

// 初始化变量 (传入参数)
var (
	bookid      string //`12526`
	mainpage    string // 主页，同Referer
	showversion bool
	runpath     string
)

// 读取参数
func init() {
	flag.StringVar(&bookid, `id`, ``, `bookid`)
	flag.BoolVar(&showversion, `v`, false, `showversion`)
	flag.Parse()
	mainpage = rooturl + bookid
	runpath, _ = os.Getwd()
}

// 主程序
func main() {
	if showversion {
		fmt.Println(version)
		os.Exit(0)
	}
	if bookid == `` {
		fmt.Println(`请携带id参数运行！`)
		os.Exit(0)
	}
	osc := make(chan os.Signal, 1)
	go explore(osc)
	signal.Notify(osc, syscall.SIGTERM, syscall.SIGINT)
	fmt.Println("程序退出", <-osc)
}

// 网络请求
func require(url string) *http.Response {
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest(`GET`, url, nil)
	req.Header.Set(`User-Agent`, useragent)
	req.Header.Add(`Referer`, mainpage)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	//defer resp.Body.Close()
	return resp
}

// 分析内容
func explore(osc chan os.Signal) {
	// 将请求转换为选择器
	query := func(resp *http.Response) *goquery.Document {
		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()
		return doc
	}
	// 分析主页
	doc := query(require(mainpage))
	coverurl, _ := doc.Find(coverdom).Attr(`src`)
	reg1, _ := regexp.Compile(`!.*`)
	reg2, _ := regexp.Compile(`^`)
	coverurl = reg2.ReplaceAllString(reg1.ReplaceAllString(coverurl, ``), `https:`)
	listnum := doc.Find(listdom).Length()
	bookname := doc.Find(title).Text()
	line := `==============================`
	bookinfo := fmt.Sprintf("书名：%v\n%v\n%v\n播音：%v\n%v\n%v\n集数：%v\n%v\n简介：%v\n%v\n地址：%v\n封面：%v\n工具：%v\n",
		bookname,
		doc.Find(tag).Text(),
		doc.Find(author).Text(),
		doc.Find(voice).Text(),
		doc.Find(update).Text(),
		doc.Find(status).Text(),
		listnum, line, doc.Find(intro).Text(), line,
		mainpage, coverurl, `https://github.com/ZxwyWebSite/ting55down`,
	)
	fmt.Println(bookinfo)
	// 创建下载目录
	downpath := runpath + `/book/` + bookname
	_, e0 := os.Stat(downpath)
	if e0 != nil {
		e1 := os.MkdirAll(downpath, os.ModePerm)
		if e1 != nil {
			log.Fatal(e1)
		}
		fmt.Println(`创建下载目录 ` + downpath)
	}
	p1, _ := os.ReadDir(downpath)
	filelist := make(map[string]any)
	for _, f1 := range p1 {
		filelist[f1.Name()] = struct{}{}
	}
	// 下载文件
	savefile := func(data []byte, name string) {
		err := os.WriteFile(downpath+`/`+name, data, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
	}
	savefile([]byte(bookinfo), `bookinfo.txt`)
	download := func(url, name string) {
		if _, y := filelist[name]; !y {
			resp := require(url)
			defer resp.Body.Close()
			data, err := io.ReadAll(resp.Body)
			if err != nil {
				//log.Fatal(err)
				log.Println(err)
			}
			//os.WriteFile(downpath+`/`+name, data, os.ModePerm)
			savefile(data, name)
		} else {
			log.Println(`跳过文件 ` + name)
		}
	}
	download(coverurl, `Cover.jpg`)
	isexist := func(num string) bool {
		if _, y := filelist[num+`.m4a`]; !y {
			return true
		} else {
			return false
		}
	}
	// 分析播放页
	type glink struct {
		Ourl   string `json:"ourl"`
		Status int    `json:"status"`
		Title  string `json:"title"`
		Url    string `json:"url"`
	}
	postlink := func(page string) (string, int) {
		pageurl := mainpage + fmt.Sprintf("-%v", page)
		doc := query(require(pageurl))
		xt, _ := doc.Find(`meta[name='_c']`).Attr(`content`)
		client := &http.Client{Timeout: 10 * time.Second}
		req, _ := http.NewRequest(`POST`, `https://m.ting55.com/glink`, strings.NewReader(fmt.Sprintf("bookId=%v&isPay=0&page=%v", bookid, page)))
		req.Header.Set(`User-Agent`, useragent)
		req.Header.Add(`Referer`, pageurl)
		req.Header.Add(`xt`, xt)
		req.Header.Add(`Content-Type`, `application/x-www-form-urlencoded; charset=UTF-8`)
		resp, err := client.Do(req)
		if err != nil {
			log.Println(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var conf glink
		json.Unmarshal(body, &conf)
		return conf.Ourl, conf.Status
	}
	for i := 0; i < listnum; i++ {
		num := fmt.Sprint(i + 1)
		fmt.Printf("开始下载第 %v 章\n", num)
		var audiourl string
		var trynum int
		if isexist(num) {
			for {
				time.Sleep(time.Second * 10)
				trynum++
				var status int
				audiourl, status = postlink(num)
				if audiourl != `` {
					break
				} else {
					fmt.Printf("第 %v 次获取失败\n", trynum)
					if status == -2 {
						fmt.Println(`访问过快！等待10分钟后继续运行`)
						time.Sleep(time.Minute * 10)
					}
				}
			}
			fmt.Println(`成功获取音频URL ` + audiourl)
			download(audiourl, fmt.Sprintf("%v.m4a", num))
		} else {
			fmt.Printf("跳过 %v.m4a\n", num)
		}
	}
	// 退出程序
	fmt.Println(`恭喜，全部章节下载完成~`)
	osc <- syscall.SIGTERM
}