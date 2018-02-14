package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/rrlinker/go-librlcom"
)

const (
	listenProtocol string = "tcp"
)

var (
	flagListenAddress      = flag.String("addr", ":40545", "listen address")
	flagSymbolResolverPath = flag.String("res-addr", "/var/run/svcsymres.sock", "path to unix socket of resolver service (symbol to library)")
)

var (
	ErrSvcLinkerNotExited = errors.New("svclinker has not exited")
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

	c := librlcom.NewCourier(conn)

loop:
	for {
		msg, err := c.Receive()
		switch err {
		case nil:
		case librlcom.ErrUnknownMessage:
			header := msg.(*librlcom.Header)
			log.Println(err, header)
			break loop
		case io.EOF:
			break loop
		default:
			log.Println(err)
			break loop
		}

		switch m := msg.(type) {
		case *librlcom.OK:
			fmt.Println("OK")
		case *librlcom.Version:
			fmt.Printf("Version: %d\n", m.Value)
		case *librlcom.Authorization:
			fmt.Printf("Token: %+v\n", m.Token)
		case *librlcom.LinkLibrary:
			fmt.Printf("Library: %+v\n", m.String.String())
			err := runSvcLinker(conn, m.String.String())
			if err != nil {
				conn.Close()
				break loop
			}
		}
	}

	log.Printf("Client disconnected from %+v\n", conn.RemoteAddr())

	conn.Close()
}

func runSvcLinker(conn *net.TCPConn, library string) error {
	connFile, err := conn.File()
	if err != nil {
		return err
	}
	procAttr := os.ProcAttr{
		Dir:   "/home/richard/mega/src/C++/rrl/svclinker/build/",
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr, connFile},
	}
	proc, err := os.StartProcess(
		"/home/richard/mega/src/C++/rrl/svclinker/build/svclinker",
		[]string{
			"svclinker",
			"3",
			*flagSymbolResolverPath,
			library,
		},
		&procAttr,
	)
	if err != nil {
		return err
	}
	state, err := proc.Wait()
	if err != nil {
		return err
	}
	if !state.Exited() {
		return ErrSvcLinkerNotExited
	}
	return nil
}
