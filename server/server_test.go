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
	navigator *Server
	ts        *httptest.Server
}

func (suite *ServerTestSuite) SetupSuite() {
	suite.navigator = New(log.NewNopLogger())
	suite.NotNil(suite.navigator.Logger())

	suite.navigator.AddGitBackedRepository("../.git", []string{})
	suite.navigator.AddGitBackedRepository("../.git", []string{"repository/testdata/charts@test"})

	suite.ts = httptest.NewServer(MetricMiddleware(suite.navigator))
}

func (suite *ServerTestSuite) TearDownSuite() {
	suite.ts.Close()
}

func (suite *ServerTestSuite) TestServeHTTP() {
	if !suite.NoError(suite.navigator.UpdateRepositories()) {
		return
	}

	resp, err := http.Get(suite.ts.URL + "/test/index.yaml")
	if suite.NoError(err) && suite.Equal(http.StatusOK, resp.StatusCode) {
		_, err := ioutil.ReadAll(resp.Body)
		suite.NoError(err)
		suite.NoError(resp.Body.Close())
	}

	req, _ := http.NewRequest("GET", suite.ts.URL+"/test/index.yaml", nil)
	req.Header.Set("Accept-Encoding", "")
	resp, err = http.DefaultClient.Do(req)
	if suite.NoError(err) && suite.Equal(http.StatusOK, resp.StatusCode) {
		_, err := ioutil.ReadAll(resp.Body)
		suite.NoError(err)
		suite.NoError(resp.Body.Close())
	}

	index, err := suite.navigator.indexManager.Get("test")
	if !suite.NoError(err) {
		return
	}

	chart, err := index.Get("mychart", "0.1.0")
	if !suite.NoError(err) {
		return
	}

	resp, err = http.Get(suite.ts.URL + "/" + chart.URLs[0])
	if suite.NoError(err) && suite.Equal(http.StatusOK, resp.StatusCode) {
		_, err := ioutil.ReadAll(resp.Body)
		suite.NoError(err)
		suite.NoError(resp.Body.Close())
	}

	tests := map[string]int{
		"/unknown/index.yaml":          http.StatusNotFound,
		"/unknown/chart":               http.StatusNotFound,
		"/unknown/unknown/unknown":     http.StatusNotFound,
		"/" + chart.URLs[0] + "/error": http.StatusInternalServerError,
	}

	for path, code := range tests {
		resp, err = http.Get(suite.ts.URL + path)
		if suite.NoError(err, path) {
			suite.Equal(code, resp.StatusCode, path)
		}
	}
}

func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}
