package repository

import (
	"bytes"
	"testing"
	"time"

	"k8s.io/helm/pkg/proto/hapi/chart"

	"github.com/stretchr/testify/suite"
)

type IndexTestSuite struct {
	suite.Suite
	index *Index
}

func (suite *IndexTestSuite) SetupSuite() {
	suite.index = NewIndex()

	md := &chart.Metadata{
		Name:    "mychart",
		Version: "0.1.0",
	}
	suite.index.Add(md, []string{"foobar/mychart-0.1.0.tgz"}, time.Now())
}

func (suite *IndexTestSuite) TestWriteTo() {
	buf := new(bytes.Buffer)
	_, err := suite.index.WriteTo(buf)
	if suite.NoError(err, "error writing index file") {
		suite.NoError(err, suite.index.Unmarshal(buf.Bytes()), "error unmarshalling index file")
	}
}

func TestIndexTestSuite(t *testing.T) {
	suite.Run(t, new(IndexTestSuite))
}
