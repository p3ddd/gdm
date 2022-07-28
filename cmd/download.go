package cmd

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

const UserAgent = `Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2704.103 Safari/537.36`

func DownloadFunc(cmd *cobra.Command, args []string) {
	downloader := NewDownloader(concurrencyFlag, resumeFlag)

	// 下载链接
	if urlFlag == "" {
		downloader.url = args[0]
	} else {
		downloader.url = urlFlag
	}

	// 文件名与输出路径
	if outputFlag == "" {
		downloader.filename = path.Base(downloader.url)
	} else {
		downloader.dirname, downloader.filename = path.Split(outputFlag)
	}
	fmt.Println("Dirname:", downloader.dirname, "Filename:", downloader.filename)

	res, err := http.Head(downloader.url)
	CheckErr(err)
	downloader.contentLength = int(res.ContentLength)
	fmt.Println("Size:", res.ContentLength, "bytes")
	if res.StatusCode == http.StatusOK && res.Header.Get("Accept-Ranges") == "bytes" {
		downloader.acceptRanges = true
	} else {
		downloader.acceptRanges = false
	}

	downloader.setBar(downloader.contentLength)

	downloader.Download()
}

type Downloader struct {
	url         string // 链接
	filename    string // 文件名
	dirname     string // 保存目录名
	concurrency int    // 并发数
	resume      bool   // 是否断点续传

	contentLength int  // 文件总大小
	acceptRanges  bool // 服务端是否支持 Ranges

	bar *progressbar.ProgressBar // 下载进度条
}

func NewDownloader(concurrency int, resume bool) *Downloader {
	return &Downloader{concurrency: concurrency, resume: resume}
}

func (d *Downloader) Download() error {
	if d.acceptRanges {
		fmt.Println("Mode: multi")
		return d.multiDownload()
	} else {
		fmt.Println("Mode: single")
		return d.singleDownload()
	}
}

func (d *Downloader) multiDownload() error {
	partSize := d.contentLength / d.concurrency
	fmt.Println("Part Size:", partSize, "bytes")

	partDir := d.getPartDir()
	os.Mkdir(partDir, 0777)
	defer os.RemoveAll(partDir)

	var wg sync.WaitGroup
	wg.Add(d.concurrency)

	rangeStart := 0

	for i := 0; i < d.concurrency; i++ {
		go func(i, rangeStart int) {
			defer wg.Done()

			// 范围是 i*size ~ (i+1)*size-1
			rangeStart = i * partSize
			rangeEnd := rangeStart + partSize - 1

			// 若是最后一部分，则设置范围到文件总长度
			if i == d.concurrency-1 {
				rangeEnd = d.contentLength - 1
			}

			// 当前分片已下载量
			downloaded := 0
			// 是否继续下载
			if d.resume {
				partFileName := d.getPartFilename(i)
				content, err := os.ReadFile(partFileName)
				if err == nil {
					downloaded = len(content)
				}
				// 设置进度条中已下载的部分
				d.bar.Add(downloaded)
			}

			// TODO resume
			d.downloadPartial(rangeStart, rangeEnd, i)
		}(i, rangeStart)
	}

	// 等待全部分片下载完成
	wg.Wait()

	// 合并文件
	d.merge()

	return nil
}

func (d *Downloader) singleDownload() error {
	res, err := http.Get(d.url)
	CheckErr(err)
	defer res.Body.Close()

	f, err := os.OpenFile(d.filename, os.O_CREATE|os.O_WRONLY, 0666)
	CheckErr(err)
	defer f.Close()

	buf := make([]byte, 32*1024) // 32 KB
	_, err = io.CopyBuffer(io.MultiWriter(f, d.bar), res.Body, buf)
	return err
}

func (d *Downloader) downloadPartial(rangeStart, rangeEnd, i int) {
	if rangeStart >= rangeEnd {
		return
	}

	req, err := http.NewRequest("GET", d.url, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", rangeStart, rangeEnd))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	flags := os.O_CREATE | os.O_WRONLY
	if d.resume {
		flags |= os.O_APPEND
	}
	partFile, err := os.OpenFile(d.getPartFilename(i), flags, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer partFile.Close()

	buf := make([]byte, 32*1024)
	_, err = io.CopyBuffer(io.MultiWriter(partFile, d.bar), resp.Body, buf)
	if err != nil {
		if err != io.EOF {
			log.Fatal(err)
		}
	}
}

// 获取分片文件夹名称
func (d *Downloader) getPartDir() string {
	return strings.SplitN(d.filename, ".", 2)[0]
}

// 获取分片文件名称
func (d *Downloader) getPartFilename(partNum int) string {
	partDir := d.getPartDir()
	return fmt.Sprintf("%s/%s-%d", partDir, d.filename, partNum)
}

// merge 合并文件
func (d *Downloader) merge() error {
	filename := path.Join(d.dirname, d.filename)
	dstFile, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0666)
	CheckErr(err)
	defer dstFile.Close()

	for i := 0; i < d.concurrency; i++ {
		partFilename := d.getPartFilename(i)
		partFile, err := os.Open(partFilename)
		CheckErr(err)
		io.Copy(dstFile, partFile)
		partFile.Close()
		os.Remove(partFilename)
	}

	return nil
}

func (d *Downloader) setBar(length int) {
	d.bar = progressbar.NewOptions(
		length,
		progressbar.OptionSpinnerType(11),
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(true),
		progressbar.OptionFullWidth(), // 进度条宽度
		// progressbar.OptionSetWidth(50),
		progressbar.OptionThrottle(200*time.Millisecond), // 再次更新之前等待时间
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionOnCompletion(func() {
			fmt.Printf("\n")
		}),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)
}
