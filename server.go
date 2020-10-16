package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
)

/*
ServerConfig common config for full blown server
*/
type ServerConfig struct {
	ListenPort int
	TLSConfig  *tls.Config
}

type request struct {
	command string
	payload string
	id      string
}

type commandHandler func(connection, request) error

type server struct {
	config   *ServerConfig
	handlers map[string]commandHandler
}

type connection struct {
	parent *server
	reader *bufio.Reader
	writer *bufio.Writer
}

func checkError(err error) {
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		panic(err)
	}
}

func (s *server) installHandlers() {
	s.handlers["NOOP"] = func(c connection, req request) error {
		c.simpleResponse(req, "NOOP")
		return nil
	}
	s.handlers["CAPABILITY"] = func(c connection, req request) error {
		c.writer.WriteString("* CAPABILITY IMAP4rev1 SASL-IR LOGIN-REFERRALS ID ENABLE IDLE LITERAL+ AUTH=OAUTHBEARER AUTH=XOAUTH2\r\n")
		c.simpleResponse(req, "OK Completed")
		return nil
	}
	s.handlers["LOGIN"] = func(c connection, req request) error {
		c.simpleResponse(req, "NO [UNAVAILABLE] Temporary authentication failure")
		return nil
	}
}

func (c *connection) simpleResponse(r request, s string) {
	c.writer.WriteString(r.id + " " + s + "\r\n")
}

func (c *connection) run() {
	c.writer.WriteString("* OK IMAP4rev1 Server Ready\r\n")
	c.writer.Flush()
	for {
		// Read line from client
		line, err := c.reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading: %v\n", err)
			return
		}
		line = strings.TrimSpace(line)
		fmt.Printf("<%s\n", line)
		fields := strings.SplitN(line, " ", 3)
		if len(fields) < 2 {
			c.writer.WriteString("BAD COMMAND\r\n")
			c.writer.Flush()
			continue
		}
		req := request{id: fields[0], command: fields[1]}
		if len(fields) == 3 {
			req.payload = fields[2]
		}
		handler := c.parent.handlers[strings.ToUpper(req.command)]
		if handler == nil {
			fmt.Printf("Missing command handler for %s\n", req.command)
			resp := fmt.Sprintf("%s Not implemented [%s]\r\n", req.id, req.command)
			c.writer.WriteString(resp)
		} else {
			cb := commandHandler(handler)
			cb(*c, req)
		}
		c.writer.Flush()
	}

}

func (s *server) handleClient(conn net.Conn) {
	defer conn.Close()
	c := connection{reader: bufio.NewReader(conn), writer: bufio.NewWriter(conn), parent: s}
	c.run()
}

/*
RunServer runs the server service
*/
func RunServer(config ServerConfig) {
	fmt.Printf("Running IMAP server on %d\n", config.ListenPort)
	addr := fmt.Sprintf("0.0.0.0:%d", config.ListenPort)
	lsn, err := tls.Listen("tcp", addr, config.TLSConfig)

	s := server{config: &config}
	s.handlers = make(map[string]commandHandler)
	s.installHandlers()

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
			s.handleClient(conn)
		}()
	}
}
