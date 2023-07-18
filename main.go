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
const version string = `v1.4`

// 初始化常量 (抓取参数)
const (
	rooturl   string = `https://m.ting55.com/book/`  // PC版播放器有样式不好搞，故从手机版页面抓取
	listdom   string = `div.plist a.f`               // 取href 注：'.f' 为免费章节
	playdom   string = `section.h-play audio#player` // 取src
	coverdom  string = `div.bimg img`                // 取src
	useragent string = `Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Mobile Safari/537.36 Edg/114.0.1823.79`
	title     string = `div.binfo h1`             // 书名
	tag       string = `div.binfo p:nth-child(2)` // 类型
	author    string = `div.binfo p:nth-child(3)` // 作者
	voice     string = `div.binfo p:nth-child(4)` // 播音
	update    string = `div.binfo p:nth-child(5)` // 时间
	status    string = `div.binfo p:nth-child(6)` // 状态
	intro     string = `div.intro p`              // 简介
)

// 初始化变量 (传入参数)
var (
	bookid      string //`12526`
	mainpage    string // 主页，同Referer
	showversion bool
	downrpath   string
	//runpath     string
)

// 读取参数
func init() {
	runpath, _ := os.Getwd()
	flag.StringVar(&bookid, `id`, ``, `要下载的小说id`)
	flag.BoolVar(&showversion, `v`, false, `查看程序版本号`)
	flag.StringVar(&downrpath, `dp`, runpath+`/book`, `指定小说下载目录`)
	flag.Parse()
	mainpage = rooturl + bookid
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
	var reqnum int
	for {
		reqnum++
		resp, err := client.Do(req)
		if err != nil {
			//log.Fatal(err)
			fmt.Println(err)
			fmt.Printf("第 %v 次请求失败\n", reqnum)
			if reqnum > 4 {
				fmt.Println(`失败次数过多，请检查网络问题`)
				os.Exit(0)
			}
			time.Sleep(time.Second * 3)
		} else {
			//defer resp.Body.Close()
			return resp
		}
	}
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
	var bookintro string
	doc.Find(intro).Each(func(i int, s *goquery.Selection) {
		if i > 0 {
			bookintro += fmt.Sprintf("\n%v", s.Text())
		} else {
			bookintro += s.Text()
		}
	})
	bookinfo := fmt.Sprintf("书名：%v\n%v\n%v\n%v\n%v\n%v\n集数：%v\n%v\n简介：%v\n%v\n地址：%v\n封面：%v\n工具：%v\n",
		bookname,
		doc.Find(tag).Text(),
		doc.Find(author).Text(),
		doc.Find(voice).Text(),
		doc.Find(update).Text(),
		doc.Find(status).Text(),
		listnum, line, bookintro, line,
		mainpage, coverurl, `https://github.com/ZxwyWebSite/ting55down`,
	)
	fmt.Println(bookinfo)
	// 创建下载目录
	downpath := downrpath + `/` + bookname
	//downpath := runpath + `/book/` + bookname
	_, e0 := os.Stat(downpath)
	if e0 != nil {
		e1 := os.MkdirAll(downpath, os.ModePerm)
		if e1 != nil {
			log.Fatal(e1)
		}
		//fmt.Println(`创建下载目录 ` + downpath)
		fmt.Printf("创建下载目录 '%v'\n", downpath)
	}
	p1, _ := os.ReadDir(downpath)
	filelist := make(map[string]any)
	for _, f1 := range p1 {
		filelist[f1.Name()] = struct{}{}
	}
	// 预计下载时间
	const da, db, dh, dm int = 50, 10, 60, 3780
	dc, dd := listnum/da, listnum%da
	de, df := dc*(dm+da*db), dd*db
	if dd == 0 && dc > 0 {
		de -= dm
	}
	dg := de + df
	var usetime string
	if dg > dh {
		di, dj := dg/dh, dg%dh
		if di > dh {
			dk, dl := di/dh, di%dh
			usetime = fmt.Sprintf("%v 时 %v 分", dk, dl)
		} else {
			usetime = fmt.Sprintf("%v 分 %v 秒", di, dj)
		}
		usetime += fmt.Sprintf(" (%v 秒)", dg)
	} else {
		usetime = fmt.Sprintf("%v 秒", dg)
	}
	fmt.Printf("预计下载时间 %v\n\n", usetime)
	// 下载文件
	savefile := func(data []byte, name string) {
		err := os.WriteFile(downpath+`/`+name, data, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
	}
	d1 := `bookinfo.txt`
	if _, y1 := filelist[d1]; !y1 {
		savefile([]byte(bookinfo), d1)
	}
	download := func(url, name string) bool {
		resp := require(url)
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			//log.Fatal(err)
			fmt.Println(err)
			return false
		}
		//os.WriteFile(downpath+`/`+name, data, os.ModePerm)
		savefile(data, name)
		return true
	}
	d2 := `Cover.jpg`
	if _, y2 := filelist[d2]; !y2 {
		download(coverurl, d2)
	}
	isexist := func(num string) bool {
		file := num + `.mp3`
		if _, y := filelist[file]; !y {
			return true
		} else {
			fmt.Println(`跳过文件 ` + file)
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
	postlink := func(page string) (string, int, string) {
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
			fmt.Println(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var conf glink
		json.Unmarshal(body, &conf)
		return conf.Ourl, conf.Status, conf.Url
	}
	startime := time.Now()
	for i := 0; i < listnum; i++ {
		num := fmt.Sprint(i + 1)
		log.Printf("开始下载第 %v 章\n", num)
		var audiourl string
		//var audioext string
		var trynum, downum int
		if isexist(num) {
			for {
				time.Sleep(time.Second * 10)
				trynum++
				var status int
				var url2 string
				audiourl, status, url2 = postlink(num)
				if audiourl != `` {
					//audioext = `.m4a`
					break
				} else if url2 != `` {
					//audioext = `.mp3`
					audiourl = url2
					break
				} else {
					fmt.Printf("第 %v 次获取失败\n", trynum)
					if status == -2 {
						fmt.Println(`访问过快！等待1小时后再试`)
						time.Sleep(time.Minute * 63)
					}
					// if status == -1 {
					// 	fmt.Println(`暂不支持下载付费章节`)
					// 	osc <- syscall.SIGTERM
					// }
				}
			}
			fmt.Println(`成功获取音频URL ` + audiourl)
			fname := fmt.Sprintf("%v.mp3", num)
			//fname := num + audioext
			for {
				downum++
				if download(audiourl, fname) {
					break
				} else {
					fmt.Printf("第 %v 次下载失败\n", downum)
					if downum > 4 {
						fmt.Println(`失败次数过多，跳过章节，可在下载完成后再次运行重试`)
						//os.Remove(downpath + `/` + fname)
						break
					}
					time.Sleep(time.Second * 3)
				}
			}
		}
	}
	// 退出程序
	//fmt.Println(`恭喜，全部章节下载完成~`)
	fmt.Printf("恭喜！全部章节下载完成~ 耗时%v\n", time.Since(startime))
	osc <- syscall.SIGTERM
}
