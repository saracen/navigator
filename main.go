package main

import (
	"flag"
	"fmt"
	"hash/fnv"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	"github.com/saracen/navigator/repository"
)

type repositoryURL struct {
	URL         string
	Directories []string
}

type repositoryURLs []repositoryURL

func (i *repositoryURLs) String() string {
	return ""
}

func (i *repositoryURLs) Set(value string) error {
	uri, err := url.Parse(value)
	if err != nil {
		return err
	}

	rurl := repositoryURL{}
	if len(uri.Fragment) > 0 {
		rurl.Directories = strings.Split(uri.Fragment, ",")
	}

	uri.Fragment = ""
	rurl.URL = uri.String()

	*i = append(*i, rurl)

	return nil
}

func main() {
	fs := flag.NewFlagSet("navigator", flag.ExitOnError)

	var (
		httpAddr = fs.String("http-addr", ":8081", "HTTP listen address")
		interval = fs.Duration("interval", time.Minute*5, "Poll interval for git repository updates")
		urls     repositoryURLs
	)

	fs.Var(&urls, "url", "Git repository to index")

	fs.Parse(os.Args[1:])

	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
		logger = level.NewFilter(logger, level.AllowInfo())
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	}

	index := repository.NewIndex()

	repositories := make(map[string]repository.Repository, len(urls))
	for _, url := range urls {
		hash := fnv.New32()
		hash.Write([]byte(url.URL))
		name := fmt.Sprintf("%x", hash.Sum(nil))

		level.Info(logger).Log("event", "add-repository", "repository", url.URL, "directories", strings.Join(url.Directories, ","))

		repositories[name] = repository.NewGitBackedRepository(logger, index, name, url.URL, url.Directories)
	}

	update := func() error {
		for _, repo := range repositories {
			err := repo.Update()
			if err != nil {
				return err
			}
		}

		return nil
	}

	// initial update
	if err := update(); err != nil {
		panic(err)
	}

	level.Info(logger).Log("event", "listening", "transport", "HTTP", "addr", *httpAddr)

	go func() {
		srv := &http.Server{
			Addr:         *httpAddr,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
			Handler:      &server{logger, index, repositories},
		}

		panic(srv.ListenAndServe())
	}()

	for range time.Tick(*interval) {
		update()
	}
}
