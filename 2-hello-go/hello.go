package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

var domain = flag.String("domain", "localhost", "Provide the domain name for the server")
var mtls = flag.Bool("mtls", false, "Enable Mutual authentication")
var httpPort = flag.String("port", "9443", "Set the HTTP(S) port")

func main() {
	flag.Parse()

	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(rw, "Hello World \n")
	})

	server := &http.Server{Addr: ":"+*httpPort}
	if *mtls {
		server = createServerWithMTLS()
	}

	// start the server loading the cert and key
	err := server.ListenAndServeTLS("3_application/certs/"+*domain+".cert.pem", "3_application/private/"+*domain+".key.pem")
	if err != nil {
		log.Fatal("Unable to start server: ", err)
	}
}

func createServerWithMTLS() *http.Server {
	// add the cert chain as the intermediate signs both the servers and the clients cert
	clientCACert, err := ioutil.ReadFile("2_intermediate/certs/ca-chain.cert.pem")
	if err != nil {
		log.Fatal(err)
	}

	clientCertPool := x509.NewCertPool()
	clientCertPool.AppendCertsFromPEM(clientCACert)

	tlsConfig := &tls.Config{
		ClientAuth:               tls.RequireAndVerifyClientCert,
		ClientCAs:                clientCertPool,
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS12,
	}

	tlsConfig.BuildNameToCertificate()

	return &http.Server{
		Addr:      ":"+*httpPort,
		TLSConfig: tlsConfig,
	}
}
