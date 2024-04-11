package res

import (
	"bgpp_wstunnel/log"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

var (
	Sid int = 1
)

type Args struct {
	Tcpport  int
	Httpport int
	Isserver bool
	isssl    bool
	Upstream string //隧道客户端向TCP服务器发起连接的地址
	Remote   string //隧道客户端向服务器发起ws连接所用URL
	Ishttp   bool
	Token    string
	Server   string
	Serverid int
	Url      string
}

//定义一个升级器，将普通的http连接升级为websocket连接
var upgrader = websocket.Upgrader{
	//定义读写缓冲区大小
	WriteBufferSize: 1024,
	ReadBufferSize:  1024,
	//校验请求
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func Startres(args Args) {
	httpmsg := make(chan *http.Response)
	httpoutgoing := make(chan *http.Request)
	msgfrom := make(chan []byte)
	msggoing := make(chan []byte)
	go tcplisten(args, msgfrom, msggoing)
	go Manger(httpoutgoing, msgfrom, Sid)
	http.HandleFunc("/tunnel", wsserver(httpmsg, msggoing, args))
	http.HandleFunc("/", proxyHandler(httpmsg, httpoutgoing)) //
	if err := http.ListenAndServe(fmt.Sprintf(":%d", args.Httpport), nil); err != nil {
		log.Error.Fatal(err) //打印错误日志并退出
	}
}
func tcplisten(args Args, msgfrom chan []byte, msggoing chan []byte) {
	args.Upstream = args.Upstream[7:]
	listener, err := net.Listen("tcp", args.Upstream)
	log.Info.Println("tcp tunnel server is listen", listener.Addr())
	if err != nil {
		log.Info.Println("tcp tunnel serve tcp listen failed:", err)
		return //线程的退出
	}
	defer listener.Close()
	//log.Printf("TCP tunnel serve is listening on %s", args.Upstream)
	for {
		conn, err := listener.Accept()
		log.Info.Println("accept !")
		if err != nil {
			log.Info.Println("tunnel serve accept tcp from remote failed:", err)
		} else {
			log.Info.Println("tcp tunnel serve receive remote tcp", conn.RemoteAddr())
			go process(conn, msgfrom, msggoing)

		}
	}
}
func process(conn net.Conn, msgfrom chan []byte, msggoing chan []byte) {
	done := make(chan struct{})
	go func() {
		//read
		for true {
			buf := make([]byte, 4096)
			log.Info.Println("tcp tunnel serve has accepted tcp", conn)
			n, err := conn.Read(buf)
			if err != nil {
				log.Info.Println("tcp tunnel serve read tcp from tcp client failed", err)
				return
			}
			msgfrom <- buf[:n]
			//send to channel
		}
	}()
	go func() {
		//wait
		for true {
			_, err := conn.Write([]byte(string(<-msggoing)))
			if err != nil {
				log.Info.Println("tcp tunnel serve write tcp response to tcp client failed", err)
				return
			}
			log.Info.Println("tcp tunnel serve receive tcp response successfully", conn)
		}
	}()
	<-done
}
func proxyHandler(msg chan *http.Response, outgoing chan *http.Request) func(w http.ResponseWriter, r *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		outgoing <- r
		log.Info.Println("res get http req and send to channel, waiting for response")
		res := <-msg
		log.Info.Println("res get response and writing response from tunnel", res.Status, "擦口红的夸奖哈克的")
		innerBody, _ := ioutil.ReadAll(res.Body)
		CopyHeaders(w.Header(), &res.Header)
		w.WriteHeader(res.StatusCode)
		w.Write(innerBody)

	}
}

func CopyHeaders(destination http.Header, source *http.Header) {
	for k, v := range *source {
		vClone := make([]string, len(v))
		copy(vClone, v)
		(destination)[k] = vClone
	}
} //复制拷贝，http头

func Manger(httpoutgoing chan *http.Request, tcpoutgoing chan []byte, sid int) {
	done := make(chan struct{})
	go func() { //发送HTTP
		defer close(done)
		for {
			var r *http.Request
			r = <-httpoutgoing
			if r.Body != nil {
				defer r.Body.Close()
			}
			body, _ := ioutil.ReadAll(r.Body)
			conn := GetClient(sid) //根据sid获得服务的url
			if conn != nil {
				req, _ := http.NewRequest(r.Method, fmt.Sprintf("%s%s", conn.Url, r.URL.Path),
					bytes.NewReader(body))
				CopyHeaders(req.Header, &r.Header)
				buf := new(bytes.Buffer)
				req.Write(buf)
				conn.Conn.WriteMessage(websocket.BinaryMessage, buf.Bytes())
				log.Info.Println("send tunnel client success HTTP", string(buf.Bytes()))
			}

		}
	}()
	go func() { //发送TCP
		defer close(done)
		for {
			conn := GetClient(sid)
			if conn != nil {
				conn.Conn.WriteMessage(websocket.BinaryMessage, []byte(string(<-tcpoutgoing)))
				log.Info.Println("send tunnel client success TCP")
			}
		}
	}()
	<-done
}

func wsserver(msg chan *http.Response, tcpmsggoing chan []byte, args Args) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		log.Info.Println("accept ws client")
		if err != nil {
			log.Info.Println("websocket连接，http升级出错:", err)
			return
		}
		defer conn.Close()
		done := make(chan struct{})
		go func() {
			defer close(done)
			for true {
				msgType, message, err := conn.ReadMessage()
				if err != nil {
					//log.Info.Println("tunnel server read message err:", err)
					return
				}
				if msgType == websocket.TextMessage {
					m := Verify{}
					p := Beatres{}
					err = json.Unmarshal(message, &m)
					err = json.Unmarshal(message, &p)
					log.Info.Println("反序列化", m, p)
					if p.Pongmessage == "world" {
						log.Info.Println("receive beat message")
						//收到心跳包回复
						t, _ := Getconn(p.Connid)
						t.HeartbeatTime = time.Now().Unix()
						log.Info.Println("时间更新", t.connid, time.Unix(t.HeartbeatTime, 0).Format("2006-01-02 15:04:05"))
						//更新心跳时间
					} else {
						var client = &Htconn{
							connid:        uuid.New().String(),
							serverid:      m.Serverid,
							Conn:          conn,
							HeartbeatTime: time.Now().Unix(),
							using:         false,
							Url:           m.Url,
							alive:         true,
							IsHTTP:        m.Ishttp,
						}
						if m.Token == args.Token {
							log.Info.Println("与隧道客户端认证成功", client.connid)
							RegisterChan <- client
							//加入注册队列
						} else {
							log.Error.Println("与客户端认证失败")
						}
					}

				} else if msgType == websocket.BinaryMessage {
					if GetClient(Sid).IsHTTP {
						reader := bytes.NewReader(message)
						scanner := bufio.NewReader(reader)
						res, _ := http.ReadResponse(scanner, nil)
						log.Info.Println("tunnel get HTTP res", res.Status)
						msg <- res
					} else {
						tcpmsggoing <- message
						log.Info.Println("tunnel get TCP res")
					}
				}
			}
		}()
		<-done

	}
}
