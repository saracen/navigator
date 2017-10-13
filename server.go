package main

import (
	"fmt"
	"net"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	"github.com/saracen/navigator/repository"
)

type server struct {
	logger log.Logger
	index  *repository.Index
	repos  map[string]repository.Repository
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	begin := time.Now()

	err = s.serve(w, r)
	if err == nil {
		level.Info(s.logger).Log("event", "request", "client", host, "method", r.Method, "path", r.URL.Path, "took", time.Since(begin))
	} else {
		level.Error(s.logger).Log("event", "request", "client", host, "method", r.Method, "path", r.URL.Path, "took", time.Since(begin), "err", err)
	}
}

func (s *server) serve(w http.ResponseWriter, r *http.Request) (err error) {
	dir, file := path.Split(r.URL.Path)
	// serve index.yaml
	if file == "index.yaml" {
		_, err := s.index.WriteTo(w)
		return err
	}

	// serve packaged chart
	chart := strings.SplitN(strings.Trim(dir, "/"), "/", 2)
	if len(chart) != 2 {
		return fmt.Errorf("Invalid package name")
	}

	if repo, ok := s.repos[chart[0]]; ok {
		vcp, err := repo.ChartPackage(chart[1])
		if err != nil {
			return err
		}

		return vcp.Archive(w)
	}

	return fmt.Errorf("Repository not found")
}
