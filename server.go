package main

import (
	"fmt"
	"hash/fnv"
	"net"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	"github.com/saracen/navigator/repository"
)

// Server is the navigator server that handles HTTP requests for charts
type Server struct {
	logger            log.Logger
	indexManager      *repository.IndexManager
	dependencyManager *repository.DependencyManager
	repos             map[string]repository.Repository
}

// NewServer returns a new server
func NewServer(logger log.Logger) *Server {
	indexManager := repository.NewIndexManager()
	return &Server{
		logger:            logger,
		indexManager:      indexManager,
		dependencyManager: repository.NewDependencyManager(logger, indexManager),
		repos:             make(map[string]repository.Repository),
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

func (s *Server) serve(w http.ResponseWriter, r *http.Request) (code int, err error) {
	indexName, file := path.Split(r.URL.Path)
	indexName = strings.Trim(indexName, "/")

	// serve index.yaml
	if file == "index.yaml" {
		index, err := s.indexManager.Get(indexName)
		if err != nil {
			return http.StatusNotFound, repository.ErrIndexNotFound
		}

		if _, err = index.WriteTo(w); err != nil {
			return http.StatusInternalServerError, err
		}
		return http.StatusOK, nil
	}

	// serve packaged chart
	chart := strings.SplitN(indexName, "/", 2)
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

// AddGitBackedRepository adds a new git backed repository to the server
func (s *Server) AddGitBackedRepository(url string, directories []string) {
	hash := fnv.New32()
	hash.Write([]byte(url))
	name := fmt.Sprintf("%x", hash.Sum(nil))

	level.Info(s.logger).Log("event", "add-repository", "repository", url, "directories", strings.Join(directories, ","))

	var indexDirectories []repository.IndexDirectory
	for _, directory := range directories {
		di := strings.SplitN(directory, "@", 2)

		indexName := "default"
		if len(di) == 2 {
			indexName = di[1]
		}

		s.indexManager.Create(indexName)
		indexDirectories = append(indexDirectories, repository.IndexDirectory{Name: di[0], IndexName: indexName})
	}

	s.repos[name] = repository.NewGitBackedRepository(s.logger, s.dependencyManager, name, url, indexDirectories)
}

// UpdateRepositories fetches changes from the source repositories and indexes new updates
func (s *Server) UpdateRepositories() error {
	for _, repo := range s.repos {
		err := repo.Update()
		if err != nil {
			return err
		}
	}
	return nil
}
