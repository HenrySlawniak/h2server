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
	"crypto/md5"
	"fmt"
	"github.com/newrelic/go-agent/v3/newrelic"
	log "github.com/sirupsen/logrus"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type fileSum struct {
	Time     time.Time
	Sum      string
	Modified time.Time
	Size     int64
}

var sums = map[string]*fileSum{}

var mu = &sync.Mutex{}

func init() {
	mime.AddExtensionType(".webm", "video/webm")
}

func serveFile(w http.ResponseWriter, r *http.Request, path string) (int64, int) {
	var txn *newrelic.Transaction
	if app != nil {
		txn = app.StartTransaction("serve-" + path)
		defer txn.End()
	}

	if path == "./client/" {
		path = "./client/index.html"
	}

	if path == "stopall/client/" {
		path = "./stopall/client/index.html"
	}

	var seg *newrelic.Segment

	if txn != nil {
		seg = newrelic.StartSegment(txn, "stat-"+path)
	}

	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		log.Error(err)
		return 0, http.StatusNotFound
	}

	if txn != nil {
		seg.End()
	}

	if txn != nil {
		seg = newrelic.StartSegment(txn, "sum-"+path)
	}

	sum, err := getFileSum(path)
	if err != nil {
		log.Error(err)
		return 0, http.StatusInternalServerError
	}

	if txn != nil {
		seg.End()
	}

	w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(path)))
	if w.Header().Get("Cache-Control") == "" {
		w.Header().Set("Cache-Control", "public")
	}
	w.Header().Set("Last-Modified", sum.Modified.UTC().Format(time.RFC1123))
	w.Header().Set("Expires", time.Now().UTC().Add(1*time.Hour).Format(time.RFC1123))
	w.Header().Set("ETag", sum.Sum)

	if r.Header.Get("If-None-Match") == sum.Sum {
		w.WriteHeader(http.StatusNotModified)
		return 0, http.StatusNotModified
	}

	http.ServeFile(w, r, path)
	return int64(sum.Size), http.StatusOK
}

func getFileSum(path string) (*fileSum, error) {
	sum := sums[path]
	if sum != nil {
		stat, err := os.Stat(path)
		if err != nil {
			return nil, err
		}

		if stat.ModTime().Unix() > sum.Modified.Unix() {
			return generateAndCacheSum(path)
		}

		if sum.Time.Add(15*time.Minute).Unix() < time.Now().Unix() {
			return generateAndCacheSum(path)
		}

		return sum, nil
	}

	return generateAndCacheSum(path)
}

func generateAndCacheSum(path string) (*fileSum, error) {
	mu.Lock()
	defer mu.Unlock()
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}

	summer := md5.New()
	io.Copy(summer, f)

	sum := &fileSum{
		Time:     time.Now(),
		Sum:      fmt.Sprintf("md5-%x", summer.Sum(nil)),
		Modified: stat.ModTime(),
		Size:     stat.Size(),
	}

	sums[path] = sum
	return sum, nil
}
