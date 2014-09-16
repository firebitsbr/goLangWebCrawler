GoLang Web Crawler
================

A Go lang web crawler searches through the top 1 million Alexa sites for content/ tags/ etc.

## Meteora Crawler

Crawl - an utility for fetching and parsing Alexa Top-1M sites in parallel mode. Alexa Top-1M gzipped CSV file: [click](https://www.dropbox.com/s/jckqlbf01bwn5g0/top-1m.csv.gz?dl=1). Do not unpack it.

### Installation

```
go get github.com/wakenn/goLangWebCrawler/crawl
wget https://www.dropbox.com/s/jckqlbf01bwn5g0/top-1m.csv.gz?dl=1 -O top-1m.csv.gz
```

### Cache

```
$ ./crawl cache -h
NAME:
   cache - caches the list of domains to crawl

USAGE:
   command cache [command options] [arguments...]

OPTIONS:
   --db 'sites.db'      a path to DB with cached domains
   --csv 'top-1m.csv.gz'    csv with domains list in format `rank,domain`
```

### Crawl

```
./crawl start -h
NAME:
   start - start crawling the cached sites and check against patterns

USAGE:
   command start [command options] [arguments...]

OPTIONS:
   --db 'sites.db'          a path to DB with cached domains
   --patterns, -p 'patterns.json'   a JSON map that specifies match patterns
   --out, -o 'crawled.txt'      a file to output the results to
   --level, -l '1'          how deeply the crawler should go (1-3)
   --jobs, -j '1'           maximum parallel jobs allowed (1-64)
   --skip, -s '0'           skips the defined number of top-positions
```

### Notes

The crawler may handle a huge amount of jobs, it's been tested with `-j 256`. The level function is not working as of now.
