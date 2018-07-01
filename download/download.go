package download

import (
	//"fmt"
	"github.com/gosuri/uiprogress"
	"log"
)

type Config struct {
	URL       string
	FileName  string
	N         int
	HTTPProxy string
}

type Download struct {
	Cfg      *Config
	Plugins  []Plugin
	progress []*uiprogress.Bar
}

type Plugin interface {
	Match(url string) bool
	Download(cfg *Config) (<-chan []Progress, error)
}

type Progress struct {
	Desc      string
	TotalSize int64
	CurrSize  int64
}

func DownloadFile(cfg *Config) {
	service := NewDownload(cfg)
	service.Start()
}

func NewDownload(cfg *Config) *Download {
	return &Download{
		Cfg:     cfg,
		Plugins: []Plugin{NewHTTPPlugin()},
	}
}

func (this *Download) Start() {
	uiprogress.Start()
	for _, pl := range this.Plugins {
		if !pl.Match(this.Cfg.URL) {
			continue
		}
		ui, err := pl.Download(this.Cfg)
		if err != nil {
			log.Println("download err:", err)
			continue
		}
		for progress := range ui {
			this.Draw(progress)
		}
		break
	}
}

func (this *Download) Draw(progress []Progress) {
	if this.progress == nil {
		for _, p := range progress {
			bar := uiprogress.AddBar(int(p.TotalSize))
			bar.AppendCompleted().AppendElapsed()
			this.progress = append(this.progress, bar)
		}
	}
	for i, p := range progress {
		this.progress[i].Set(int(p.CurrSize))
	}
}
