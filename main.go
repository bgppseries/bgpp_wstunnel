package main

import (
	"bgpp_wstunnel/client"
	"bgpp_wstunnel/log"
	"bgpp_wstunnel/res"
	"bufio"
	"flag"
	"fmt"
	"os"
)

func main() {
	args := res.Args{}
	flag.IntVar(&args.Serverid, "serverid", 1, "port for server to listen tcp package")
	flag.IntVar(&args.Tcpport, "tcpport", 8001, "port for server to listen tcp package")
	flag.IntVar(&args.Httpport, "httpport", 8000, "port for server to listen http package")
	flag.BoolVar(&args.Isserver, "isserver", true, "server or client")
	flag.StringVar(&args.Remote, "remote", "127.0.0.1:8000", " server address i.e. 127.0.0.1:8000")
	flag.StringVar(&args.Upstream, "upstream", "http://127.0.0.1:3000", "upstream server i.e. http://127.0.0.1:3000")
	//flag.BoolVar(&args.isSSl, "isSSl", true, "use ssl or not")
	flag.StringVar(&args.Url, "url", "http://127.0.0.1:789", "wu")
	flag.StringVar(&args.Token, "token", "sjdk", "used to verfy")
	flag.StringVar(&args.Server, "server", "sjdk", "used to verfy")
	flag.BoolVar(&args.Ishttp, "ishttp", true, "http over websocket or tcp over websocket")

	flag.Parse()
	log.Info.Println("log file test")
	fmt.Println("hello welcome")
	go res.Saveconn()
	go res.Heartbeat()
	if args.Isserver {
		go res.Startres(args)
	} else {
		if args.Ishttp {
			go client.StartHTTPclient(args)
		} else {
			go client.StartTCPclient(args)
		}
	}
	for true {

		if args.Isserver {
			inputReader := bufio.NewReader(os.Stdin)
			input, err := inputReader.ReadString('\n')
			if err != nil {
				fmt.Printf("There were errors reading, exiting program.")
				return
			}
			switch input {
			case "select\r\n":
				fmt.Println("your input is select")
				println("请输入serverid")
				var n int
				_, err := fmt.Scanf("%d", &n)
				res.Sid = n
				log.Info.Println("sid has been changed,now sid is", res.Sid)
				if err != nil {
					log.Error.Printf("cuowu", err)
					return
				}
			case "print\r\n":
				fmt.Println("your input is printf")
				//res.Del()
				res.Manager.PrintConn()
			case "\n":

			default:
				fmt.Println("输入错误,请重新输入")
			}
		}
	}

}
