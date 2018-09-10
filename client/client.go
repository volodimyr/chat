package main

import (
	"github.com/volodimyr/chat/config"
	"io"
	"log"
	"net"
	"os"
)

func main() {
	conn, err := net.Dial(config.Network, config.Port)
	defer conn.Close()
	if err != nil {
		log.Fatalln(err)
	}

	done := make(chan struct{})
	go func() {
		_, err := io.Copy(os.Stdin, conn)
		if err != nil {
			log.Fatalln(err)
		}
		log.Println("It looks like server dropped connection")
		done <- struct{}{}
	}()
	mustCopy(conn, os.Stdin)
	<-done
}

func mustCopy(dst io.Writer, src io.Reader) {
	if _, err := io.Copy(dst, src); err != nil {
		log.Fatalln(err)
	}
}
