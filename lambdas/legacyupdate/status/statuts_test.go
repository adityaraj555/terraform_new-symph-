package status

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewStatus(t *testing.T) {
	newstatus := New()
	assert.Equal(t, new(status), newstatus)
}
