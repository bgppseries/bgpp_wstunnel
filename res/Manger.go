package res

import (
	"bgpp_wstunnel/log"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"math/rand"
	"sync"
	"time"
)

type Htconn struct {
	connid        string //负责唯一标志连接
	serverid      int    //服务ID，供用户选择
	Conn          *websocket.Conn
	HeartbeatTime int64  //上一次心跳时间
	using         bool   //记录是否正在使用
	Url           string //URL,用于HTTP隧道服务器转发HTTP请求
	alive         bool
	IsHTTP        bool
}

type connManager struct {
	Clients  map[string]*Htconn // 保存连接
	Accounts map[int][]string   // 账号和连接关系,map的key是账号id即：AccountId，这里主要考虑到一个账号多个连接
	mu       *sync.Mutex
}

// 定义一个管理Manager
var Manager = connManager{
	Clients:  make(map[string]*Htconn), // 参与连接的用户，出于性能的考虑，需要设置最大连接数
	Accounts: make(map[int][]string),   // 账号和连接关系
	mu:       new(sync.Mutex),
}
var (
	RegisterChan   = make(chan *Htconn, 100) // 注册链
	unregisterChan = make(chan *Htconn, 100) // 注销链
)

type Beatbag struct { //心跳包格式
	Pingmessage string `json:"pingmessage"`
	Connid      string `json:"connid"`
}
type Beatres struct {
	Pongmessage string `json:"pongmessage"`
	Connid      string `json:"connid"`
}
type Verify struct { //认证信息格式
	Token    string `json:"token"`
	Server   string `json:"server"`
	Serverid int    `json:"serverid"`
	Ishttp   bool   `json:"ishttp"`
	Url      string `json:"url"`
}

const (
	HeartbeatCheckTime       = 9 * time.Second // 心跳检测几秒检测一次
	HeartbeatTime      int64 = 12              // 心跳距离上一次的最大时间
)

//注册注销
func Saveconn() {
	for {

		select {
		case conn := <-RegisterChan:
			Serverbind(conn)
		case conn := <-unregisterChan:
			_ = conn.Conn.Close()
			delconn(conn)
		}
	}
}

//连接绑定
func Serverbind(c *Htconn) {
	Manager.mu.Lock()
	defer Manager.mu.Unlock()
	Manager.Clients[c.connid] = c //将该连接加入到map中
	if _, ok := Manager.Accounts[c.serverid]; ok {
		Manager.Accounts[c.serverid] = append(Manager.Accounts[c.serverid], c.connid)
	} else {
		Manager.Accounts[c.serverid] = []string{c.connid}
	}
}

func delconn(c *Htconn) {
	Manager.mu.Lock()
	defer Manager.mu.Unlock()
	delete(Manager.Clients, c.connid)
	for k, v := range Manager.Accounts[c.serverid] {
		if v == c.connid {
			// 删除服务端中维护的连接ID
			Manager.Accounts[c.serverid] = append(Manager.Accounts[c.serverid][:k], Manager.Accounts[c.serverid][k+1:]...)
		}
	}
	if len(Manager.Accounts[c.serverid]) == 0 {
		// 如果可连接的客户端为空，则删除服务
		fmt.Println(c.serverid)
		delete(Manager.Accounts, c.serverid)
	}
}

////中断连接，并解绑
//func delconn(c *Htconn) {
//	Manager.mu.Lock()
//	defer Manager.mu.Unlock()
//	delete(Manager.Clients, c.connid)
//	if len(Manager.Accounts[c.serverid]) > 0 {
//		for k, clientid := range Manager.Accounts[c.serverid] {
//			if clientid == c.connid {
//				Manager.Accounts[c.serverid] = append(Manager.Accounts[c.serverid][:k], Manager.Accounts[c.serverid][k+1:]...)
//			}
//		}
//	}
//}

//获取连接 根据connid获取Htconn
func Getconn(connid string) (*Htconn, bool) {
	Manager.mu.Lock()
	defer Manager.mu.Unlock()
	conn, ok := Manager.Clients[connid]
	return conn, ok
}

//根据服务serverid 获取 连接connid
func GetClient(accountId int) *Htconn {
	clients := make([]*Htconn, 0)
	Manager.mu.Lock()
	defer Manager.mu.Unlock()

	if len(Manager.Accounts[accountId]) > 0 {
		for _, clientId := range Manager.Accounts[accountId] {
			if c, ok := Manager.Clients[clientId]; ok {
				clients = append(clients, c)
			}
		}

		m := rand.Intn(100) //随机法负载均衡
		log.Info.Println(clients[m%len(clients)].connid, "选择了节点", m)
		return clients[m%len(clients)]
	}
	return nil
}

func (manager *connManager) PrintConn() {
	manager.mu.Lock()
	for k, _ := range manager.Accounts {
		fmt.Println("服务", k, "连接的客户端有：")
		for _, vv := range manager.Clients {
			if vv.serverid == k {
				fmt.Println("客户端id:", vv.connid, "上次心跳时间:", time.Unix(vv.HeartbeatTime, 0).Format("2006-01-02 15:04:05"), "服务：", vv.Url)
			}
		}
	}
	manager.mu.Unlock()
}
func Del() {
	Manager.mu.Lock()
	for _, c := range Manager.Clients {
		if err := c.Conn.WriteMessage(websocket.TextMessage, []byte("ya")); err != nil {
			fmt.Println("test")
			delconn(c)
		}
	}
	Manager.mu.Unlock()
}

//心跳检测
func Heartbeat() {
	for {
		// 获取所有的连接Htconn
		Manager.mu.Lock()
		clients := make([]*Htconn, len(Manager.Clients))
		for _, c := range Manager.Clients {
			b := Beatbag{
				Pingmessage: "hello",
				Connid:      c.connid,
			}
			m, err := json.Marshal(b)
			if err != nil {
				log.Info.Println("heartbeat 心跳包序列化失败")
			}
			if time.Now().Unix()-c.HeartbeatTime > HeartbeatTime {
				unregisterChan <- c
			} else {
				c.Conn.WriteMessage(websocket.TextMessage, m)
				log.Info.Println("心跳发送成功", string(m))
				clients = append(clients, c)
			}
		}
		Manager.mu.Unlock()
		time.Sleep(6 * time.Second)
	}
}
