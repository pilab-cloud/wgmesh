package wgmesh

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"gopkg.in/yaml.v2"
)

type WireGuardClient interface {
	io.Closer
	Device(name string) (*wgtypes.Device, error)
	ConfigureDevice(name string, config wgtypes.Config) error
}

type Config struct {
	NetworkName string `yaml:"network_name"`
	Peers       []Peer `yaml:"peers"`
	ListenPort  int    `yaml:"listen_port"`
	PrivateKey  string `yaml:"private_key"`
}

type Peer struct {
	Name       string   `yaml:"name"`
	IP         string   `yaml:"ip"`
	PrivateKey string   `yaml:"private_key,omitempty"`
	PublicKey  string   `yaml:"public_key,omitempty"`
	AllowedIPs []string `yaml:"allowed_ips"`
	Endpoint   string   `yaml:"endpoint,omitempty"`
	Port       int      `yaml:"port,omitempty"`
	NAT        bool     `yaml:"nat,omitempty"`
}

type PeerState string

const (
	PeerStateUp    PeerState = "up"
	PeerStateDown  PeerState = "down"
	PeerStateError PeerState = "error"
)

type PeerStatus struct {
	Name      string    `yaml:"name"`
	State     PeerState `yaml:"status"` // "up", "down", "error"
	LastSeen  time.Time `yaml:"last_seen,omitempty"`
	Error     string    `yaml:"error,omitempty"`
	BytesSent uint64    `yaml:"bytes_sent"`
	BytesRecv uint64    `yaml:"bytes_recv"`
}

type MeshState string

const (
	MeshStateUp      MeshState = "up"
	MeshStateDown    MeshState = "down"
	MeshStatePartial MeshState = "partial"
)

type MeshStatus struct {
	NetworkName string                `yaml:"network_name"`
	Status      MeshState             `yaml:"status"` // "up", "partial", "down"
	Peers       map[string]PeerStatus `yaml:"peers"`
	LastUpdate  time.Time             `yaml:"last_update"`
}

type WgMesh struct {
	Config       *Config
	YamlFilePath string
	status       MeshStatus
	statusMu     sync.RWMutex
	Client       WireGuardClient
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

func NewWgMesh(yamlPath string) (*WgMesh, error) {
	client, err := wgctrl.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create wireguard client: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	m := &WgMesh{
		YamlFilePath: yamlPath,
		status: MeshStatus{
			Peers: make(map[string]PeerStatus),
		},
		Client: client,
		ctx:    ctx,
		cancel: cancel,
	}

	config, err := m.LoadConfig(yamlPath)
	if err != nil {
		cancel()
		client.Close()
		return nil, err
	}
	m.Config = config
	m.status.NetworkName = config.NetworkName

	return m, nil
}

// Close gracefully shuts down the WgMesh instance
func (w *WgMesh) Close() error {
	w.cancel()  // Signal all goroutines to stop
	w.wg.Wait() // Wait for all goroutines to finish
	return w.Client.Close()
}

func (w *WgMesh) GetStatus() MeshStatus {
	w.statusMu.RLock()
	defer w.statusMu.RUnlock()
	return w.status
}

func (w *WgMesh) updatePeerState(name string, state PeerState, err error) {
	w.statusMu.Lock()
	defer w.statusMu.Unlock()

	peerStatus := w.status.Peers[name]
	peerStatus.Name = name
	peerStatus.State = state
	if err != nil {
		peerStatus.Error = err.Error()
	} else {
		peerStatus.Error = ""
	}
	peerStatus.LastSeen = time.Now()
	w.status.Peers[name] = peerStatus

	// Update overall mesh status
	allUp := true
	allDown := true
	for _, p := range w.status.Peers {
		if p.State != "up" {
			allUp = false
		}
		if p.State != "down" {
			allDown = false
		}
	}

	if allUp {
		w.status.Status = "up"
	} else if allDown {
		w.status.Status = "down"
	} else {
		w.status.Status = "partial"
	}
	w.status.LastUpdate = time.Now()
}

func (w *WgMesh) handlePeerError(peer Peer, err error) {
	log.Error().
		Err(err).
		Str("peer", peer.Name).
		Msg("Failed to configure peer")

	w.updatePeerState(peer.Name, PeerStateError, err)
}

func (w *WgMesh) Start() error {
	// Start the WireGuard tunnel
	if err := w.StartTunnel(); err != nil {
		return fmt.Errorf("failed to start WireGuard tunnel: %w", err)
	}

	// Start the file watcher in a separate goroutine
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		if err := w.startFileWatcher(); err != nil {
			log.Error().Err(err).Msg("File watcher stopped with error")
		}
	}()

	return nil
}

func (w *WgMesh) startFileWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to initialize file watcher: %w", err)
	}
	defer watcher.Close()

	// Add the YAML file to the watcher
	if err := watcher.Add(w.YamlFilePath); err != nil {
		return fmt.Errorf("failed to watch YAML file: %w", err)
	}

	log.Info().Msg("File watcher started for YAML file: " + w.YamlFilePath)

	for {
		select {
		case <-w.ctx.Done():
			return nil
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Info().Msg("Detected YAML file change")
				w.handleConfigChange()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			log.Error().Err(err).Msg("Error watching file")
		}
	}
}

func (w *WgMesh) handleConfigChange() {
	// Backup the current YAML file
	err := w.backupConfig()
	if err != nil {
		log.Error().Err(err).Msg("Failed to backup configuration file")
		return
	}

	// Load the new configuration
	newConfig, err := w.LoadConfig(w.YamlFilePath)
	if err != nil {
		log.Error().Err(err).Msg("Failed to load updated configuration")
		return
	}

	// Compute mesh diffs
	addedPeers, removedPeers, updatedPeers := w.diffMesh(w.Config.Peers, newConfig.Peers)

	// Apply changes for added peers
	for _, peer := range addedPeers {
		log.Info().Msg("Adding new peer: " + peer.Name)
		err := w.addPeer(peer)
		if err != nil {
			log.Error().Err(err).Msg("Failed to add peer: " + peer.Name)
		}
	}

	// Apply changes for removed peers
	for _, peer := range removedPeers {
		log.Info().Msg("Removing peer: " + peer.Name)
		err := w.removePeer(peer)
		if err != nil {
			log.Error().Err(err).Msg("Failed to remove peer: " + peer.Name)
		}
	}

	// Apply changes for updated peers
	for _, peer := range updatedPeers {
		log.Info().Msg("Updating peer: " + peer.Name)
		err := w.updatePeer(peer)
		if err != nil {
			log.Error().Err(err).Msg("Failed to update peer: " + peer.Name)
		}
	}

	// Update the in-memory configuration
	w.Config = newConfig
}

func (w *WgMesh) backupConfig() error {
	backupPath := w.YamlFilePath + ".backup_" + time.Now().Format("20060102_150405")

	return w.WriteCurrentConfig(backupPath)
}

func (w *WgMesh) WriteCurrentConfig(path string) error {
	data, err := yaml.Marshal(w.Config)
	if err != nil {
		return err
	}

	// While it's containing sensitive data, it should be 600
	return os.WriteFile(path, data, 0o600)
}

func (w *WgMesh) addPeer(peer Peer) error {
	log.Info().Msg("Adding peer: " + peer.Name)

	peerConfig, err := w.createPeerConfig(peer)
	if err != nil {
		w.handlePeerError(peer, err)
		return err
	}

	// Configure the WireGuard interface with just this peer
	cfg := wgtypes.Config{
		Peers: []wgtypes.PeerConfig{peerConfig},
	}

	if err := w.Client.ConfigureDevice(w.Config.NetworkName, cfg); err != nil {
		w.handlePeerError(peer, err)
		return fmt.Errorf("failed to add peer %s: %w", peer.Name, err)
	}

	w.updatePeerState(peer.Name, "configuring", nil)
	log.Info().Msg("Successfully added peer: " + peer.Name)
	return nil
}

func (w *WgMesh) removePeer(peer Peer) error {
	log.Info().Msg("Removing peer: " + peer.Name)

	// Remove the peer using wg (WireGuard command-line tool)
	// args := []string{"set", w.Config.NetworkName, "peer", peer.PublicKey, "remove"}
	// err := w.CommandRunner.Run("wg", args...)
	// if err != nil {
	// 	log.Error().Err(err).Msg("Failed to remove peer: " + peer.Name)
	// 	return err
	// }

	// Optionally bring down the interface for the removed peer

	if err := w.StopTunnel(); err != nil {
		log.Error().Err(err).Msg("Failed to stop tunnel for peer: " + peer.Name)
		return err
	}

	log.Info().Msg("Successfully removed peer: " + peer.Name)
	return nil
}

func (w *WgMesh) updatePeer(peer Peer) error {
	// Remove the old peer first
	if err := w.removePeer(peer); err != nil {
		log.Warn().Err(err).Msgf("Failed to remove old peer %s before update", peer.Name)
	}

	// Add the peer with new configuration
	return w.addPeer(peer)
}

func (w *WgMesh) LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (w *WgMesh) diffMesh(oldPeers, newPeers []Peer) ([]Peer, []Peer, []Peer) {
	var addedPeers, removedPeers, updatedPeers []Peer
	oldPeerMap := make(map[string]Peer)
	newPeerMap := make(map[string]Peer)

	// Create a map of old peers
	for _, peer := range oldPeers {
		oldPeerMap[peer.Name] = peer
	}

	// Create a map of new peers
	for _, peer := range newPeers {
		newPeerMap[peer.Name] = peer
	}

	// Compare old and new peers
	for name, oldPeer := range oldPeerMap {
		newPeer, ok := newPeerMap[name]
		if !ok {
			// Peer is in old configuration but not in new configuration
			removedPeers = append(removedPeers, oldPeer)
		} else if !reflect.DeepEqual(oldPeer, newPeer) {
			// Peer is in both configurations but with changes
			updatedPeers = append(updatedPeers, newPeer)
		}
	}

	// Find added peers
	for name, newPeer := range newPeerMap {
		_, ok := oldPeerMap[name]
		if !ok {
			// Peer is in new configuration but not in old configuration
			addedPeers = append(addedPeers, newPeer)
		}
	}

	return addedPeers, removedPeers, updatedPeers
}

func (w *WgMesh) logChange(message string) {
	log.Info().Msg(message)
}

func getChanges(oldPeer, newPeer Peer) string {
	var changes []string

	if oldPeer.IP != newPeer.IP {
		changes = append(changes, "IP: "+oldPeer.IP+" -> "+newPeer.IP)
	}
	if oldPeer.PrivateKey != newPeer.PrivateKey {
		changes = append(changes, "PrivateKey: "+oldPeer.PrivateKey+" -> "+newPeer.PrivateKey)
	}
	if oldPeer.PublicKey != newPeer.PublicKey {
		changes = append(changes, "PublicKey: "+oldPeer.PublicKey+" -> "+newPeer.PublicKey)
	}
	if !reflect.DeepEqual(oldPeer.AllowedIPs, newPeer.AllowedIPs) {
		changes = append(changes, "AllowedIPs: "+strings.Join(oldPeer.AllowedIPs, ",")+" -> "+strings.Join(newPeer.AllowedIPs, ","))
	}
	if oldPeer.Endpoint != newPeer.Endpoint {
		changes = append(changes, "Endpoint: "+oldPeer.Endpoint+" -> "+newPeer.Endpoint)
	}
	if oldPeer.Port != newPeer.Port {
		changes = append(changes, "Port: "+strconv.Itoa(oldPeer.Port)+" -> "+strconv.Itoa(newPeer.Port))
	}
	if oldPeer.NAT != newPeer.NAT {
		changes = append(changes, "NAT: "+strconv.FormatBool(oldPeer.NAT)+" -> "+strconv.FormatBool(newPeer.NAT))
	}

	return strings.Join(changes, ", ")
}

func (w *WgMesh) StartTunnel() error {
	var lastErr error

	// Parse private key
	privateKey, err := wgtypes.ParseKey(w.Config.PrivateKey)
	if err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}

	// Create WireGuard configuration
	peerConfigs := make([]wgtypes.PeerConfig, 0, len(w.Config.Peers))
	for _, peer := range w.Config.Peers {
		peerConfig, err := w.createPeerConfig(peer)
		if err != nil {
			w.handlePeerError(peer, err)
			lastErr = err
			continue
		}
		peerConfigs = append(peerConfigs, peerConfig)
		w.updatePeerState(peer.Name, "configuring", nil)
	}

	// Configure the WireGuard interface
	cfg := wgtypes.Config{
		PrivateKey: &privateKey,
		ListenPort: &w.Config.ListenPort,
		Peers:      peerConfigs,
	}

	// Apply configuration
	if err := w.Client.ConfigureDevice(w.Config.NetworkName, cfg); err != nil {
		log.Error().Err(err).Msg("Failed to configure WireGuard device")
		// Mark all peers as error
		for _, peer := range w.Config.Peers {
			w.updatePeerState(peer.Name, "error", err)
		}
		return fmt.Errorf("failed to configure WireGuard device: %w", err)
	}

	// Start monitoring goroutine
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.monitorPeers()
	}()

	// If we had any peer errors but the device is running, return the last error
	// This allows the mesh to operate in a degraded state
	return lastErr
}

func (w *WgMesh) createPeerConfig(peer Peer) (wgtypes.PeerConfig, error) {
	pubKey, err := wgtypes.ParseKey(peer.PublicKey)
	if err != nil {
		return wgtypes.PeerConfig{}, fmt.Errorf("invalid public key for peer %s: %w", peer.Name, err)
	}

	var endpoint *net.UDPAddr
	if peer.Endpoint != "" {
		endpoint, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", peer.Endpoint, peer.Port))
		if err != nil {
			return wgtypes.PeerConfig{}, fmt.Errorf("invalid endpoint for peer %s: %w", peer.Name, err)
		}
	}

	allowedIPs := make([]net.IPNet, 0, len(peer.AllowedIPs))
	for _, ip := range peer.AllowedIPs {
		_, ipNet, err := net.ParseCIDR(ip)
		if err != nil {
			return wgtypes.PeerConfig{}, fmt.Errorf("invalid allowed IP for peer %s: %w", peer.Name, err)
		}
		allowedIPs = append(allowedIPs, *ipNet)
	}

	return wgtypes.PeerConfig{
		PublicKey:         pubKey,
		Endpoint:          endpoint,
		AllowedIPs:        allowedIPs,
		ReplaceAllowedIPs: true,
	}, nil
}

func (w *WgMesh) monitorPeers() {
	defer w.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			device, err := w.Client.Device(w.Config.NetworkName)
			if err != nil {
				log.Error().Err(err).Msg("Failed to get device status")
				continue
			}

			// Update status for all peers
			for _, peer := range device.Peers {
				peerName := w.getPeerNameByKey(peer.PublicKey.String())
				if peerName == "" {
					continue
				}

				w.statusMu.Lock()
				status := w.status.Peers[peerName]
				status.BytesRecv = uint64(peer.ReceiveBytes)
				status.BytesSent = uint64(peer.TransmitBytes)

				if !peer.LastHandshakeTime.IsZero() && time.Since(peer.LastHandshakeTime) < 3*time.Minute {
					status.State = "up"
					status.LastSeen = peer.LastHandshakeTime
				} else {
					status.State = "down"
				}

				w.status.Peers[peerName] = status
				w.statusMu.Unlock()
			}
		}
	}
}

func (w *WgMesh) getPeerNameByKey(publicKey string) string {
	for _, peer := range w.Config.Peers {
		if peer.PublicKey == publicKey {
			return peer.Name
		}
	}
	return ""
}

func (w *WgMesh) StopTunnel() error {
	client, err := wgctrl.New()
	if err != nil {
		log.Error().Err(err).Msg("Failed to create WireGuard client")
		return err
	}
	defer client.Close()

	deviceConfig := wgtypes.Config{
		ReplacePeers: true, // Clear all peers
		Peers:        nil,  // No peers
	}

	err = client.ConfigureDevice(w.Config.NetworkName, deviceConfig)
	if err != nil {
		log.Error().Err(err).Msg("Failed to clear WireGuard device configuration")
		return err
	}

	log.Info().Msgf("WireGuard tunnel %s stopped successfully", w.Config.NetworkName)
	return nil
}

func (w *WgMesh) RestartTunnel() error {
	// Restart the WireGuard tunnel
	err := w.StopTunnel()
	if err != nil {
		log.Error().Err(err).Msg("Failed to stop the WireGuard tunnel")
		return err
	}

	err = w.StartTunnel()
	if err != nil {
		log.Error().Err(err).Msg("Failed to start the WireGuard tunnel")
		return err
	}

	return nil
}

func (w *WgMesh) generatePeerConfig(peer Peer) string {
	// Generate the [Peer] section for WireGuard configuration
	var builder strings.Builder
	builder.WriteString("[Peer]\n")
	builder.WriteString("PublicKey = " + peer.PublicKey + "\n")
	if peer.Endpoint != "" {
		builder.WriteString("Endpoint = " + peer.Endpoint + "\n")
	}
	builder.WriteString("AllowedIPs = " + strings.Join(peer.AllowedIPs, ",") + "\n")
	if peer.Port != 0 {
		builder.WriteString("PersistentKeepalive = " + strconv.Itoa(peer.Port) + "\n")
	}
	return builder.String()
}
