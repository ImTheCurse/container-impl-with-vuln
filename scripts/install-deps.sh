#!/usr/bin/env sh
set -eu

need_sudo=0
if [ "$(id -u)" -ne 0 ]; then
  need_sudo=1
fi

run_pkg_cmd() {
  if [ "$need_sudo" -eq 1 ]; then
    sudo "$@"
  else
    "$@"
  fi
}

if ! command -v pkg-config >/dev/null 2>&1; then
  echo "pkg-config not found; it will be installed."
fi

if ! command -v sh >/dev/null 2>&1; then
  echo "shell not found"
  exit 1
fi

if [ -r /etc/os-release ]; then
  . /etc/os-release
  os_id=${ID:-}
  os_like=${ID_LIKE:-}
else
  os_id=""
  os_like=""
fi

install_apt() {
  run_pkg_cmd apt-get update
  run_pkg_cmd apt-get install -y pkg-config libseccomp-dev
}

install_dnf() {
  run_pkg_cmd dnf install -y pkgconf-pkg-config libseccomp-devel
}

install_yum() {
  run_pkg_cmd yum install -y pkgconf-pkg-config libseccomp-devel
}

install_pacman() {
  run_pkg_cmd pacman -Sy --noconfirm --needed pkgconf libseccomp
}

install_apk() {
  run_pkg_cmd apk add --no-cache pkgconf libseccomp-dev
}

install_zypper() {
  run_pkg_cmd zypper --non-interactive install pkgconf-pkg-config libseccomp-devel
}

has_cmd() {
  command -v "$1" >/dev/null 2>&1
}

echo "Detected distro ID='${os_id}' ID_LIKE='${os_like}'"

case "$os_id" in
  ubuntu|debian)
    install_apt
    ;;
  fedora)
    install_dnf
    ;;
  centos|rhel|rocky|almalinux)
    if has_cmd dnf; then
      install_dnf
    else
      install_yum
    fi
    ;;
  arch|manjaro)
    install_pacman
    ;;
  alpine)
    install_apk
    ;;
  opensuse*|sles)
    install_zypper
    ;;
  *)
    if echo "$os_like" | grep -Eq "debian|ubuntu"; then
      install_apt
    elif echo "$os_like" | grep -Eq "rhel|fedora|centos|suse"; then
      if has_cmd dnf; then
        install_dnf
      elif has_cmd yum; then
        install_yum
      else
        install_zypper
      fi
    elif echo "$os_like" | grep -Eq "arch"; then
      install_pacman
    elif echo "$os_like" | grep -Eq "alpine"; then
      install_apk
    else
      echo "Unsupported Linux distribution. Install manually:"
      echo "  - pkg-config (or pkgconf)"
      echo "  - libseccomp development package (libseccomp-dev/libseccomp-devel)"
      exit 1
    fi
    ;;
esac

if pkg-config --exists libseccomp; then
  echo "libseccomp detected via pkg-config."
  pkg-config --modversion libseccomp || true
else
  echo "libseccomp still not available in pkg-config path."
  echo "You may need to set PKG_CONFIG_PATH manually."
  exit 1
fi
