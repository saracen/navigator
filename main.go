package main

import (
	"flag"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/saracen/navigator/server"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
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

func configure(args []string) (*server.Server, time.Duration, *http.Server) {
	fs := flag.NewFlagSet("navigator", flag.ExitOnError)

	var (
		httpAddr = fs.String("http-addr", ":8080", "HTTP listen address")
		interval = fs.Duration("interval", time.Minute*5, "Poll interval for git repository updates")
		urls     repositoryURLs
	)

	fs.Var(&urls, "url", "Git repository to index")
	fs.Parse(args)

	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
		logger = level.NewFilter(logger, level.AllowInfo())
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	}

	navigator := server.New(logger)

	for _, url := range urls {
		navigator.AddGitBackedRepository(url.URL, url.Directories)
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", prometheus.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.Handle("/", server.MetricMiddleware(navigator))

	return navigator, *interval, &http.Server{
		Addr:         *httpAddr,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      mux,
	}
}

func main() {
	navigator, interval, srv := configure(os.Args[1:])

	// initial update
	if err := navigator.UpdateRepositories(); err != nil {
		panic(err)
	}

	level.Info(navigator.Logger()).Log("event", "listening", "transport", "HTTP", "addr", srv.Addr)

	go func() {
		panic(srv.ListenAndServe())
	}()

	for range time.Tick(interval) {
		navigator.UpdateRepositories()
	}
}
