# [Private BitTorrent tracker for everyone](https://privtracker.com/)

PrivTracker allows to share torrent files just with your friends, nobody else.
Unlike public trackers, it shares peers only within a group which is using the same Announce URL.
It really works like a private tracker, but can be generated with one click of a button. 

---
### Build
```bash
# Clone this repository.
$ git clone https://github.com/meehow/privtracker.git

# cd into the directory
$ cd privtracker

# Run go build
$ go build
```
### Usage
```bash
# Runs on port 1337 and redirects to privtracker.com by default.
$ ./privtracker
```
```bash
# Export PORT and DOMAIN variables to use custom values.
$ export PORT=12345 DOMAIN=customprivtracker.com; ./privtracker
```