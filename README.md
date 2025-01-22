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

### Build & install
```bash
$ go install github.com/meehow/privtracker@latest
```

### Usage
```bash
# Runs on port 1337 by default.
$ ~/go/bin/privtracker
```

```bash
# Set PORT to 443 if you want to enable automatic TLS handling
$ PORT=443 ~/go/bin/privtracker
```

### Example Systemd service

This is an example of `/etc/systemd/system/privtracker.service` which can handle your privtracker service.

Remember to check directory names if you are going to use it.

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

### Docker Compose

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
