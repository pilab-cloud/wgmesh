# wgmesh

![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)
![Go Version](https://img.shields.io/badge/Go-1.20-blue.svg)
![Release](https://img.shields.io/github/release/pilab-cloud/wgmesh.svg)
![Build Status](https://github.com/pilab-cloud/wgmesh/actions/workflows/build.yml/badge.svg)
![Go Report Card](https://goreportcard.com/badge/github.com/pilab-cloud/wgmesh)

## 🚀 Overview

WireGuard Mesh Manager (`wgmesh`) is a robust tool for managing WireGuard mesh networks. It provides automatic peer configuration, real-time monitoring, and dynamic configuration updates through a YAML-based configuration system.

### Key Features

- 🔄 **Dynamic Configuration**: Hot-reload configuration changes without service restart
- 📊 **Real-time Monitoring**: Track peer status, connection health, and traffic statistics
- 🛡️ **Graceful Error Handling**: Continues operating in degraded state if some peers fail
- 🔒 **Secure by Default**: Proper key management and secure configuration handling
- 📝 **Detailed Logging**: Comprehensive logging of all network changes and events

## 📋 Requirements

- Linux system with WireGuard kernel module
- WireGuard tools package
- Proper permissions to configure network interfaces

## 🔧 Installation

### Using RPM Package (Recommended)

1. **Download the Latest Release:**
   Visit the [Releases](https://github.com/pilab-cloud/wgmesh/releases) page and download the appropriate RPM package for your system.

2. **Install the RPM Package:**
   ```bash
   sudo rpm -i wgmesh-<version>.rpm
   ```

### From Source

```bash
go install github.com/pilab-cloud/wgmesh/cmd/wgmesh@latest
```

## ⚙️ Configuration

Create a YAML configuration file at `/etc/wgmesh/wgmesh.yaml`:

```yaml
network_name: wg0
listen_port: 51820
private_key: <your-private-key>  # Base64-encoded WireGuard private key
peers:
  - name: peer1
    ip: 10.0.0.1/24
    public_key: <peer1-public-key>
    allowed_ips: ["10.0.0.0/24"]
    endpoint: "peer1.example.com:51820"
    persistent_keepalive: 25
    nat: true
```

### Configuration Options

- `network_name`: Name of the WireGuard interface
- `listen_port`: UDP port for WireGuard traffic
- `private_key`: Base64-encoded WireGuard private key
- `mtu`: Interface MTU
- `dns`: DNS servers
- `table`: Routing table

#### Peer Options

- `name`: Unique identifier for the peer
- `ip`: IP address for this peer in the mesh
- `public_key`: Peer's WireGuard public key
- `allowed_ips`: List of allowed IP ranges
- `endpoint`: Optional endpoint address (hostname:port)
- `persistent_keepalive`: Keepalive interval in seconds
- `nat`: Enable NAT traversal features

## 🚀 Usage

### Service Management

1. **Start the Service:**
   ```bash
   sudo systemctl start wgmesh
   ```

2. **Enable Auto-start:**
   ```bash
   sudo systemctl enable wgmesh
   ```

3. **Check Status:**
   ```bash
   sudo systemctl status wgmesh
   ```

### Monitoring

1. **View Service Logs:**
   ```bash
   sudo journalctl -u wgmesh -f
   ```

2. **Check Peer Status:**
   ```bash
   # View WireGuard interface status
   sudo wg show wg0
   
   # View detailed peer statistics
   sudo wg show wg0 dump
   ```

### Troubleshooting

Common issues and solutions:

1. **Permission Denied:**
   ```bash
   # Ensure proper permissions
   sudo setcap cap_net_admin=+ep /usr/local/bin/wgmesh
   ```

2. **Configuration Errors:**
   ```bash
   # Validate configuration
   sudo wgmesh --validate-config
   ```

3. **Connection Issues:**
   ```bash
   # Check firewall rules
   sudo firewall-cmd --list-ports
   
   # Add WireGuard port if needed
   sudo firewall-cmd --add-port=51820/udp --permanent
   sudo firewall-cmd --reload
   ```

## 🔍 Monitoring and Metrics

The service provides real-time monitoring through structured logging:

- **Peer Status:**
  - Connection state (up/down)
  - Last handshake time
  - Transfer statistics
  - Latency metrics

- **Configuration Changes:**
  - Peer additions/removals
  - Configuration updates
  - Error states

- **Performance Metrics:**
  - Bandwidth usage
  - Packet loss
  - Handshake latency

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Setup

```bash
# Install development dependencies
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run tests
go test -v ./...

# Run linter
golangci-lint run
```

## 📜 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🔗 Links

- [WireGuard Official Site](https://www.wireguard.com/)
- [Progressive Innovation LAB](https://pilab.hu)
- [Documentation](https://pilab.hu/docs/wgmesh)
- [Issue Tracker](https://github.com/pilab-cloud/wgmesh/issues)
- [GoReleaser](https://goreleaser.com/)
- [fsnotify](https://github.com/fsnotify/fsnotify)
- [wgctrl](https://github.com/wgctrl/wgctrl)

---

<p align="center">
Sponsored with ❤️ by
</p>
<p align="center">
    <a href="https://newpush.com" target="_blank">
    <img src="https://www.newpush.com/images/np_logo_blue_SVG.svg" width="128"/>
    </a><br>
    We focus on reliability, quality, and value.
</p>

---

<p style="padding-top: 2rem;" align="center">
Pioneering the future, together</p>

<p align="center">
<img src="https://pilab.hu/images/pi-logo-header.svg" alt="PiVirt Logo" width="100"></p>


