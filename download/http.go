package download

import (
	"net/http"
	"net/url"

	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type HTTPPlugin struct {
	client    *http.Client
	process   []Progress
	tickClose chan bool
	ui        chan []Progress
	fds       []*os.File

	size int64
	wg   sync.WaitGroup
}

func NewHTTPPlugin() *HTTPPlugin {
	return &HTTPPlugin{
		client:    http.DefaultClient,
		tickClose: make(chan bool),
		ui:        make(chan []Progress),
	}
}

func (this *HTTPPlugin) Match(uri string) bool {
	u, err := url.Parse(uri)
	if err != nil {
		return false
	}
	return u.Scheme == "http"
}

func (this *HTTPPlugin) Download(cfg *Config) (<-chan []Progress, error) {
	size, supportRange := this.Info(cfg.URL)
	if size <= 0 || !supportRange {
		cfg.N = 1
	}
	this.size = size
	chunks := SplitChunk(size, cfg.N)
	this.process = make([]Progress, len(chunks))
	for i, chunk := range chunks {
		chunks[i].sort = i
		name := fmt.Sprintf("%v-%v.godl", cfg.FileName, chunk.sort)
		fd, err := os.Create(name)
		if err != nil {
			return nil, err
		}
		this.fds = append(this.fds, fd)
	}
	go this.tick()

	this.wg.Add(len(chunks))
	for i := range chunks {
		go func(i int) {
			this.DownloadChunk(cfg, chunks[i])
			this.wg.Done()
		}(i)
	}
	go this.WaitMerge(cfg)
	return this.ui, nil
}

func (this *HTTPPlugin) WaitMerge(cfg *Config) {
	this.wg.Wait()
	this.tickClose <- true
	close(this.tickClose)
	fd0 := this.fds[0]
	for _, fd := range this.fds[1:] {
		fd.Seek(0, os.SEEK_SET)
		io.Copy(fd0, fd)
		fd.Close()
		os.Remove(fd.Name())
	}
	fd0.Close()
	os.Rename(fd0.Name(), cfg.FileName)
	close(this.ui)
}

func (this *HTTPPlugin) tick() {
	tk := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-tk.C:
			this.ui <- this.process
		case <-this.tickClose:
			tk.Stop()
			return
		}
	}
}

func (this *HTTPPlugin) DownloadChunk(cfg *Config, chunk Chunk) {
	for i := 0; i < 10; i++ {
		req, err := http.NewRequest("GET", cfg.URL, nil)
		if err != nil {
			continue
		}
		req.Header.Add("Range", chunk.Range())
		for _, header := range cfg.Header {
			v := strings.SplitN(header, ":", 2)
			if len(v) != 2 {
				continue
			}
			req.Header.Add(v[0], v[2])
		}
		if req.Header.Get("Content-Type") == "" {
			req.Header.Add("Content-Type", "godl/0.1")
		}
		resp, err := this.client.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		fd := this.fds[chunk.sort]
		fd.Seek(0, os.SEEK_SET)
		length := (chunk.end-chunk.start)/100 + 1
		if chunk.start > chunk.end {
			length = (this.size-chunk.start)/100 + 1
		}
		step := make([]byte, length)
		finish := false
		this.process[chunk.sort].TotalSize = chunk.end - chunk.start
		if chunk.start == 0 && chunk.end == 0 {
			this.process[chunk.sort].TotalSize = this.size
		}
		if this.process[chunk.sort].TotalSize <= 0 {
			this.process[chunk.sort].TotalSize = this.size - chunk.start
		}
		this.process[chunk.sort].Desc = filepath.Base(fd.Name())
		for {
			n, err := resp.Body.Read(step)
			if err != nil {
				if err == io.EOF {
					finish = true
				}
			}
			fd.Write(step[0:n])
			this.process[chunk.sort].CurrSize += int64(n)
			if err != nil {
				break
			}
		}
		if !finish {
			continue
		}
		return
	}
}

func (this *HTTPPlugin) Info(u string) (int64, bool) {
	resp, err := this.client.Head(u)
	if err != nil {
		return 0, false
	}
	v := resp.Header.Get("Content-Length")
	if v == "" {
		return 0, false
	}
	size, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, false
	}
	r := resp.Header.Get("Accept-Ranges")
	if r != "" && r != "none" {
		return size, true
	}
	return size, false
}
