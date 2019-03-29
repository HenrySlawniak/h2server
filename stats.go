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
	"net"
	"net/http"
	"strings"
	"time"
)

var loc, _ = time.LoadLocation("America/Chicago")

func logRequest(w http.ResponseWriter, r *http.Request, bytes int64, responseCode int) {
	host := r.Host
	var err error

	if strings.Contains(host, ":") {
		host, _, err = net.SplitHostPort(r.Host)
		if err != nil {
			log.Error(err)
		}
	}

	ip := r.RemoteAddr

	if strings.Contains(ip, "127.0.0.1") || strings.Contains(ip, "[::1]") {
		if r.Header.Get("X-Real-IP") != "" {
			ip = r.Header.Get("X-Real-IP")
			log.Debug(ip)
		}
	}

	ip, _, err = net.SplitHostPort(ip)
	if err != nil {
		log.Error(err)
	}

	// remote_addr - [local_time] "method host path query protocol" response_code bytes_written referer user_agent
	logStr := fmt.Sprintf(
		"%s - [%s] \"%s %s %s %s %s\" %d %d \"%s\" \"%s\"",
		ip,
		time.Now().In(loc).Format("02/Jan/2006:15:04:05 -0700"),
		r.Method,
		host,
		r.URL.Path,
		r.URL.RawQuery,
		r.Proto,
		responseCode,
		bytes,
		r.Referer(),
		r.UserAgent(),
	)

	f.WriteString(logStr + "\n")
}
