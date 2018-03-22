package server

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/suite"
)

type ServerTestSuite struct {
	suite.Suite
	srv *Server
}

func (suite *ServerTestSuite) SetupSuite() {
	suite.srv = New(log.NewNopLogger())
	suite.srv.AddGitBackedRepository("../.git", []string{"repository/testdata/charts@test"})
}

func (suite *ServerTestSuite) TestServeHTTP() {
	if !suite.NoError(suite.srv.UpdateRepositories()) {
		return
	}

	ts := httptest.NewServer(MetricMiddleware(suite.srv))
	defer ts.Close()

	res, err := http.Get(ts.URL + "/test/index.yaml")
	if suite.NoError(err) {
		_, err := ioutil.ReadAll(res.Body)
		suite.NoError(err)
		suite.NoError(res.Body.Close())
	}

	index, err := suite.srv.indexManager.Get("test")
	if !suite.NoError(err) {
		return
	}

	chart, err := index.Get("mychart", "0.1.0")
	if suite.NoError(err) {
		res, err := http.Get(ts.URL + "/" + chart.URLs[0])
		if suite.NoError(err) {
			_, err := ioutil.ReadAll(res.Body)
			suite.NoError(err)
			suite.NoError(res.Body.Close())
		}
	}
}

func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}
