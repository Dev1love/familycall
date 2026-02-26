# Family Messenger — Deployment Guide

## Prerequisites

- A domain name (e.g., `family.example.com`)
- A VPS with Docker installed (min 512MB RAM, 1 CPU)
- DNS A record pointing your domain to the VPS IP

## Quick Start

1. SSH into your VPS:
   ```bash
   ssh user@your-vps-ip
   ```

2. Create a directory for the app:
   ```bash
   mkdir -p /opt/familycall && cd /opt/familycall
   ```

3. Create `docker-compose.yml`:
   ```yaml
   services:
     familycall:
       image: ghcr.io/dev1love/familycall:latest
       ports:
         - "443:443"
         - "3478:3478/udp"
       volumes:
         - ./data:/data
       environment:
         - DOMAIN=family.example.com
       restart: unless-stopped
   ```

4. Replace `family.example.com` with your actual domain.

5. Start the service:
   ```bash
   docker compose up -d
   ```

6. Open `https://family.example.com` in your browser.

## First Setup

1. **Register as organizer** — The first user to register becomes the family organizer (admin).
2. **Create invites** — Go to Contacts → Add Contact → share the invite link with family members.
3. **Family members join** — They open the invite link, choose a username, and they're in.

## Features

- **Text chat** — Direct and group messaging
- **Group calls** — Video/audio calls with up to 5 participants
- **Push notifications** — Get notified when offline
- **PWA** — Install as an app on phone/desktop

## Ports

| Port | Protocol | Purpose |
|------|----------|---------|
| 443 | TCP | HTTPS (web interface + API) |
| 3478 | UDP | TURN server (NAT traversal for calls) |

Make sure both ports are open in your VPS firewall.

## Data & Backup

All data is stored in the `./data` volume:
- `familycall.db` — SQLite database (users, chats, messages)
- `keys/` — JWT secret, VAPID keys, TURN credentials
- `certs/` — Let's Encrypt SSL certificates

### Manual backup:
```bash
docker compose stop
tar czf backup-$(date +%Y%m%d).tar.gz data/
docker compose start
```

### Restore:
```bash
docker compose stop
tar xzf backup-YYYYMMDD.tar.gz
docker compose start
```

The web interface also provides backup/restore via the organizer's settings.

## Updating

```bash
docker compose pull
docker compose up -d
```

## Troubleshooting

### SSL certificate not working
- Ensure DNS A record is pointing to the correct IP
- Port 443 must be accessible from the internet
- Wait a few minutes after first launch for Let's Encrypt to issue the certificate

### Calls not connecting
- Ensure port 3478/udp is open
- Check that the TURN server is reachable: the app auto-detects the server's public IP

### Push notifications not working
- Ensure HTTPS is working (push requires secure context)
- Check browser permissions for notifications

## Integration Test Checklist

- [ ] Register first user (organizer)
- [ ] Create an invite link
- [ ] Second user joins via invite
- [ ] Send direct message → verify delivery
- [ ] Create group chat with 3+ members
- [ ] Send group message → verify all members receive it
- [ ] Verify push notification for offline user
- [ ] Start group call from group chat
- [ ] Second user joins call → verify video
- [ ] Third user joins → verify mesh (3 video streams)
- [ ] Leave call → verify cleanup
- [ ] Backup via web interface
- [ ] Restore backup
