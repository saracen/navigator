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

func (suite *IndexTestSuite) TestAddVersions() {
	a := &chart.Metadata{
		Name:    "myversionedchart",
		Version: "0.1.0",
	}
	b := &chart.Metadata{
		Name:    "myversionedchart",
		Version: "0.1.1",
	}

	// new chart
	suite.True(suite.index.Add(a, []string{"foobar/myversionedchart-0.1.0.tgz"}, time.Now()))

	// same version, more recent
	suite.True(suite.index.Add(a, []string{"foobar/myversionedchart-0.1.0.tgz"}, time.Now().Add(time.Hour*1)))

	// same version, less recent
	suite.False(suite.index.Add(a, []string{"foobar/myversionedchart-0.1.0.tgz"}, time.Now().Add(time.Hour*-1)))

	// new version
	suite.True(suite.index.Add(b, []string{"foobar/myversionedchart-0.1.1.tgz"}, time.Now()))
}

func (suite *IndexTestSuite) TestCount() {
	packages, versions := suite.index.Count()

	suite.Equal(2, packages)
	suite.Equal(3, versions)
}

func (suite *IndexTestSuite) TestWriteTo() {
	buf := new(bytes.Buffer)
	_, err := suite.index.WriteTo(buf)
	if suite.NoError(err) {
		suite.NoError(err, suite.index.Unmarshal(buf.Bytes()))
	}

	// test cache
	_, err = suite.index.WriteTo(buf)
	suite.NoError(err)
}

func TestIndexTestSuite(t *testing.T) {
	suite.Run(t, new(IndexTestSuite))
}
