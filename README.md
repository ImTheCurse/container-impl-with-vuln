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

## 🏗️ 2) Install system deps + build (recommended)

Use the bootstrap helper to install `pkg-config` + `libseccomp` dev headers and then build:

```bash
go run ./tools/bootstrap
```

Useful options:

```bash
# only install dependencies
go run ./tools/bootstrap --install-only

# skip install and only build
go run ./tools/bootstrap --skip-install -o container-impl-with-vuln
```

## 🏗️ 3) Build the binary manually

```bash
go build -o container-impl-with-vuln .
```

## 🛠️ 4) Configure commands

Edit `container.json` (already included) with the commands you want to run inside the container.

Example:

```json
{
  "buildCommands": ["ls", "apt update", "touch ROOT_FS_FILE", "hostname", "ps"],
  "copy": ["container/. container/."],
  "workingDirectory": "/home/ubuntu/"
}

```

## 🚀 5) Run

```bash
sudo ./container-impl-with-vuln --blueprint container.json
```
