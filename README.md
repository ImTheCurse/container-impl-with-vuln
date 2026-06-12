# 🧪 container-impl-with-vuln

Minimal container runner in Go using linux system containerization and isolation + vulnerability demo on it.

## ✅ Prerequisites

- 🐧 Linux
- 🧰 Go 1.22+
- 🔐 `sudo` access (required for `chroot` / hostname changes)

## 📦 1) Prepare the root filesystem

Create the `rootfs` directory if it does not exist, then extract the Ubuntu filesystem tarball:

```bash
sudo mkdir -p rootfs
sudo tar -xpf ubuntu-fs.tar -C rootfs
```

## 🏗️ 2) Build the binary

```bash
go build -o container-impl-with-vuln .
```

## 🛠️ 3) Configure commands

Edit `container.json` (already included) with the commands you want to run inside the container.

Example:

```json
{
  "buildCommands": ["hostname", "pwd", "ls -la"],
  "copy": ["*"],
  "workingDirectory": "/tmp"
}
```

## 🚀 4) Run

```bash
sudo ./container-impl-with-vuln --blueprint container.json
```
