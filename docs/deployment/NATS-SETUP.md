# NATS JetStream Setup Guide for Clinic Queue

Step-by-step guide for deploying and configuring **NATS JetStream** on a Debian VPS using **Podman** specifically for the **Clinic Queue** application.

---

## Infrastructure Overview

- **Service Name:** NATS JetStream Engine (`nats-clinic`)
- **Host Port:** `4222` (TCP)
- **Disk Storage:** `/opt/nats/clinic/data` (SSD Persistence)
- **Target Consumer:** Go Backend (e.g., on Northflank)

---

## 1. Firewall Configuration

Ensure the following inbound rules are configured in your cloud firewall (e.g., Vultr Firewall):

1. **SSH:** Port `22` (TCP, Source: `0.0.0.0/0`)
2. **NATS Clinic:** Port `4222` (TCP, Source: `0.0.0.0/0`)
3. Ensure the Firewall Group is attached to your VPS instance.

---

## 2. Installing & Running NATS Clinic

### Step A: Create Directories and Configuration File
Execute the following commands on the VPS terminal:

```bash
mkdir -p /opt/nats/clinic/data /opt/nats/clinic/conf

cat << 'EOF' > /opt/nats/clinic/conf/nats.conf
port: 4222

# JetStream Storage Settings
jetstream {
  store_dir: "/data"
  max_mem: 128MB
  max_file: 3GB
}

# User Authentication (Replace with your actual credentials)
authorization {
  user: "<NATS_USER>"
  password: "<NATS_STRONG_PASSWORD>"
  timeout: 5
}
EOF
```

### Step B: Launch NATS Container using Podman
```bash
podman run -d --name nats-clinic \
  --restart always \
  -p 4222:4222 \
  -v /opt/nats/clinic/data:/data:Z \
  -v /opt/nats/clinic/conf/nats.conf:/etc/nats/nats.conf:Z \
  docker.io/library/nats:latest \
  -c /etc/nats/nats.conf
```

---

## 3. Backend Configuration

In your backend environment variables (e.g., on Northflank), configure the connection string using your credentials:

```env
NATS_URL=nats://<NATS_USER>:<NATS_STRONG_PASSWORD>@<VPS_IP_OR_DOMAIN>:4222
```

---

## 4. Maintenance Commands

### Check Container Status
```bash
podman ps
```

### Inspect Container Logs
```bash
podman logs -f nats-clinic
```

### Restart Container
```bash
podman restart nats-clinic
```
