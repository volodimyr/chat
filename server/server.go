package main

import (
	"bufio"
	"fmt"
	"github.com/volodimyr/chat/config"
	"io"
	"log"
	"net"
)

type user struct {
	Name    string
	Clients map[net.Conn]struct{}
}

type message struct {
	From     string
	FromAddr net.Addr
	Text     string
}

type client struct {
	Conn net.Conn
	Name string
}

type chat struct {
	Users       map[string]user
	NewConn     chan client
	DiscardConn chan client
	Input       chan message
}

func (cht *chat) start() {
	for {
		select {
		case conn := <-cht.NewConn:
			if u, ok := cht.Users[conn.Name]; ok {
				//add new connection for existing user
				u.Clients[conn.Conn] = struct{}{}
				//don't need to notify anybody as user has already opened at least one client
				break
			}
			//create new user and mark online
			cht.Users[conn.Name] = user{
				conn.Name,
				map[net.Conn]struct{}{conn.Conn: {}},
			}
			//notify
			go func() {
				cht.Input <- message{
					From: "server",
					Text: fmt.Sprintf("%s online", conn.Name),
				}
			}()
		case conn := <-cht.DiscardConn:
			if len(cht.Users[conn.Name].Clients) == 1 {
				//delete client, but keep user
				delete(cht.Users, conn.Name)
				//notify
				go func() {
					cht.Input <- message{
						From: "server",
						Text: fmt.Sprintf("%s offline", conn.Name),
					}
				}()
				break
			}
			//delete user if no clients alive
			delete(cht.Users[conn.Name].Clients, conn.Conn)
		case msg := <-cht.Input:
			cht.broadcastMessage(msg)
		}
	}
}

//broadcast message every client except itself
//if receiver user == sender user therefore we avoid "msg.Username" (user could have multiple clients)
func (cht *chat) broadcastMessage(msg message) {
	for _, user := range cht.Users {
		for conn := range user.Clients {
			if msg.FromAddr != conn.RemoteAddr() {
				if msg.From == user.Name {
					io.WriteString(conn, msg.Text+"\n")
					continue
				}
				io.WriteString(conn, msg.From+": "+msg.Text+"\n")
			}
		}
	}
}

func handleConn(cht *chat, conn net.Conn) {
	defer conn.Close()

	io.WriteString(conn, "Enter your name, please:")

	scanner := bufio.NewScanner(conn)
	scanner.Scan()

	user := user{Name: scanner.Text()}
	cht.NewConn <- client{conn, user.Name}

	defer func() {
		cht.DiscardConn <- client{conn, user.Name}
	}()

	//read from conn
	func() {
		for scanner.Scan() {
			ln := scanner.Text()
			cht.Input <- message{From: user.Name, FromAddr: conn.RemoteAddr(), Text: ln}
		}
	}()
}

func main() {
	server, err := net.Listen(config.Network, config.Port)
	if err != nil {
		log.Fatalln(err.Error())
	}
	defer server.Close()

	cht := &chat{
		Users:       make(map[string]user),
		NewConn:     make(chan client),
		DiscardConn: make(chan client),
		Input:       make(chan message),
	}

	go cht.start()

	for {
		conn, err := server.Accept()
		if err != nil {
			log.Fatalln(err.Error())
		}
		go handleConn(cht, conn)
	}
}
