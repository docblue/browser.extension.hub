package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/cihub/seelog"
	"github.com/fsnotify/fsnotify"
	"github.com/julienschmidt/httprouter"
)

var (
	// FROM config file
	listenningPort = 15101
	cgiUpdateUrl   = ""

	// CLI运行参数
	ver    = flag.Bool("version", false, "prints current version")
	curVer = "v0.0.181129"

	//其他变量
	cgiPath            = filepath.Join(filepath.Dir(os.Args[0]), "cgi")
	cgiInPath          = filepath.Join(filepath.Dir(os.Args[0]), "cgi.in")
	cgiOutPath         = filepath.Join(filepath.Dir(os.Args[0]), "cgi.out")
	fileWatcherTimeout = 6000 //ms
	cgiDeployPeriod    = 6    //minute
)

func main() {
	flag.Parse()
	if *ver {
		fmt.Fprintf(os.Stdout, curVer)
		os.Exit(rcOK)
	}

	os.Mkdir(cgiPath, os.ModeDir)
	os.Mkdir(cgiInPath, os.ModeDir)
	os.Mkdir(cgiOutPath, os.ModeDir)

	//
	// 解析配置文件
	//

	if err := parseConfigurationFile(os.Args[0] + ".ini"); err == nil {
		fmt.Fprintf(os.Stdout, "version:%s\n\n", curVer)
	} else {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(rcConfigLoadErr)
	}

	//
	//初始化seelog
	//

	defer seelog.Flush()
	logger, err := seelog.LoggerFromConfigAsString(makeSeelogConfig())
	if err != nil {
		fmt.Fprintf(os.Stderr, "log initialize fail\n")
		os.Exit(rcLogInitErr)
	}
	seelog.ReplaceLogger(logger)

	//
	// 启动cgi.out监听
	//

	go CgiOutWatcher()

	//
	// 定时更新CGI
	//

	go cgiUpdater()

	//
	// http服务
	//

	router := httprouter.New()
	router.GET("/", index)
	router.GET("/echo", echo)  //可用于检测HUB状态
	router.POST("/echo", echo) //可用于检测HUB状态
	router.POST("/hthub/:cgi", hthub)
	router.GET("/hthub/:cgi", hthub)
	seelog.Tracef("listening on:%d", listenningPort)
	seelog.Criticalf("%s", http.ListenAndServe(fmt.Sprintf(":%d", listenningPort), router))
}

func CgiOutWatcher() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		seelog.Criticalf("fsnotify.NewWatcher() fail:%s", err)
		return
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					seelog.Criticalf("<-watcher.Events fail:%s", err)
					return
				}
				if event.Op&fsnotify.Create == fsnotify.Create {
					seelog.Tracef("watched new file created:%s", event.Name)
					if sigInterface, ok := createSignals.Load(filepath.Base(event.Name)); ok {
						sig, _ := sigInterface.(chan int)
						sig <- 1
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					seelog.Errorf("<-watcher.Errors:%s", err)
				}
			}
		}
	}()

	err = watcher.Add(cgiOutPath)
	if err != nil {
		seelog.Criticalf("watcher.Add() fail:%s", err)
	}
	<-done
}

func cgiUpdater() {
	for {
		cgisUrl, err := url.ParseRequestURI(cgiUpdateUrl)
		if err != nil {
			seelog.Errorf("cgiUpdateUrl is INVALID:%s", cgiUpdateUrl)
			return
		}

		resp, err := http.Get(cgiUpdateUrl)
		if err != nil {
			seelog.Debugf("http.Get(%s) error:%s", cgiUpdateUrl, err)
			time.Sleep(time.Minute * time.Duration(cgiDeployPeriod))
			continue
		}

		cgis := []CGI{}
		agiArray, _ := ioutil.ReadAll(resp.Body)
		seelog.Debugf("cgis server response:%s", agiArray)

		err = json.Unmarshal(agiArray, &cgis)
		if err != nil {
			seelog.Debugf("json.Unmarshal error:%s", err)
			time.Sleep(time.Minute * time.Duration(cgiDeployPeriod))
			continue
		}

		for _, c := range cgis {
			if c.Ver > getCgiVer(filepath.Join(cgiPath, c.Cgi)) {
				res, err := http.Get(fmt.Sprintf("%s://%s/%s", cgisUrl.Scheme, cgisUrl.Host, c.Url))
				if err != nil {
					seelog.Errorf("http.Get() fail:%s", err)
				} else {
					defer res.Body.Close()
					data, _ := ioutil.ReadAll(res.Body)
					ioutil.WriteFile(filepath.Join(cgiPath, c.Cgi), data, os.ModePerm) //如果程序已经启动就等下次
				}
			}
		}

		time.Sleep(time.Minute * time.Duration(cgiDeployPeriod))
		//time.Sleep(time.Second * 10)
	}
}

func makeSeelogConfig() string {

	headerCfg := `
<seelog>
    <outputs formatid="ops">
		<filter levels="debug,trace,info,warn,error,critical">
			 <console/>
		</filter>        
           `

	opsCfg := fmt.Sprintf(`<filter levels="debug,trace,info,warn,error,critical"><rollingfile type="size" namemode="prefix" filename="%s/%s.log" maxsize="1024000" maxrolls="10" /> </filter>`,
		filepath.Join(filepath.Dir(os.Args[0]), "logs"), filepath.Base(os.Args[0]))

	footerCfg := `
    </outputs>
    <formats>
        <format id="ops" format="%Date(2006-01-02 15:04:05.000) - %Msg%r%n"/>
    </formats>
</seelog>
`

	return headerCfg + opsCfg + footerCfg
}

func getCgiVer(cgi string) string {
	cmd := exec.Command(cgi, "-version")
	if stdoutStderr, err := cmd.CombinedOutput(); err != nil {
		seelog.Errorf("cmd.CombinedOutput fail:%s", err)
	} else {
		return string(stdoutStderr)
	}
	return ""
}
