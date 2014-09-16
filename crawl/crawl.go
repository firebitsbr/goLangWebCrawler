package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/codegangsta/cli"
	"github.com/missionMeteora/crawler"
)

func crawlSites(c *cli.Context) {
	level := 1 //c.Int("level")
	jobs := c.Int("jobs")
	skip := c.Int("skip")

	// verify args
	if level < 1 || level > 3 {
		log.Fatalln("invalid deepness level, value is out of range")
	}
	if jobs < 1 || jobs > 1024 {
		log.Fatalln("invalid threads number, value is out of range")
	}
	if skip < 0 {
		log.Fatalln("invalid skip number, value is less than 0")
	}
	// open resources
	db := mustOpenDB(c.String("db"))
	defer db.Close()
	patterns := mustLoadPatterns(c.String("patterns"))
	out := mustOpenAppendFile(c.String("out"))
	defer out.Close()

	// setup runtime performance
	nCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(nCPU * 2)
	log.Printf("crawl started on %d cores, jobs: %d, deepness: %d", nCPU, jobs, level)

	// crawler init
	resultsC := make(chan *crawler.Result, parallelBufSize)
	errorsC := make(chan struct{}, parallelBufSize)
	cr := crawler.New(db, &crawler.Config{
		Jobs:     jobs,
		Level:    level,
		Patterns: patterns,
	})

	// bar setup
	bar := pb.New64(sitesNum - int64(skip))
	bar.SetRefreshRate(time.Second)
	bar.ShowTimeLeft = true
	bar.ShowSpeed = true
	bar.Start()
	cr.OnProgress = bar.Increment
	defer bar.FinishPrint("All sites have been crawled.")

	// result processing routine
	go func() {
		for result := range resultsC {
			if err := writeResults(out, result); err != nil {
				log.Println("error saving result:", err)
			}
		}
	}()
	// errors processing routine
	var errCounter int
	go func() {
		for _ = range errorsC {
			errCounter++
		}
	}()
	// interrupt watcher
	exitC := make(chan os.Signal, 1)
	signal.Notify(exitC, os.Interrupt)
	go func() {
		<-exitC
		log.Println("Errors total:", errCounter)
		os.Exit(0)
	}()

	// begin crawling
	if err := cr.Crawl(resultsC, errorsC, 0); err != nil {
		log.Fatalln("unable to start crawling:", err)
	}

	log.Println("Errors total:", errCounter)
}

func mustOpenAppendFile(name string) *os.File {
	file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalln(err)
	}
	return file
}

func writeResults(w io.Writer, res *crawler.Result) error {
	buf := bufio.NewWriter(w)
	fmt.Fprintf(buf, "URL: %s\n", res.URL)
	fmt.Fprintf(buf, "Patterns: %v\n", res.Patterns)
	fmt.Fprintf(buf, "Title: %s\n", res.Title)
	fmt.Fprintf(buf, "Description: %s\n", res.Description)
	fmt.Fprintf(buf, "Language: %s\n", strings.ToUpper(res.Language))
	buf.WriteString("===========================================================\n\n")
	return buf.Flush()
}

func mustLoadPatterns(name string) (patterns map[string]string) {
	buf, err := ioutil.ReadFile(name)
	if err != nil {
		log.Println(err)
	}
	if err := json.Unmarshal(buf, &patterns); err != nil {
		log.Println("error loading patterns:", err)
	}
	return
}
