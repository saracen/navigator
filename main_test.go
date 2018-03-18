package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type MainTestSuite struct {
	suite.Suite
}

func (suite *ServerTestSuite) TestBasicConfiguration() {
	navigator, interval, srv := configure([]string{"--url", "./.git#repository/testdata/charts", "--interval", "5m", "--http-addr", ":3333"})

	suite.NoError(navigator.UpdateRepositories())
	suite.Equal(5*time.Minute, interval, "interval not as expected")
	suite.Equal(":3333", srv.Addr, "http port not as expected")
}

func TestMainTestSuite(t *testing.T) {
	suite.Run(t, new(MainTestSuite))
}
