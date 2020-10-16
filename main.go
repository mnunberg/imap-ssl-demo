package main

import (
	"crypto/tls"
	"flag"
	"sync"
)

func main() {
	var tlsCert string = "server.pem"
	var tlsKey string = "key.pem"
	tlsConfig := tls.Config{InsecureSkipVerify: true}
	sconfig := ServerConfig{
		TLSConfig: &tlsConfig,
	}
	pconfig := ProxyConfig{
		TLSConfig: &tlsConfig,
	}

	flag.IntVar(&sconfig.ListenPort, "s", 993, "Listening port for server, 0 for disable")
	flag.IntVar(&pconfig.ListenPort, "p", 8993, "Listening port for proxy, 0 for disable")
	flag.StringVar(&pconfig.DestAddr, "u", "127.0.0.1:9993", "Upstream port for proxy origin")
	flag.StringVar(&tlsCert, "c", "server.pem", "Path to SSL certificate")
	flag.StringVar(&tlsKey, "k", "key.pem", "Path to SSL private key")
	flag.Parse()

	cert, err := tls.LoadX509KeyPair(tlsCert, tlsKey)
	if err != nil {
		panic(err)
	}

	tlsConfig.Certificates = []tls.Certificate{cert}
	var wg sync.WaitGroup

	wg.Add(1)
	if sconfig.ListenPort != 0 {
		go RunServer(sconfig)
	}
	if pconfig.ListenPort != 0 {
		go RunProxy(pconfig)
	}
	wg.Wait()
}
