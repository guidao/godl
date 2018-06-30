package download

import (
	"log"
)

type Config struct {
	URL      string
	FileName string
	N        int
}

type Download struct {
	Cfg     *Config
	Plugins []Plugin
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

}
