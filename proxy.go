package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
)

/*
ProxyConfig configuration for proxy..
*/
type ProxyConfig struct {
	DestAddr   string
	ListenPort int
	TLSConfig  *tls.Config
}

type proxyDirection string

const (
	toOrigin proxyDirection = ">"
	toClient                = "<"
)

/*
Proxies a single pipe direction
- client, server
- dir is the direction, for logging
- id uniquely identifies the connection in the log
*/
func proxyPipe(client, server net.Conn, dir proxyDirection, id string) {
	var rdsock, wrsock net.Conn
	var idstr string

	if dir == toOrigin {
		rdsock = client
		wrsock = server
		idstr = fmt.Sprintf("%s[%s]", id, ">")
	} else {
		rdsock = server
		wrsock = client
		idstr = fmt.Sprintf("%s[%s]", id, "<")
	}

	wr := bufio.NewWriter(wrsock)
	rd := bufio.NewReader(rdsock)

	for {
		resp, err := rd.ReadString('\n')
		resp = strings.TrimSpace(resp)
		if resp != "" {
			fmt.Printf("%s: %s\r\n", idstr, resp)
			_, wrerr := wr.WriteString(resp + "\r\n")
			wr.Flush()
			if wrerr != nil {
				fmt.Printf("%s: Failure writing (%v)\n", idstr, wrerr)
				break
			}
		}
		if err != nil {
			fmt.Printf("%s: Failure reading (%v)\n", idstr, err)
			break
		}
	}

	wrsock.Close()
	rdsock.Close()

}

func handleProxyConn(conn net.Conn, config ProxyConfig, idnum int) {
	// Set up outbound connect, and readers/writers
	var wg sync.WaitGroup
	defer conn.Close()
	originConn, err := tls.Dial("tcp", config.DestAddr, config.TLSConfig)
	if err != nil {
		fmt.Printf("Could not connect outbound connection to %s: %v\n", config.DestAddr, err)
		return
	}

	defer originConn.Close()

	// Set up two go threads, one for reading, one for writing
	wg.Add(2)
	fmt.Printf("Connection ID %d = %s\n", idnum, conn.RemoteAddr().String())
	cid := strconv.Itoa(idnum)

	go func() {
		defer wg.Done()
		proxyPipe(conn, originConn, toOrigin, cid)
	}()
	go func() {
		defer wg.Done()
		proxyPipe(conn, originConn, toClient, cid)
	}()
	wg.Wait()
}

// RunProxy runs the proxy
func RunProxy(config ProxyConfig) {
	fmt.Printf("Proxy listening on port %d\n", config.ListenPort)
	lsn, err := tls.Listen("tcp", "0.0.0.0:"+strconv.Itoa(config.ListenPort), config.TLSConfig)
	if err != nil {
		panic(err)
	}
	idserial := 0
	for {
		conn, err := lsn.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: " + err.Error())
			continue
		}
		idserial++
		go handleProxyConn(conn, config, idserial)
	}
}
