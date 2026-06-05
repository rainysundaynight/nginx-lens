package upstream

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckTCPLocalhost(t *testing.T) {
	// Порт, который скорее всего закрыт — проверяем что функция не паникует
	result := CheckTCP("127.0.0.1:59999", 0.5, 1)
	assert.IsType(t, false, result)
}

func TestResolveAddressIP(t *testing.T) {
	result := ResolveAddress("127.0.0.1:8080", false, 300, "")
	assert.Equal(t, []string{"127.0.0.1:8080"}, result)
}

func TestSplitHostPort(t *testing.T) {
	host, port, err := splitHostPort("example.com:8080")
	assert.NoError(t, err)
	assert.Equal(t, "example.com", host)
	assert.Equal(t, "8080", port)
}
