package client

import (
	"bgpp_wstunnel/log"
	"bgpp_wstunnel/res"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

type verfy struct {
	Token string `json:"token"`

	Server   string `json:"server"`
	Serverid int    `json:"serverid"`
	Ishttp   bool   `json:"ishttp"`
	Url      string `json:"url"`
}
type pingbag struct {
	Pingmessage string `json:"pingmessage"`
	Connid      string `json:"connid"`
}
type pongbag struct {
	Pongmessage string `json:"pongmessage"`
	Connid      string `json:"connid"`
}

func StartTCPclient(args res.Args) {
	//u := url.URL{Scheme: "ws", Host: args.Remote, Path: "/tcpws"}
	//Remote是服务器的地址：192.168.0.101:80
	//upstream是TCP服务器的地址
	//建立ws连接
	//u := url.URL{Scheme: "ws", Host: args.Remote, Path: "/tcpws"}
	for {
		c, _, err := websocket.DefaultDialer.Dial(args.Remote, nil)
		if err != nil {
			log.Info.Println("tunnel client connect to tunnel server failed", err)
			time.Sleep(2 * time.Second)
		} else {
			log.Info.Println("tunnell client connect success")

			client := verfy{
				Token:    "args.Token",
				Server:   "args.Server",
				Serverid: 1,
				Ishttp:   false,
				Url:      "TCP",
			}
			client.Token = args.Token
			client.Server = args.Server
			client.Serverid = args.Serverid
			d, err := json.Marshal(client)
			if err != nil {
				log.Info.Println("TCP client verfybag Marshal failed", err)
			}
			err = c.WriteMessage(websocket.TextMessage, d)
			if err != nil {
				log.Info.Println("verify write to tunnel server failed client", err)
			} else {
				log.Info.Println("verfy bag write to tunnel server success", client, string(d))
			}
			done := make(chan struct{})
			go func() {
				defer close(done)
				for {
					t, message, err := c.ReadMessage()
					if err != nil {
						log.Info.Println("ws tunnel read from tunnel server failed:", err)
						return
					}
					if t == websocket.TextMessage {
						ping := &res.Beatbag{}
						err = json.Unmarshal(message, &ping)
						if err != nil {
							log.Info.Println("tunnel client Unmarshal failed", err)
						} else {
							log.Info.Println("隧道客户端收到心跳包", string(message))
							pong := pongbag{
								Pongmessage: "world",
								Connid:      ping.Connid,
							}
							m, err := json.Marshal(pong)
							if err != nil {
								log.Info.Println("tunnel client Unmarshal failed", err)
							}
							c.WriteMessage(websocket.TextMessage, m)
						}
					} else {
						// TCP proxy To Upstream
						tcp1, err := net.Dial("tcp", args.Upstream)
						n1, err := tcp1.Write([]byte(string(message)))
						if err != nil {
							log.Info.Println("tcp tunnel ws to tcp tcp wriet failed:", err)
							return
						}
						log.Info.Println("ws--->tcp write", n1)
						buffer := make([]byte, 4096)
						n2, err := tcp1.Read(buffer)
						log.Info.Println("隧道客户端读取到TCP远端的回复：", buffer[:n2])
						log.Info.Println("ws--->tcp read", n2)
						log.Info.Println("tcp tunnel client the init ask", buffer[:n2])
						if err != nil {
							log.Info.Println("ws--->tcp read failed:", err)
							return
						} else {
							err := c.WriteMessage(websocket.BinaryMessage, []byte(string(buffer[:n2])))
							log.Info.Println("隧道客户端已将TCp回复转发给隧道服务器", string(buffer))
							if err != nil {
								return
							}
						}
					}
				}
			}()
			<-done
		}

	}
}

func StartHTTPclient(args res.Args) {
	//u := url.URL{Scheme: "ws", Host: args.Remote, Path: "/h2ws"}
	//建立ws连接
	//Remote是服务器的地址：192.168.0.101:80
	for true {

		c, _, err := websocket.DefaultDialer.Dial(args.Remote, nil)

		if c == nil {
			log.Info.Println("tunnel client connect to tunnel server failed", err)
			time.Sleep(2 * time.Second)
		} else {
			client := verfy{
				Token:    args.Token,
				Url:      args.Url,
				Server:   args.Server,
				Serverid: args.Serverid,
				Ishttp:   true,
			}
			d, err := json.Marshal(client)
			if err != nil {
				log.Info.Println("HTTP client verfybag Marshal failed", err)
			}
			err = c.WriteMessage(websocket.TextMessage, d)
			if err != nil {
				log.Info.Println("verify write to tunnel server failed client", err)
			}
			defer func(c *websocket.Conn) {
				err := c.Close()
				if err != nil {
					log.Info.Println("运行客户端websocket.conn.close出错:", err)
				}
			}(c)
			done := make(chan struct{})
			go func() {
				defer close(done)
				for {
					t, message, err := c.ReadMessage()
					if err != nil {
						return
					}
					if t == websocket.TextMessage {
						ping := res.Beatbag{}
						err = json.Unmarshal(message, &ping)
						if err != nil {
							log.Info.Println("tunnel client Unmarshal failed", err)
							return
						} else {
							log.Info.Println("隧道客户端收到心跳")
							pong := pongbag{
								Pongmessage: "world",
								Connid:      ping.Connid,
							}

							m, err := json.Marshal(pong)
							if err != nil {
								log.Info.Println("tunnel client Unmarshal failed", err)
							}
							c.WriteMessage(websocket.TextMessage, m)
						}
					} else if t == websocket.BinaryMessage {

						// proxyToUpstream
						log.Info.Printf("recv: %d", len(message))
						log.Info.Printf(string(message))
						buf := bytes.NewBuffer(message)
						bufReader := bufio.NewReader(buf)
						req, _ := http.ReadRequest(bufReader)
						body, _ := ioutil.ReadAll(req.Body) //复制req.Body--->body
						newReq, _ := http.NewRequest(req.Method, fmt.Sprintf("http://%s%s", req.Host, req.URL.String()), bytes.NewReader(body))
						////////////////代理    哇咔咔

						res.CopyHeaders(newReq.Header, &req.Header) //就实现这么个玩意 copy
						client := http.DefaultClient
						client.CheckRedirect = func(req *http.Request, via []*http.Request) error { //转发规则
							return http.ErrUseLastResponse
						}
						res, resErr := client.Do(newReq)

						if resErr != nil {
							log.Info.Println(resErr)
						} else {
							log.Info.Printf("Upstream tunnel res: %s\n", res.Status)

							buf2 := new(bytes.Buffer)

							err := res.Write(buf2)

							if err != nil {
								return
							}
							if res.Body != nil {
								defer res.Body.Close()
							}
							fmt.Println("Whole response", buf2.Len())

							c.WriteMessage(websocket.BinaryMessage, buf2.Bytes())
							fmt.Println("test")
						}
					}

				}
			}()

			<-done
		}

	}
}
