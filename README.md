# [Private BitTorrent tracker for everyone](https://privtracker.com/)

PrivTracker allows you to share torrent files only with your friends and nobody else.
Unlike public trackers, it shares peers only within a group using the same announce URL.
It really works like a private tracker, but can be generated with one click of a button.

## The Problem PrivTracker Solves

Sharing large files has always been difficult without compromising privacy or simplicity.
Centralized services require full uploads before sharing, often with account registration and fees for large files.
Hosting a file server demands technical knowledge, like opening firewall ports, and requires your computer to stay online.
PrivTracker solves this by enabling private torrent sharing within a trusted group.
Using UPnP, most torrent clients can automatically handle port opening, and only one person in the group needs an open port for everyone to download.
Unlike public trackers, PrivTracker shares peers' IPs only within the group and keeps files off public networks like DHT, ensuring privacy and efficient sharing.

### Build
```bash
git clone https://github.com/meehow/privtracker.git
cd privtracker
make build
```

You can also download pre-built binaries for your system from the [Releases page](https://github.com/meehow/privtracker/releases).

### Run PrivTracker without TLS

By default, PrivTracker will use port 1337.

```bash
./privtracker
```

### Run PrivTracker with automatic TLS / HTTPS

If you change the port to 443, PrivTracker will enable automatic TLS handling using [Let's Encrypt](https://letsencrypt.org/) to acquire a certificate.
Port 443 must be accessible from the internet, and you must have a domain name pointing to your server.

```bash
sudo setcap cap_net_bind_service=+ep privtracker # allow binding to ports below 1024
PORT=443 ./privtracker
```

### Example Systemd service

Below is an example of `/etc/systemd/system/privtracker.service`, which can be used to manage your PrivTracker service.

Ensure the directory paths are correct before using this configuration.

```ini
[Unit]
Description=privtracker
After=network.target
Requires=network.target

[Service]
Type=simple
User=privtracker
RestartSec=1s
Restart=on-failure
Environment=PORT=443
AmbientCapabilities=CAP_NET_BIND_SERVICE
WorkingDirectory=/home/privtracker/web
ExecStart=/home/privtracker/web/privtracker

[Install]
WantedBy=multi-user.target
```

### Example Docker Compose

```yaml
services:
    privtracker:
        image: meehow/privtracker
        restart: unless-stopped
        user: 1000:1000
        environment:
            - PORT=1337
        # volume is only needed if we listed on port 443
        volumes:
            - autocert:/.cache/golang-autocert
        ports:
            - 1337:1337

volumes:
    autocert:
```
