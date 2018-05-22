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
	"fmt"
	"github.com/NYTimes/gziphandler"
	"github.com/gorilla/mux"
	"net/http"
)

// Server wraps our *mux.Router so we can globally modify responses
type Server struct {
	r *mux.Router
}

var server Server

func (s Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Server", fmt.Sprintf("Gate v%s (%s)", commit, buildTime))
	if !devMode {
		w.Header().Set("Content-Security-Policy", fmt.Sprintf("default-src https://*.%[1]s; form-action https://*.%[1]s; block-all-mixed-content; upgrade-insecure-requests", domain))
	}
	gziphandler.GzipHandler(s.r).ServeHTTP(w, req)
}

func setupHTTPServer() {
	server = Server{
		r: mux.NewRouter(),
	}
}
