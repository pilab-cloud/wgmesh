package wgmesh_test

import (
	"os"
	"testing"
	"time"

	"github.com/pilab-cloud/wgmesh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// MockWireguardClient is a mock implementation of the WireGuard client
type MockWireguardClient struct {
	mock.Mock
}

func (m *MockWireguardClient) Device(name string) (*wgtypes.Device, error) {
	args := m.Called(name)
	if dev := args.Get(0); dev != nil {
		return dev.(*wgtypes.Device), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockWireguardClient) ConfigureDevice(name string, cfg wgtypes.Config) error {
	args := m.Called(name, cfg)
	return args.Error(0)
}

func (m *MockWireguardClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name     string
		yamlData string
		wantErr  bool
		validate func(*testing.T, *wgmesh.Config)
	}{
		{
			name: "valid config",
			yamlData: `
network_name: wg0
listen_port: 51820
private_key: ANVQk8Dtlqb9FwKITBjsNy7q4a1olz1kLQ8YeC/03U8=
peers:
  - name: peer1
    ip: 10.0.0.1/24
    public_key: a/iotNMJnrHngs6pBu/fFusGJW88oFYf3/U/hKCq3EA=
    allowed_ips: ["10.0.0.0/24"]
`,
			wantErr: false,
			validate: func(t *testing.T, cfg *wgmesh.Config) {
				assert.Equal(t, "wg0", cfg.NetworkName)
				assert.Equal(t, 51820, cfg.ListenPort)
				assert.Equal(t, "ANVQk8Dtlqb9FwKITBjsNy7q4a1olz1kLQ8YeC/03U8=", cfg.PrivateKey)
				require.Len(t, cfg.Peers, 1)
				assert.Equal(t, "peer1", cfg.Peers[0].Name)
				assert.Equal(t, "10.0.0.1/24", cfg.Peers[0].IP)
				assert.Equal(t, "a/iotNMJnrHngs6pBu/fFusGJW88oFYf3/U/hKCq3EA=", cfg.Peers[0].PublicKey)
				assert.Equal(t, []string{"10.0.0.0/24"}, cfg.Peers[0].AllowedIPs)
			},
		},
		{
			name: "invalid yaml",
			yamlData: `
invalid: [yaml
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpfile, err := os.CreateTemp("", "config*.yaml")
			require.NoError(t, err)
			defer os.Remove(tmpfile.Name())

			_, err = tmpfile.WriteString(tt.yamlData)
			require.NoError(t, err)
			require.NoError(t, tmpfile.Close())

			// Create WgMesh instance
			mesh, err := wgmesh.NewWgMesh(tmpfile.Name())
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, mesh.Config)
			}
		})
	}
}

func TestFileWatcher(t *testing.T) {
	t.Skip("Skipping integration test")

	// Create temporary config file
	tmpfile, err := os.CreateTemp("", "config*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	initialConfig := `
network_name: wg0
listen_port: 51820
private_key: ANVQk8Dtlqb9FwKITBjsNy7q4a1olz1kLQ8YeC/03U8=
peers: []
`
	_, err = tmpfile.WriteString(initialConfig)
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	// Create WgMesh instance
	mesh, err := wgmesh.NewWgMesh(tmpfile.Name())
	require.NoError(t, err)

	// Start mesh
	err = mesh.Start()
	require.NoError(t, err)

	// Wait for file watcher to start
	time.Sleep(100 * time.Millisecond)

	// Modify config file
	newConfig := `
network_name: wg0
listen_port: 51820
private_key: ANVQk8Dtlqb9FwKITBjsNy7q4a1olz1kLQ8YeC/03U8=
peers:
  - name: peer1
    ip: 10.0.0.1/24
    public_key: a/iotNMJnrHngs6pBu/fFusGJW88oFYf3/U/hKCq3EA=
    allowed_ips: ["10.0.0.0/24"]
`
	err = os.WriteFile(tmpfile.Name(), []byte(newConfig), 0o644)
	require.NoError(t, err)

	// Wait for config change to be detected
	time.Sleep(100 * time.Millisecond)

	// Verify config was updated
	assert.Len(t, mesh.Config.Peers, 1)
	assert.Equal(t, "peer1", mesh.Config.Peers[0].Name)

	// Cleanup
	mesh.Close()
}

func TestPeerMonitoring(t *testing.T) {
	t.Skip("Skipping integration test")
	// Create mock WireGuard client
	mockClient := &MockWireguardClient{}

	// Create temporary config file
	tmpfile, err := os.CreateTemp("", "config*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	config := `
network_name: wg0
listen_port: 51820
private_key: ANVQk8Dtlqb9FwKITBjsNy7q4a1olz1kLQ8YeC/03U8=
peers:
  - name: peer1
    ip: 10.0.0.1/24
    public_key: a/iotNMJnrHngs6pBu/fFusGJW88oFYf3/U/hKCq3EA=
    allowed_ips: ["10.0.0.0/24"]
`
	_, err = tmpfile.WriteString(config)
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	// Create WgMesh instance
	mesh, err := wgmesh.NewWgMesh(tmpfile.Name())
	require.NoError(t, err)

	// Replace client with mock
	mesh.Client = mockClient

	// Mock device response
	mockClient.On("Device", "wg0").Return(&wgtypes.Device{
		Peers: []wgtypes.Peer{
			{
				PublicKey:         wgtypes.Key{}, // Replace with actual key
				LastHandshakeTime: time.Now(),
				ReceiveBytes:      1000,
				TransmitBytes:     2000,
			},
		},
	}, nil)

	// Start monitoring
	mesh.Start()

	// Wait for status update
	time.Sleep(100 * time.Millisecond)

	// Verify status
	status := mesh.GetStatus()
	peer := status.Peers["peer1"]
	assert.Equal(t, "up", peer.State)
	assert.Equal(t, uint64(1000), peer.BytesRecv)
	assert.Equal(t, uint64(2000), peer.BytesSent)

	// Cleanup
	mesh.Close()
	mockClient.AssertExpectations(t)
}
