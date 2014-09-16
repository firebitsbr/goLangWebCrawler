package crawler

import (
	"bytes"
	"errors"
	"runtime"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
	"github.com/missionMeteora/go-metainspector/metainspector"
	"github.com/rakyll/coop"
)

var parallelN = runtime.NumCPU() * 2
var parallelBufSize = 128 * parallelN

var BucketSites = []byte("sites")

var ErrNoMatch = errors.New("site does not have any of the patterns")

const connTimeout = time.Second * 10

type Config struct {
	Jobs     int
	Level    int
	Patterns map[string]string
}

type Crawler struct {
	OnProgress func() int

	db   *bolt.DB
	cfg  *Config
	uriC chan string
}

type Result struct {
	URL         string
	Patterns    []string
	Title       string
	Description string
	Language    string
}

// New creates a new crawler.
func New(db *bolt.DB, cfg *Config) *Crawler {
	return &Crawler{
		db:  db,
		cfg: cfg,
	}
}

// Crawl starts crawling from specific rank offset.
func (c *Crawler) Crawl(resultsC chan<- *Result, errorsC chan<- struct{}, skip int) error {
	c.uriC = make(chan string, parallelBufSize)
	skipB := strconv.AppendInt([]byte{}, int64(skip), 10)
	if err := c.checkBucket(BucketSites); err != nil {
		return err
	}

	// fill the pool of URLs
	go func() {
		c.db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket(BucketSites)
			b.ForEach(func(k, v []byte) error {
				if bytesLess(k, skipB) {
					return nil
				}
				c.uriC <- string(v)
				return nil
			})
			return nil
		})
		close(c.uriC)
	}()

	process := func() {
		for uri := range c.uriC {
			if c.OnProgress != nil {
				c.OnProgress()
			}
			result, err := c.crawlURI(uri)
			if err != nil {
				if err != ErrNoMatch {
					errorsC <- struct{}{} // no report, just counting
					// log.Println("error crawling resource:", err)
				}
				continue
			}
			resultsC <- result
		}
	}

	// run processing jobs
	<-coop.Replicate(c.cfg.Jobs, process)
	close(resultsC)
	close(errorsC)
	return nil
}

func (c *Crawler) crawlURI(uri string) (result *Result, err error) {
	mi, err := metainspector.New(uri, &metainspector.Config{
		Timeout: connTimeout,
	})
	if err != nil {
		return nil, err
	}
	result = &Result{
		URL:         mi.Url(),
		Title:       mi.Title(),
		Description: mi.Description(),
		Language:    mi.Language(),
		Patterns:    make([]string, 0, len(c.cfg.Patterns)),
	}
	body := mi.Body()
	for name, pattern := range c.cfg.Patterns {
		if bytes.Contains(body, []byte(pattern)) {
			result.Patterns = append(result.Patterns, name)
		}
	}
	if len(result.Patterns) < 1 {
		return nil, ErrNoMatch
	}
	return
}

func (c *Crawler) checkBucket(name []byte) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		if b := tx.Bucket(name); b == nil {
			return errors.New("no such bucket: " + string(name))
		}
		return nil
	})
}

// bytesLess return true iff a < b.
func bytesLess(a, b []byte) bool {
	return bytes.Compare(a, b) < 0
}
