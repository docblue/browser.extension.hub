
This httpd,based cgi protocol,transport JSON-RPC message transparently between browser & non-browser app. It is a candidate in some scenarios.


本程序借鉴CGI协议，通过简单的交互协议，通过http/websocket，让浏览器和本地进程之间进行透传JSON-RPC协议，保证了易用性和扩展性。在一些业务场景中可以考虑这种方式。本程序仅供参考。

## Screenshot

![show](https://raw.githubusercontent.com/docblue/browser.extension.hub/master/show.gif)


## Todo

- add *_test.go   
 添加单元测试
- support more language of handler   
  支持更多语言，使CGI开发更方便  
- add IPC for high performence   
  添加IPC，使进程交互更高效


## Touch it!

A win32 setup program uploaded here. Download , install and take a test. 

已经在release目录上传了一个安装包，请下载安装试用

## Contact
docblue@163.com

## License

This project is under Apache v2 License. See the [LICENSE](LICENSE) file for the full license text.
