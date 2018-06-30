package main

import (
	"fmt"
	dl "github.com/guidao/godl/download"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"path/filepath"
)

var godl = &cobra.Command{
	Use:   "godl",
	Short: "godl",
	Long:  "powerful download tools",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := &dl.Config{
			URL:      args[0],
			N:        viper.GetInt("thread"),
			FileName: viper.GetString("name"),
		}
		if cfg.URL == "" {
			cfg.URL = viper.GetString("url")
		}
		if cfg.URL == "" {
			fmt.Println("未找到下载地址")
			return
		}
		if cfg.FileName == "" {
			cfg.FileName = filepath.Base(cfg.URL)
		}
		dl.DownloadFile(cfg)
	},
}

func init() {
	godl.Flags().IntP("thread", "n", 1, "-n 2")
	godl.Flags().StringP("name", "o", "", "-o xxx.pdf")
	godl.Flags().StringP("url", "u", "", "-u http://aaa.bbb/xxx.pdf")
	viper.BindPFlags(godl.Flags())
}

func main() {
	if err := godl.Execute(); err != nil {
		fmt.Println("download file", err)
	}
}
