package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type MainTestSuite struct {
	suite.Suite
}

func (suite *MainTestSuite) TestBasicConfiguration() {
	navigator, interval, srv := configure([]string{"--url", "./.git#repository/testdata/charts", "--interval", "5m", "--http-addr", ":3333"})

	suite.NoError(navigator.UpdateRepositories())
	suite.Equal(5*time.Minute, interval, "interval not as expected")
	suite.Equal(":3333", srv.Addr, "http port not as expected")
}

func (suite *MainTestSuite) TestHealthHandler() {
	_, _, srv := configure([]string{"--url", "./.git#repository/testdata/charts"})

	ts := httptest.NewServer(srv.Handler)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/health")
	suite.NoError(err)
	suite.Equal(http.StatusOK, res.StatusCode)
	suite.NoError(res.Body.Close())
}

func (suite *MainTestSuite) TestMetricsHandler() {
	_, _, srv := configure([]string{"--url", "./.git#repository/testdata/charts"})

	ts := httptest.NewServer(srv.Handler)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/metrics")
	suite.NoError(err)
	suite.Equal(http.StatusOK, res.StatusCode)
	suite.NoError(res.Body.Close())
}

func TestMainTestSuite(t *testing.T) {
	suite.Run(t, new(MainTestSuite))
}
