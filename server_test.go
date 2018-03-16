package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/saracen/navigator/repository"
	"github.com/stretchr/testify/suite"
)

type ServerTestSuite struct {
	suite.Suite
	srv *server
}

func (suite *ServerTestSuite) SetupSuite() {
	logger := log.NewNopLogger()
	index := repository.NewIndex()
	suite.srv = &server{
		logger: logger,
		index:  index,
		repos: map[string]repository.Repository{
			"test": repository.NewGitBackedRepository(logger, index, "test", "./.git", []string{"repository/testdata/charts"}),
		},
	}
}

func (suite *ServerTestSuite) TestServeHTTP() {
	if !suite.NoError(suite.srv.repos["test"].Update(), "error fetching index") {
		return
	}

	ts := httptest.NewServer(suite.srv)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/index.yaml")
	if suite.NoError(err, "error fetching index") {
		_, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		suite.NoError(err, "error reading index")
	}

	chart, err := suite.srv.index.Get("mychart", "0.1.0")
	if suite.NoError(err, "error checking for chart") {
		res, err := http.Get(ts.URL + "/" + chart.URLs[0])
		if suite.NoError(err, "error fetching chart") {
			_, err := ioutil.ReadAll(res.Body)
			res.Body.Close()
			suite.NoError(err, "error reading chart")
		}
	}
}

func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}
