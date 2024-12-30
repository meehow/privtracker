# [Private BitTorrent tracker for everyone](https://privtracker.com/)

PrivTracker allows to share torrent files just with your friends, nobody else.
Unlike public trackers, it shares peers only within a group which is using the same Announce URL.
It really works like a private tracker, but can be generated with one click of a button.

---
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

```toml
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
version: "3"
services:
    privtracker:
        build: https://github.com/meehow/privtracker.git
        restart: unless-stopped
        user: 1000:1000
        environment:
            - TZ=${TZ}
            - PORT=1337
        volumes:
            - config:/config
        ports:
            - 1337:1337

volumes:
    config:
```
