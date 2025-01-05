package wgmesh

import (
	"net"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"gopkg.in/yaml.v2"
)

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

type WgMesh struct {
	Config       *Config
	YamlFilePath string
}

func NewWgMesh(yamlPath string) (*WgMesh, error) {
	m := &WgMesh{
		YamlFilePath: yamlPath,
	}

	config, err := m.LoadConfig(yamlPath)
	if err != nil {
		return nil, err
	}
	m.Config = config

	return m, nil
}

func (w *WgMesh) Start() {
	// Start the WireGuard tunnel
	err := w.StartTunnel()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start the WireGuard tunnel")
		return
	}

	// Start the file watcher
	w.startFileWatcher()
}

func (w *WgMesh) startFileWatcher() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize file watcher")
	}
	defer watcher.Close()

	// Add the YAML file to the watcher
	err = watcher.Add(w.YamlFilePath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to watch YAML file")
	}

	log.Info().Msg("File watcher started for YAML file: " + w.YamlFilePath)

	// Start watching for changes
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Info().Msg("Detected YAML file change")
				w.handleConfigChange()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
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

// WriteCurrentConfig writes the current configuration to a file. Useful for backups.
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

	// Generate a configuration for the new peer
	peerConfig := w.generatePeerConfig(peer)
	_ = peerConfig

	// Apply the configuration using wg (WireGuard command-line tool)
	// args := []string{
	// 	"set", w.Config.NetworkName,
	// 	"peer", peer.PublicKey,
	// 	"allowed-ips", strings.Join(peer.AllowedIPs, ","),
	// }
	// if peer.Endpoint != "" {
	// 	args = append(args, "endpoint", peer.Endpoint)
	// }
	// err := w.CommandRunner.Run("wg", args...)
	// if err != nil {
	// 	log.Error().Err(err).Msg("Failed to add peer: " + peer.Name)
	// 	return err
	// }

	// Optionally bring up the interface for the added peer

	if err := w.StartTunnel(); err != nil {
		log.Error().Err(err).Msg("Failed to start tunnel for peer: " + peer.Name)
		return err
	}

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
	log.Info().Msg("Updating peer: " + peer.Name)

	// Remove and re-add the peer to apply updates
	err := w.removePeer(peer)
	if err != nil {
		log.Error().Err(err).Msg("Failed to remove peer during update: " + peer.Name)
		return err
	}

	err = w.addPeer(peer)
	if err != nil {
		log.Error().Err(err).Msg("Failed to re-add peer during update: " + peer.Name)
		return err
	}

	log.Info().Msg("Successfully updated peer: " + peer.Name)
	return nil
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
	client, err := wgctrl.New()
	if err != nil {
		log.Error().Err(err).Msg("Failed to create WireGuard client")
		return err
	}
	defer client.Close()

	// Convert peers to WireGuard peer configurations
	var peers []wgtypes.PeerConfig
	for _, peer := range w.Config.Peers {
		publicKey, err := wgtypes.ParseKey(peer.PublicKey)
		if err != nil {
			log.Error().Err(err).Msgf("Invalid public key for peer %s", peer.Name)
			return err
		}

		peerConfig := wgtypes.PeerConfig{
			PublicKey: publicKey,
			AllowedIPs: func() []net.IPNet {
				var allowedIPs []net.IPNet
				for _, ip := range peer.AllowedIPs {
					_, ipNet, err := net.ParseCIDR(ip)
					if err != nil {
						log.Error().Err(err).Msgf("Invalid CIDR for peer %s: %s", peer.Name, ip)
						continue
					}
					allowedIPs = append(allowedIPs, *ipNet)
				}
				return allowedIPs
			}(),
			Endpoint: func() *net.UDPAddr {
				if peer.Endpoint != "" {
					addr, err := net.ResolveUDPAddr("udp", peer.Endpoint)
					if err != nil {
						log.Error().Err(err).Msgf("Invalid endpoint for peer %s", peer.Name)
						return nil
					}
					return addr
				}
				return nil
			}(),
			PersistentKeepaliveInterval: func() *time.Duration {
				if peer.NAT {
					interval := time.Duration(25) * time.Second
					return &interval
				}
				return nil
			}(),
		}
		peers = append(peers, peerConfig)
	}

	// Configure the WireGuard device
	privateKey, err := wgtypes.ParseKey(w.Config.PrivateKey)
	if err != nil {
		log.Error().Err(err).Msg("Invalid private key for the WireGuard device")
		return err
	}

	deviceConfig := wgtypes.Config{
		PrivateKey: &privateKey,
		ListenPort: func() *int {
			if w.Config.ListenPort != 0 {
				return &w.Config.ListenPort
			}
			return nil
		}(),
		ReplacePeers: true,
		Peers:        peers,
	}

	err = client.ConfigureDevice(w.Config.NetworkName, deviceConfig)
	if err != nil {
		log.Error().Err(err).Msg("Failed to configure WireGuard device")
		return err
	}

	log.Info().Msgf("WireGuard tunnel %s started successfully", w.Config.NetworkName)
	return nil
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
