package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/RIscRIpt/rrl/rhs/courier"
)

const (
	listenProtocol string = "tcp"
)

var (
	flagListenAddress = flag.String("addr", ":40545", "listen address")
)

func fatalError(when string, what error) {
	log.Fatalf(
		"Fatal error occured on `%s`\nError: %v\n",
		when,
		what,
	)
}

func main() {
	flag.Parse()

	listener, err := net.Listen(listenProtocol, *flagListenAddress)
	if err != nil {
		fatalError("net.Listen", err)
	}

	exitSignal := make(chan os.Signal)
	signal.Notify(exitSignal,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	go handleClients(listener.(*net.TCPListener))

	s := <-exitSignal
	log.Printf("received signal `%s`, exitting...\n", s.String())

	listener.Close()
}

func handleClients(listener *net.TCPListener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error occured on `listener.Accept`\nError: %v\n", err)
		}
		go handleClient(conn.(*net.TCPConn))
	}
}

func handleClient(conn *net.TCPConn) {
	//var msg [16]byte

	log.Printf("Client connected from %+v\n", conn.RemoteAddr())

	c := courier.New(conn)

loop:
	for {
		msg, err := c.Receive()
		switch err {
		case nil:
		case courier.ErrUnknownMessage:
			header := msg.(courier.Header)
			log.Println(err, header)
			break loop
		case io.EOF:
			break loop
		default:
			log.Println(err)
			break loop
		}

		switch m := msg.(type) {
		case courier.OK:
			fmt.Println("OK")
		case courier.Version:
			fmt.Printf("Version: %d\n", m.Value)
		case courier.Authorization:
			fmt.Printf("Token: %+v\n", m.Token)
		case courier.LinkLibrary:
			fmt.Printf("Library: %+v\n", m.Name())
			runSvcLinker(conn, m.Name())
			conn.Close()
			break loop
		}
	}

	log.Printf("Client disconnected from %+v\n", conn.RemoteAddr())

	conn.Close()
}

func runSvcLinker(conn *net.TCPConn, library string) {
	connFile, err := conn.File()
	if err != nil {
		panic(err)
	}
	procAttr := os.ProcAttr{
		Dir:   "/home/richard/mega/src/C++/rrl/svclinker/build/",
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr, connFile},
	}
	proc, err := os.StartProcess("/home/richard/mega/src/C++/rrl/svclinker/build/svclinker", []string{"svclinker", "3", library}, &procAttr)
	if err != nil {
		panic(err)
	}
	proc.Release()
}
