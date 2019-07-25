package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	texttemplate "text/template"
	"time"

	"github.com/cihub/seelog"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
	"github.com/satori/go.uuid"
)

var (
	upgrader      = websocket.Upgrader{} // use default options
	createSignals sync.Map
)

func init() {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
}

func jsonRequst(w http.ResponseWriter, r *http.Request) (wsConnect *websocket.Conn, jsonRequst string) {
	seelog.Debugf("request:METHOD=%s,PATH=%s", r.Method, r.URL.Path)

	if websocket.IsWebSocketUpgrade(r) {
		wsConnect, _ = upgrader.Upgrade(w, r, nil)
	}

	//
	// 提取JSON数据
	//

	jsonRequstSlice := []byte{}
	if wsConnect != nil {
		_, jsonRequstSlice, _ = wsConnect.ReadMessage()
	} else {
		jsonRequstSlice, _ = ioutil.ReadAll(r.Body)
	}

	jsonRequst = string(jsonRequstSlice)
	seelog.Debugf("request json:%s", jsonRequst)

	return wsConnect, jsonRequst
}

func jsonResponse(w http.ResponseWriter, wsConnect *websocket.Conn, jsonResponse string) {
	seelog.Debugf("response json:%s", jsonResponse)

	if wsConnect != nil {
		wsConnect.WriteMessage(websocket.TextMessage, []byte(jsonResponse))
		wsConnect.Close()
	} else {
		w.Header().Set("Content-Type", "application/json;charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, jsonResponse)
	}
}

func echo(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	wc, jsonIn := jsonRequst(w, r)

	//
	// 处理JSON数据
	//

	jsonOut := jsonIn

	jsonResponse(w, wc, jsonOut)
}

func waitCgiResponse(key string) bool {
	sig := make(chan int, 1)
	createSignals.Store(key, sig)

	hit := false
	timer200msCount := 0
	for {
		select {
		case <-sig:
			hit = true
			createSignals.Delete(key)
		default:
		}

		if hit {
			break
		} else if timer200msCount = timer200msCount + 1; timer200msCount < fileWatcherTimeout/200 {
			time.Sleep(time.Millisecond * 200)
		} else {
			break
		}
	}

	return hit
}

func hthub(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	wc, jsonIn := jsonRequst(w, r)
	jsonOut := ""

	// ------------------------------------------------------
	// 处理json-rpc请求
	//

	//获取cgi程序名称

	cgiName := ps.ByName("cgi")
	cgiAppFullName := ""
	seelog.Tracef("cgi name:%s", cgiName)

	// IF 无效名称

	cgiFiles, _ := ioutil.ReadDir(cgiPath)
	for _, file := range cgiFiles {
		if strings.HasPrefix(file.Name(), cgiName) && supporttedFileType[filepath.Ext(file.Name())] {
			cgiAppFullName = filepath.Join(cgiPath, file.Name())
			break
		}
	}

	if cgiAppFullName == "" {
		seelog.Errorf("NOT exsit:%s", cgiName)
		jsonBytes, _ := json.Marshal(Result{rpcInvalidCgi, rpcCodesLabel[rpcInvalidCgi], ""})
		jsonResponse(w, wc, string(jsonBytes))
		return
	}

	// 序列化请求数据到文本文件中

	inFile := filepath.Join(cgiInPath, fmt.Sprintf("%s-%s", cgiName, uuid.Must(uuid.NewV4())))
	if err := ioutil.WriteFile(inFile, []byte(jsonIn), os.ModePerm); err != nil {
		seelog.Errorf("WriteFile fail:%s", err)
		jsonBytes, _ := json.Marshal(Result{rpcUnkonwn, rpcCodesLabel[rpcUnkonwn], ""})
		jsonResponse(w, wc, string(jsonBytes))
		return
	}

	// 启动cgi程序，并监听rpc响应文件

	cmd := exec.Command(cgiAppFullName, "-request", inFile /*fmt.Sprintf("\"%s\"", inFile)*/)
	if err := cmd.Start(); err != nil {
		seelog.Errorf("cmd.Start(%s) fail:%s", cgiAppFullName, err)
		jsonBytes, _ := json.Marshal(Result{rpcLuanchFail, rpcCodesLabel[rpcLuanchFail], ""})
		jsonResponse(w, wc, string(jsonBytes))
		return
	}

	if waitCgiResponse(filepath.Base(inFile)) == false {
		seelog.Errorf("cgi response time out")
		jsonBytes, _ := json.Marshal(Result{rpcCgiTimeout, rpcCodesLabel[rpcCgiTimeout], ""})
		jsonResponse(w, wc, string(jsonBytes))
		return
	}

	// 读取RPC响应文件

	time.Sleep(time.Millisecond * 200) //确保文件已写入完毕
	rpcResponseFile := filepath.Join(cgiOutPath, filepath.Base(inFile))
	rpcBytes, err := ioutil.ReadFile(rpcResponseFile)
	if err != nil {
		seelog.Errorf("ioutil.ReadFile(%s) fail:%s", rpcResponseFile, err)
		jsonBytes, _ := json.Marshal(Result{rpcUnkonwn, rpcCodesLabel[rpcUnkonwn], ""})
		jsonResponse(w, wc, string(jsonBytes))
		return
	}
	jsonOut = string(rpcBytes)

	//删除RPC响应文件
	os.Remove(rpcResponseFile)

	//
	//--------------------------------------------------------

	jsonBytes, _ := json.Marshal(Result{rpcOK, rpcCodesLabel[rpcOK], jsonOut})
	jsonResponse(w, wc, string(jsonBytes))
}

func index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	cgiList := ""
	cgiFiles, _ := ioutil.ReadDir(cgiPath)
	for _, file := range cgiFiles {
		if _, ok := supporttedFileType[filepath.Ext(file.Name())]; ok {
			cgiList += fmt.Sprintf("<tr><td>%s</td><td>%s</td></tr>",
				file.Name(), getCgiVer(filepath.Join(cgiPath, file.Name())))
		}
	}

	buf := bytes.NewBufferString("")
	t, _ := texttemplate.New("indexPage").Parse(indexPage)
	t.Execute(buf, map[string]interface{}{
		"CgiDir":  cgiPath,
		"CgiList": cgiList,
	})
	fmt.Fprint(w, buf.String())
}
