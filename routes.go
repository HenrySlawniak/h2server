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
	"github.com/go-playground/log"
	"net/http"
	"os"
	"strings"
)

func addRoutesToRouter() {
	server.r.Host("ifcfg.org").Name("ifcfg.org").PathPrefix("/").HandlerFunc(server.ifcfgRootHandler)
	server.r.Host("v4.ifcfg.org").Name("ifcfg.org-v4").PathPrefix("/").HandlerFunc(server.ifcfgRootHandler)
	server.r.Host("v6.ifcfg.org").Name("ifcfg.org-v6").PathPrefix("/").HandlerFunc(server.ifcfgRootHandler)

	server.r.Host("stopallthe.download").Path("/ing/provision").Handler(http.RedirectHandler("https://gist.githubusercontent.com/HenrySlawniak/c31cedaec491c68631a6f62b5d94a740/raw", http.StatusFound))
	server.r.Host("stopallthe.download").Path("/ing/install-go").Handler(http.RedirectHandler("https://gist.githubusercontent.com/HenrySlawniak/1b17dc248f57016ee820a7502d7285ce/raw", http.StatusFound))
	server.r.Host("stopallthe.download").Name("stopall").PathPrefix("/ing/").HandlerFunc(server.stopAllIngHandler)
	server.r.Host("stopallthe.download").Name("stopall").PathPrefix("/").HandlerFunc(server.stopAllRootHandler)

	server.r.PathPrefix("/").HandlerFunc(server.indexHandler).Name("catch-all")
}

// GetIP returns the remote ip of the request, by stripping off the port from the RemoteAddr
func GetIP(r *http.Request) string {
	split := strings.Split(r.RemoteAddr, ":")
	ip := strings.Join(split[:len(split)-1], ":")
	// This is bad, and I feel bad
	ip = strings.Replace(ip, "[", "", 1)
	ip = strings.Replace(ip, "]", "", 1)
	return ip
}

func (s *Server) ifcfgRootHandler(w http.ResponseWriter, r *http.Request) {
	ip := GetIP(r)
	w.Header().Set("Server", "ifcfg.org")

	if strings.Contains(r.Header.Get("User-Agent"), "curl") || r.Header.Get("Accept") == "text/plain" {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(ip + "\n"))
		return
	}
	w.Write([]byte(ip))
}

func (s *Server) stopAllRootHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/ing"+r.URL.RequestURI(), http.StatusFound)
}

func (s *Server) stopAllIngHandler(w http.ResponseWriter, r *http.Request) {
	serveFile(w, r, "stopall/client"+strings.Replace(r.URL.RequestURI(), "/ing", "", -1))
}

func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	host := strings.Split(r.Host, ":")[0]

	if !domainIsRegistered(host) {
		log.Debugf("Host is %s", host)
		// Make sure we do this syncronousley
		addToDomainList(host)
	}

	staticFolder := "./sites/" + host
	if _, err := os.Stat(staticFolder); err != nil {
		staticFolder = "./client"
	}

	var n int64
	var code int

	if inf, err := os.Stat(staticFolder + path); err == nil && !inf.IsDir() {
		n, code = serveFile(w, r, staticFolder+path)
	} else if inf, err := os.Stat(staticFolder + path + "/index.html"); err == nil && !inf.IsDir() {
		n, code = serveFile(w, r, staticFolder+path+"/index.html")
	} else {
		n, code = serveFile(w, r, staticFolder+"/index.html")
	}

	go logRequest(w, r, n, code)
}
