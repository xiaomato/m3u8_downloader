// package m3u8downloader
package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/xiaomato/m3u8_downloader/cript"
	"github.com/xiaomato/m3u8_downloader/m3u8"
)

type downloader struct {
	links     []string
	linkFiles map[string]string
	infos     map[string]string
	filename  string
	n         int32
	tmpDir    string
	available chan struct{}
	done      chan struct{}
}

func NewM3u8Downloader(url string, filename string, n int) (*downloader, error) {
	f, err := os.MkdirTemp("./", "")
	if err != nil {
		return nil, err
	}
	links, infos, err := m3u8.ParseURL(url)
	if err != nil {
		return nil, err
	}
	avalable := make(chan struct{}, n)
	for i := 0; i < n; i++ {
		avalable <- struct{}{}
	}
	return &downloader{
		links:     links,
		infos:     infos,
		filename:  fmt.Sprintf("./output/%s.ts", filename),
		tmpDir:    f,
		available: avalable,
		n:         0,
		linkFiles: map[string]string{},
		done:      make(chan struct{}),
	}, nil
}

func (d *downloader) Download() error {
	println(d.filename)
	for i, v := range d.links {
		d.linkFiles[v] = fmt.Sprintf("%s/%v.ts", d.tmpDir, i)
	}
	linkChan := make(chan string, len(d.links))
	for _, v := range d.links {
		linkChan <- v
	}
	for {
		select {
		case link := <-linkChan:
			<-d.available
			go func(l string) {
				data, err := d.downloadLink(l)
				if err != nil {
					println(err.Error())
					linkChan <- l
					return
				}
				data, err = cript.AES128Decrypt(data, []byte(d.infos["KEY"]), nil)
				if err != nil {
					println(err.Error())
					linkChan <- l
					return
				}
				if err := d.saveFile(d.linkFiles[l], data); err != nil {
					println(err.Error())
					linkChan <- l
					return
				}
				d.available <- struct{}{}
				atomic.AddInt32(&d.n, 1)
				if int(d.n) == len(d.links) {
					close(d.done)
				}
			}(link)
		case <-d.done:
			return d.merge()
		}
	}
}

func (d *downloader) merge() error {
	defer os.RemoveAll(d.tmpDir)
	file, err := os.Create(d.filename)
	if err != nil {
		return err
	}
	writer := bufio.NewWriter(file)
	for _, v := range d.links {
		data, err := ioutil.ReadFile(d.linkFiles[v])
		if err != nil {
			continue
		}
		_, err = writer.Write(data)
		if err != nil {
			continue
		}
	}
	return writer.Flush()
}

func (d *downloader) downloadLink(link string) ([]byte, error) {
	c := http.Client{
		Timeout: time.Minute,
	}
	rsp, err := c.Get(link)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (d *downloader) saveFile(filename string, data []byte) error {
	return ioutil.WriteFile(filename, data, 0777)

}
