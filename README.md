# wgmesh

![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)
![Go Version](https://img.shields.io/badge/Go-1.20-blue.svg)
![Release](https://img.shields.io/github/release/yourusername/wgmesh.svg)
![Build Status](https://github.com/yourusername/wgmesh/actions/workflows/build.yml/badge.svg)
![Go Report Card](https://goreportcard.com/badge/github.com/yourusername/wgmesh)

## 🚀 Overview

**wgmesh** is an intelligent, automated tool designed to streamline the deployment, configuration, and management of a WireGuard mesh network. Built with Go, wgmesh leverages modern technologies like `fsnotify` for real-time configuration monitoring and `wgctrl` for dynamic WireGuard interface management. Whether you're managing a small network or scaling up to a robust infrastructure, wgmesh ensures your mesh network remains consistent, secure, and effortlessly maintainable.

![wgmesh Logo](wgmesh.png)

## 📦 Features

- **Automated Deployment:** Simplify the setup of your WireGuard mesh with centralized YAML configurations.
- **Dynamic Configuration:** Real-time monitoring and application of configuration changes without downtime.
- **Systemd Integration:** Seamlessly manage wgmesh as a systemd service for reliability and ease of use.
- **Goreleaser Integration:** Streamlined building and signing of RPM packages for smooth distributions.
- **Comprehensive Logging:** Detailed logs for auditing and troubleshooting.
- **Extensible Architecture:** Easily extend wgmesh to fit unique networking requirements.

## 🛠️ Installation

### Prerequisites

- **Go:** Ensure you have Go installed. [Download Go](https://golang.org/dl/)
- **WireGuard:** Install WireGuard tools on your system. [WireGuard Installation Guide](https://www.wireguard.com/install/)

### Using Pre-built Binaries

1. **Download the Latest Release:**

   Visit the [Releases](https://github.com/yourusername/wgmesh/releases) page and download the appropriate RPM package for your system.

2. **Install the RPM Package:**

   ```bash
   sudo rpm -i wgmesh-<version>-x86_64.rpm
   ```

3. **Enable and Start the Service:**

   ```bash
   sudo systemctl enable wgmesh
   sudo systemctl start wgmesh
   ```

### Building from Source

1. **Clone the Repository:**

   ```bash
   git clone https://github.com/pilab-cloud/wgmesh.git
   cd wgmesh
   ```

2. **Build the Application:**

   ```bash
   go build -o wgmesh ./cmd/wgmesh
   ```

3. **Install the Application:**

   Move the binary to a directory in your `PATH`, such as `/usr/local/bin/`:

   ```bash
   sudo mv wgmesh /usr/local/bin/
   ```

4. **Configure Systemd Service:**

   Copy the provided systemd service file:

   ```bash
   sudo cp configs/pivirt-appliance/wgmesh.service /etc/systemd/system/
   sudo systemctl daemon-reload
   sudo systemctl enable wgmesh
   sudo systemctl start wgmesh
   ```

## 📄 Configuration

wgmesh uses a centralized YAML configuration file to define the mesh network topology and peer details.

### Sample `wgmesh.yaml`

```yaml
network_name: wg0
listen_port: 51820
private_key: YOUR_PRIVATE_KEY_HERE
peers:
  - name: peer1
    ip: 10.0.0.2
    public_key: abc123
    allowed_ips:
      - 10.0.0.2/32
  - name: peer2
    ip: 10.0.0.3
    public_key: def456
    allowed_ips:
      - 10.0.0.3/32
    endpoint: "192.168.1.100:51820"
    nat: true
```

### Configuration Fields

- **`network_name`**: Name of the WireGuard network interface.
- **`listen_port`**: Port WireGuard listens on.
- **`private_key`**: Your WireGuard private key.
- **`peers`**: List of peer configurations.
  - **`name`**: Unique identifier for the peer.
  - **`ip`**: Internal IP address assigned to the peer.
  - **`public_key`**: WireGuard public key of the peer.
  - **`allowed_ips`**: IPs/Subnets the peer is allowed to access.
  - **`endpoint`**: Public IP and port of the peer for NAT traversal.
  - **`nat`**: Boolean indicating if the peer is behind NAT.

## 📈 Usage

Once configured, wgmesh automatically manages your WireGuard mesh network based on the YAML file. Any changes to the configuration file are detected and applied in real-time.

### Common Commands

- **Start wgmesh Service:**

  ```bash
  sudo systemctl start wgmesh
  ```

- **Stop wgmesh Service:**

  ```bash
  sudo systemctl stop wgmesh
  ```

- **Restart wgmesh Service:**

  ```bash
  sudo systemctl restart wgmesh
  ```

- **Check Service Status:**

  ```bash
  sudo systemctl status wgmesh
  ```

## 🖥️ Systemd Integration

wgmesh is configured to run as a systemd service, ensuring it starts on boot and can be managed using standard systemctl commands.

### Example Systemd Service (`wgmesh.service`)

```ini
[Unit]
Description=WireGuard Mesh Network Manager
After=network.target

[Service]
ExecStart=/usr/local/bin/wgmesh /etc/wgmesh/wgmesh.yaml
Restart=always
User=root
Group=root

[Install]
WantedBy=multi-user.target
```

### Managing the Service

- **Enable on Boot:**

  ```bash
  sudo systemctl enable wgmesh
  ```

- **Disable on Boot:**

  ```bash
  sudo systemctl disable wgmesh
  ```

## 🔐 Security

wgmesh leverages robust logging and secure configuration practices to ensure your mesh network remains protected. Always safeguard your private keys and restrict access to configuration files.

## 🛠️ Development

### Setting Up the Development Environment

1. **Clone the Repository:**

   ```bash
   git clone https://github.com/pilab-cloud/wgmesh.git
   cd wgmesh
   ```

2. **Install Dependencies:**

   ```bash
   go mod download
   ```

3. **Run Tests:**

   ```bash
   go test ./...
   ```

### Running Locally

Start wgmesh with your configuration file:

```bash
wgmesh wgmesh.yaml
```

### Contributing

Contributions are welcome! Please follow these guidelines:

1. **Fork the Repository**
2. **Create a Feature Branch**

   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Commit Your Changes**

   ```bash
   git commit -m "Add some feature"
   ```

4. **Push to the Branch**

   ```bash
   git push origin feature/your-feature-name
   ```

5. **Open a Pull Request**

## 📜 License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## 📝 Documentation

Comprehensive documentation is available in the [docs](docs/specs/wireguard-mesh.md) directory.

## 💬 Support

For support, please open an issue on the [GitHub Issues](https://github.com/yourusername/wgmesh/issues) page or contact [gyula@pilab.hu](mailto:gyula@pilab.hu).

## 📄 Changelog

Detailed changes for each release are documented in the [CHANGELOG](CHANGELOG.md).

## 🎉 Acknowledgements

- [WireGuard](https://www.wireguard.com/)
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