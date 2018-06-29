package main // import "github.com/guidao/godl"

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

type Config struct {
	URL            string
	IsSupportRange bool
	Size           int64
	CurrSize       int64
	TotalConn      int
	WG             sync.WaitGroup
	write          chan []byte
}

type Chunk struct {
	Index int
	Start int64
	End   int64
}

func (this Chunk) Range() string {
	if this.End <= 0 {
		return fmt.Sprintf("%v-", this.Start)
	}
	return fmt.Sprintf("%v-%v", this.Start, this.End)
}

func main() {
	cfg := new(Config)
	cfg.URL = "http://box2d.org/manual.pdf"
	cfg.TotalConn = 3
	err := download(cfg)
	if err != nil {
		log.Printf("err:%v\n", err)
	}
}

func download(cfg *Config) error {
	resp, err := http.Head(cfg.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	rg := resp.Header.Get("Accept-Ranges")
	if rg != "" && rg != "none" {
		cfg.IsSupportRange = true
	} else {
		cfg.TotalConn = 1
	}
	size := resp.Header.Get("Content-Length")
	if size != "" {
		cfg.Size, err = strconv.ParseInt(size, 10, 64)
		if err != nil {
			return err
		}
	}
	return download(cfg)
}

func downloadFile(cfg *Config) error {
	cfg.WG.Add(cfg.TotalConn)
	chunks := splitChunk(cfg)
	fileName := filepath.Base(cfg.URL)
	for i := range chunks {
		go func(chunk Chunk) {
			defer cfg.WG.Done()

			req, err := http.NewRequest("GET", cfg.URL, nil)
			if err != nil {
				return
			}
			req.Header.Add("Range", chunk.Range())
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()
			chunkName := fmt.Sprintf("%v_%v", fileName, chunk.Index)
			fd, err := os.Create(chunkName)
			if err != nil {
				return
			}
			io.Copy(fd, resp.Body)
			fd.Close()
		}(chunks[i])
	}
	cfg.WG.Wait()
	//Merge
	fileNameN := fmt.Sprintf("%v_%v", fileName, chunks[0].Index)
	fd, err := os.OpenFile(fileNameN, os.O_APPEND, 0)
	if err != nil {
		return err
	}
	for i := range chunks[1:] {
		fileName := fmt.Sprintf("%v_%v", fileName, chunks[i].Index)
		fdI, err := os.Open(fileName)
		if err != nil {
			return err
		}
		io.Copy(fd, fdI)
		fdI.Close()
	}
	fd.Close()
	return nil
}

func splitChunk(cfg *Config) []Chunk {
	var chunks []Chunk
	start := int64(0)
	length := cfg.Size / int64(cfg.TotalConn)
	i := 0
	for {
		if start > cfg.Size {
			break
		}
		end := start + length
		if end > cfg.Size {
			end = 0
		}
		chunks = append(chunks, Chunk{
			Index: i,
			Start: start,
			End:   start + length,
		})
		i++
		start += length
	}
	return chunks
}
