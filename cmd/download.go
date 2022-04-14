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

	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

func Download(cmd *cobra.Command, args []string) {
	url, _ := cmd.Flags().GetString("url")
	filename, _ := cmd.Flags().GetString("output")
	concurrency, _ := cmd.Flags().GetInt("concurrency")
	resume, _ := cmd.Flags().GetBool("resume")

	fmt.Printf("[url] %v\n[filename] %v\n[concurrency] %v\n", url, filename, concurrency)

	NewDownloader(concurrency, resume).Download(url, filename)
}

type Downloader struct {
	concurrency int
	resume      bool

	bar *progressbar.ProgressBar
}

func NewDownloader(concurrency int, resume bool) *Downloader {
	return &Downloader{concurrency: concurrency, resume: resume}
}

func (d *Downloader) Download(url, filename string) error {
	if filename == "" {
		filename = path.Base(url)
	}

	resp, err := http.Head(url)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	if resp.StatusCode == http.StatusOK && resp.Header.Get("Accept-Ranges") == "bytes" {
		fmt.Println("[mode] multi")
		return d.multiDownload(url, filename, int(resp.ContentLength))
	}

	fmt.Println("[mode] single")
	return d.singleDownload(url, filename)
}

func (d *Downloader) multiDownload(url, filename string, contentLen int) error {
	d.setBar(contentLen)

	partSize := contentLen / d.concurrency

	partDir := d.getPartDir(filename)
	os.Mkdir(partDir, 0777)
	defer os.RemoveAll(partDir)

	var wg sync.WaitGroup
	wg.Add(d.concurrency)

	rangeStart := 0

	for i := 0; i < d.concurrency; i++ {
		go func(i, rangeStart int) {
			defer wg.Done()

			rangeEnd := rangeStart + partSize
			if i == d.concurrency-1 {
				rangeEnd = contentLen
			}

			downloaded := 0
			if d.resume {
				partFileName := d.getPartFilename(filename, i)
				content, err := os.ReadFile(partFileName)
				if err == nil {
					downloaded = len(content)
				}
				d.bar.Add(downloaded)
			}

			d.downloadPartial(url, filename, rangeStart, rangeEnd, i)
		}(i, rangeStart)

		rangeStart += partSize + 1
	}

	wg.Wait()

	d.merge(filename)

	return nil
}

func (d *Downloader) singleDownload(url, filename string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	d.setBar(int(resp.ContentLength))

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	buf := make([]byte, 32*1024)
	_, err = io.CopyBuffer(io.MultiWriter(f, d.bar), resp.Body, buf)
	return err
}

func (d *Downloader) downloadPartial(url, filename string, rangeStart, rangeEnd, i int) {
	if rangeStart >= rangeEnd {
		return
	}

	req, err := http.NewRequest("GET", url, nil)
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
	partFile, err := os.OpenFile(d.getPartFilename(filename, i), flags, 0666)
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

func (d *Downloader) getPartDir(filename string) string {
	return strings.SplitN(filename, ".", 2)[0]
}

func (d *Downloader) getPartFilename(filename string, partNum int) string {
	partDir := d.getPartDir(filename)
	return fmt.Sprintf("%s/%s-%d", partDir, filename, partNum)
}

// merge 合并文件
func (d *Downloader) merge(filename string) error {
	dstFile, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return nil
	}
	defer dstFile.Close()

	for i := 0; i < d.concurrency; i++ {
		partFilename := d.getPartFilename(filename, i)
		partFile, err := os.Open(partFilename)
		if err != nil {
			return err
		}
		io.Copy(dstFile, partFile)
		partFile.Close()
		os.Remove(partFilename)
	}

	return nil
}

func (d *Downloader) setBar(length int) {
	d.bar = progressbar.NewOptions(
		length,
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(50),
		progressbar.OptionSetDescription("Downloading..."),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)
}
