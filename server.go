package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

func checkError(err error) {
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		panic(err)
	}
}

func handleClient(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	writer.WriteString("* OK IMAP4rev1 Server Ready\r\n")
	writer.Flush()

	// Wait for input
	resp, err := reader.ReadString('\n')
	checkError(err)
	log.Println(resp)
	// Expect CAPABILITY request..
	fields := strings.SplitN(resp, " ", 2)
	line := fields[1]
	line = strings.TrimSpace(line)
	line = strings.ToUpper(line)
	log.Printf("Parsed %s\n", line)
	if line != "CAPABILITY" {
		fmt.Println("Expected CAPABILITY, got something else")
		os.Exit(1)
	}
	writer.WriteString("* CAPABILITY IMAP4rev1 AUTH=OAUTHBEARER SASL-IR\r\n")
	writer.WriteString(fields[0] + " OK Completed\r\n")
	writer.Flush()

	// Read token from client..
	resp, err = reader.ReadString('\n')
	checkError(err)
	log.Println("Got second client response")
	log.Println(resp)
}

func main() {
	cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
	checkError(err)
	config := tls.Config{Certificates: []tls.Certificate{cert}, InsecureSkipVerify: true}
	lsn, err := tls.Listen("tcp", "0.0.0.0:993", &config)
	checkError(err)
	for {
		conn, err := lsn.Accept()
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		fmt.Println("Accepted new IMAP server connection..")

		go func() {
			defer func() {
				if err := recover(); err != nil {
					fmt.Println("Connection failed: ", err)
				}
			}()
			handleClient(conn)
		}()
	}
}
