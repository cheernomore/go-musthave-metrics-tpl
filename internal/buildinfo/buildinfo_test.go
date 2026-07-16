package buildinfo

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrint_WithValues(t *testing.T) {
	var buf bytes.Buffer
	Print(&buf, "v1.0.0", "2026-06-29", "abc123")

	assert.Equal(t,
		"Build version: v1.0.0\nBuild date: 2026-06-29\nBuild commit: abc123\n",
		buf.String(),
	)
}

func TestPrint_Empty(t *testing.T) {
	var buf bytes.Buffer
	Print(&buf, "", "", "")

	assert.Equal(t,
		"Build version: N/A\nBuild date: N/A\nBuild commit: N/A\n",
		buf.String(),
	)
}

func TestPrint_PartiallyEmpty(t *testing.T) {
	var buf bytes.Buffer
	Print(&buf, "v1.2.3", "", "deadbeef")

	assert.Equal(t,
		"Build version: v1.2.3\nBuild date: N/A\nBuild commit: deadbeef\n",
		buf.String(),
	)
}
