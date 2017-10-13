package main

import (
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

	code, err := s.serve(w, r)
	if err == nil {
		if code != http.StatusOK {
			w.WriteHeader(code)
		}
		level.Info(s.logger).Log("event", "request", "client", host, "method", r.Method, "path", r.URL.Path, "took", time.Since(begin))
	} else {
		http.Error(w, err.Error(), code)
		level.Error(s.logger).Log("event", "request", "client", host, "method", r.Method, "path", r.URL.Path, "took", time.Since(begin), "err", err)
	}
}

func (s *server) serve(w http.ResponseWriter, r *http.Request) (code int, err error) {
	dir, file := path.Split(r.URL.Path)
	// serve index.yaml
	if file == "index.yaml" {
		_, err := s.index.WriteTo(w)
		if err != nil {
			return http.StatusInternalServerError, err
		}
		return http.StatusOK, nil
	}

	// serve packaged chart
	chart := strings.SplitN(strings.Trim(dir, "/"), "/", 2)
	if len(chart) != 2 {
		return http.StatusNotFound, repository.ErrInvalidPackageName
	}

	if repo, ok := s.repos[chart[0]]; ok {
		vcp, err := repo.ChartPackage(chart[1])
		if err != nil {
			return http.StatusInternalServerError, err
		}

		if err = vcp.Archive(w); err != nil {
			return http.StatusInternalServerError, err
		}

		return http.StatusOK, nil
	}

	return http.StatusNotFound, repository.ErrRepositoryNotFound
}
