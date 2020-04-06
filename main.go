// Copyright (c) 2018 Henry Slawniak <https://datacenterscumbags.com/>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"github.com/newrelic/go-agent/v3/integrations/nrlogrus"
	"github.com/newrelic/go-agent/v3/newrelic"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/net/http2"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var (
	buildTime  string
	commit     string
	domainList = []string{}
	m          autocert.Manager

	devMode bool
	domain  string
	listen  string
	dial    string
	nrKey   string
	app     *newrelic.Application

	logFile = filepath.Join(".logs", "access.log")
	f       *os.File
)

func init() {
	flag.Parse()
	os.Setenv("GODEBUG", os.Getenv("GODEBUG")+",tls13=1")

	// <phil> I hate this stuff so much, I try to avoid it - environment variables have shortened my life in I'm sure a measurable way
	devMode = os.Getenv("DEV") == "true" || false
	var found bool

	domain, found = os.LookupEnv("DOMAIN")
	if !found {
		domain = "henry.network"
	}

	listen, found = os.LookupEnv("LISTEN")
	if !found {
		listen = ":https"

	}

	if found {
		log.Info("Setting up New Relic Go Agent")
		var err error
		app, err = newrelic.NewApplication(
			newrelic.ConfigFromEnvironment(),
			nrlogrus.ConfigStandardLogger(),
			newrelic.ConfigAppName("h2server"),
		)
		if err != nil {
			log.Panic(err)
		}
	}

	var err error

	if _, err = os.Stat(filepath.Dir(logFile)); os.IsNotExist(err) {
		os.MkdirAll(filepath.Dir(logFile), 0755)
	}

	f, err = os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0750)
	if err != nil {
		log.Panic(err)
	}
}

func main() {
	defer f.Close()
	log.Info("Starting h2server")
	if buildTime != "" {
		log.Info("Built: " + buildTime)
	}
	if commit != "" {
		log.Info("Revision: " + commit)
	}
	log.Info("Go: " + runtime.Version())

	loadDomainList()

	setupHTTPServer()
	addRoutesToRouter()

	if devMode {
		rootSrv := &http.Server{
			Addr:    listen,
			Handler: server,

			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
		}

		log.Fatal(rootSrv.ListenAndServe())
	} else {
		tlsConf := &tls.Config{
			MinVersion:               tls.VersionTLS12,
			PreferServerCipherSuites: true,

			CurvePreferences: []tls.CurveID{
				tls.CurveP256,
				tls.X25519,
			},

			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			},
		}

		m = autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(domainList...),
			Cache:      autocert.DirCache("certs"),
		}

		tlsConf.GetCertificate = m.GetCertificate

		rootSrv := &http.Server{
			Addr:      listen,
			Handler:   server,
			TLSConfig: tlsConf,

			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
		}
		go http.ListenAndServe(":http", m.HTTPHandler(nil))

		log.Infof("Listening on %s", listen)

		http2.ConfigureServer(rootSrv, &http2.Server{})
		log.Fatal(rootSrv.ListenAndServeTLS("", ""))
	}

}

func domainIsRegistered(domain string) bool {
	for _, d := range domainList {
		if d == domain {
			return true
		}
	}
	return false
}

func addToDomainList(domain string) {
	domain = strings.TrimSpace(domain)
	log.Infof("Adding %s to domain list", domain)
	if domain == "" {
		log.Warn("Cannot use an empty string as a domain")
		return
	}

	for _, d := range domainList {
		if d == domain {
			log.Infof("%s already in domain list, returning\n", domain)
			return
		}
	}
	domainList = append(domainList, domain)

	m.HostPolicy = autocert.HostWhitelist(domainList...)
}

func loadDomainList() {
	var f *os.File
	var err error
	if _, err = os.Stat("domains.txt"); os.IsNotExist(err) {
		f, err = os.Create("domains.txt")
		if err != nil {
			log.Fatal(err)
			panic(err)
		}
	} else {
		f, err = os.Open("domains.txt")
		if err != nil {
			log.Fatal(err)
			panic(err)
		}
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}

	for scanner.Scan() {
		addToDomainList(scanner.Text())
	}

	envDomains, found := os.LookupEnv("DOMAINS")
	if found {
		for _, domain := range strings.Split(envDomains, ",") {
			addToDomainList(domain)
		}
	}

	log.Infof("There are now %d domains registered\n", len(domainList))
}
