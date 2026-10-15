package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

func main() {
	cer, err := generateKeyPair()
	if err != nil {
		log.Fatal(err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}

	config := &tls.Config{Certificates: []tls.Certificate{*cer}}
	listener, err := tls.Listen("tcp", ":8443", config)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		outRequest := make(map[string]any)

		headers := make(map[string]string)
		for k, v := range request.Header {
			headers[k] = strings.Join(v, ";")
		}

		outRequest["headers"] = headers
		outRequest["remoteAddr"] = request.RemoteAddr
		outRequest["method"] = request.Method
		outRequest["proto"] = request.Proto
		outRequest["host"] = request.Host

		out := make(map[string]any)
		out["request"] = outRequest
		out["hostname"] = hostname

		bytes, err := json.Marshal(out)
		if err != nil {
			writer.WriteHeader(500)
			return
		}

		writer.Header().Add("Content-Type", "application/json")
		_, _ = writer.Write(bytes)
	})

	go func() {
		server := http3.Server{
			Handler:    mux,
			Addr:       "0.0.0.0:8443",
			TLSConfig:  http3.ConfigureTLSConfig(config), // use your tls.Config here
			QUICConfig: &quic.Config{},
		}
		if err := server.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	go func() {

		if err := http.Serve(listener, mux); err != nil {
			log.Fatal(err)
		}
	}()

	if err = http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}

}

func generateKeyPair() (*tls.Certificate, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("cannot generate RSA key: %s", err)
	}

	serial, _ := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	tml := x509.Certificate{
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 1, 0),
		SerialNumber:          serial,
		Subject:               pkix.Name{},
		BasicConstraintsValid: true,
	}
	publicKeyBytes, err := x509.CreateCertificate(rand.Reader, &tml, &tml, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("certificate cannot be created: %s", err)
	}

	certificate, err := x509.ParseCertificate(publicKeyBytes)
	if err != nil {
		return nil, err
	}
	var cert tls.Certificate
	cert.Certificate = [][]byte{publicKeyBytes}
	cert.Leaf = certificate
	cert.PrivateKey = privateKey

	return &cert, nil
}
