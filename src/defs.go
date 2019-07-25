package main

const (
	rcOK = iota
	rcConfigLoadErr
	rcLogInitErr
	rcDetectErr
	rcConnectErr
	rcPatientErr
	rcConnectDbErr
)

const (
	rpcOK         = iota
	rpcUnkonwn    //未知错误
	rpcInvalidCgi //无效CGI
	rpcCgiTimeout //CGI无响应
	rpcLuanchFail //CGI启动失败
)

var rpcCodesLabel = map[int]string{
	rpcOK:         "操作成功",
	rpcUnkonwn:    "未知错误",
	rpcInvalidCgi: "无效CGI",
	rpcCgiTimeout: "CGI无响应",
	rpcLuanchFail: "CGI启动失败",
}

type Result struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data string `json:"data"`
}

var supporttedFileType = map[string]bool{
	".exe": true,
	//".jar":  true,
	//".jarw": true,
}

type CGI struct {
	Cgi string `json:"cgi"` //cgi程序的名称，附带后缀
	Ver string `json:"ver"` //cgi程序的版本号
	Url string `json:"url"` //下载链接
}

const indexPage = `
<!DOCTYPE html>
<html>
<head><title>broswer.extension.hub</title></head>
<body style="text-align: center;">
<h1>Welcome to broswer.extension.hub!</h1>
<h2>CGI directory:</h2><p>{{.CgiDir}}</p>
<table  border="1" align="center"><tr><th>CGI</th><th>version</th></tr>{{.CgiList}}</table>
</body>
</html>
`
