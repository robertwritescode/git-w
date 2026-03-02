package output

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/suite"
)

type WriteSuite struct {
	suite.Suite
}

// TestWriteSuite uses suite.Run directly (not testutil.RunSuite) to avoid
// importing testutil, which depends on repo → output — that would create a
// circular dependency.
func TestWriteSuite(t *testing.T) {
	suite.Run(t, new(WriteSuite))
}

func (s *WriteSuite) TestWritef_FormatsOutput() {
	var buf bytes.Buffer
	Writef(&buf, "hello %s %d", "world", 42)
	s.Assert().Equal("hello world 42", buf.String())
}

func (s *WriteSuite) TestWritef_DiscardsWriteErrors() {
	// Writef must not panic when the writer returns an error.
	Writef(errWriter{}, "should not panic or return error")
}

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
