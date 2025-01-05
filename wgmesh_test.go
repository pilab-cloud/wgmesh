package wgmesh_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/pilab-cloud/wgmesh"
)

// Helper to create temporary YAML files
func createTempConfig(content string) (string, func()) {
	tempFile, err := os.CreateTemp("", "wgmesh_test_*.yaml")
	if err != nil {
		panic(err)
	}
	_, _ = tempFile.Write([]byte(content))
	_ = tempFile.Close()
	return tempFile.Name(), func() { os.Remove(tempFile.Name()) }
}

// Error path test for invalid configuration
func TestInvalidConfig(t *testing.T) {
	configContent := `
network_name: wg0
peers:
  - name: peer1
    ip: 10.0.0.2
`
	configPath, cleanup := createTempConfig(configContent)
	defer cleanup()

	_, err := wgmesh.NewWgMesh(configPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}
