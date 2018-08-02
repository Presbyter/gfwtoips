package main

import (
	"os"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"encoding/base64"
	"bufio"
	"io"
	"flag"
	"regexp"
	"fmt"
	"time"
)

const (
	COMMONREG = `^\!|\[|^@@|^\d+\.\d+\.\d+\.\d+`
	DOMAINREG = `([\w\-\_]+\.[\w\.\-\_]+)`
)

var (
	gfwTxtUrl    string
	redirectIp   string
	redirectPort int
)

func init() {
	flag.StringVar(&gfwTxtUrl, "baseurl", "https://raw.githubusercontent.com/gfwlist/gfwlist/master/gfwlist.txt", "数据来源地址(base64编码)")
	flag.StringVar(&redirectIp, "ip", "127.0.0.1", "转发IP地址")
	flag.IntVar(&redirectPort, "port", 5354, "转发端口")
}

func main() {
	flag.Parse()

	if _, err := os.Stat("./gfwlist.txt"); err == nil {
		log.Printf("remove gfwlist.txt \n")
		os.Remove("./gfwlist.txt")
	}
	gfwFile, err := os.OpenFile("./gfwlist.txt", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	defer gfwFile.Close()
	if err != nil {
		log.Printf("create gfwlist.txt error. err:%s\n", err.Error())
		panic(err)
	}

	res, err := http.Get(gfwTxtUrl)
	if err != nil {
		log.Printf("get gfwlist url failed. err:%s\n", err.Error())
		panic(err)
	}

	defer res.Body.Close()

	decoder := base64.NewDecoder(base64.StdEncoding, res.Body)

	reader := bufio.NewReader(decoder)
	domainMap := make(map[string]string)
	for {
		a, _, err := reader.ReadLine()
		if err != nil && err != io.EOF {
			panic(err)
		} else if err == io.EOF {
			break
		}
		gfwFile.WriteString(string(a) + "\n")
		if r, _ := regexp.Match(COMMONREG, a); r {
			log.Println("当前内容是注解.", string(a))
			continue
		}
		if r, _ := regexp.Match(DOMAINREG, a); r {
			reg, _ := regexp.Compile(DOMAINREG)
			a = reg.Find(a)
			log.Println("当前内容是域名.", string(a))
			if _, ok := domainMap[string(a)]; !ok {
				domainMap[string(a)] = string(a)
			}
		}

	}

	CreateGfwConfig(domainMap)

	log.Println("Current application path:", GetAppPath())
}

func GetAppPath() string {
	file, _ := exec.LookPath(os.Args[0])
	path, _ := filepath.Abs(file)
	index := strings.LastIndex(path, string(os.PathSeparator))

	return path[:index]
}

func CreateGfwConfig(m map[string]string) {
	if _, err := os.Stat("./gfwlist.conf"); err == nil {
		log.Printf("remove gfwlist.conf \n")
		os.Remove("./gfwlist.conf")
	}

	f, err := os.OpenFile("./gfwlist.conf", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	defer f.Close()
	if err != nil {
		log.Printf("create gfwlist.conf error. err:%s\n", err.Error())
		panic(err)
	}

	f.WriteString(fmt.Sprintf("# update on %v\n", time.Now().Format(time.RFC3339)))

	for _, v := range m {
		v = strings.Replace(v, "*", "", -1)
		v = strings.Replace(v, "/", "", -1)
		f.WriteString(fmt.Sprintf("server=/.%s/%s#%d\n", v, redirectIp, redirectPort))
		f.WriteString(fmt.Sprintf("ipset=/.%s/gfwlist\n", v))
	}
}
