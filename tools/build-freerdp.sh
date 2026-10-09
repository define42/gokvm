#!/usr/bin/env bash
set -euo pipefail

freerdp_tools=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
freerdp_version=3.32.1
freerdp_sha256=3021cf8848efbd0064664e187cb61e6101e4e74a4d8ffb495d9f7715e22d67de
freerdp_cache="$freerdp_tools/.cache"
freerdp_source="$freerdp_cache/freerdp-$freerdp_version"
freerdp_archive="$freerdp_source.tar.gz"
freerdp_build="$freerdp_cache/freerdp-build-$freerdp_version"
freerdp_install="$freerdp_tools/freerdp"

# Optional locally unpacked development packages keep this build usable on
# hosts where system-wide package installation is unavailable.
freerdp_deps="$freerdp_tools/.deps/usr"
if [[ -d "$freerdp_deps" ]]; then
    freerdp_arch=$(cc -print-multiarch)
    export PATH="$freerdp_deps/bin:$PATH"
    export LD_LIBRARY_PATH="$freerdp_deps/lib/$freerdp_arch${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"
    export PKG_CONFIG_PATH="$freerdp_deps/lib/$freerdp_arch/pkgconfig${PKG_CONFIG_PATH:+:$PKG_CONFIG_PATH}"
    export CMAKE_PREFIX_PATH="$freerdp_deps${CMAKE_PREFIX_PATH:+:$CMAKE_PREFIX_PATH}"
fi

for freerdp_command in cmake cc c++ make curl tar sha256sum pkg-config; do
    if ! command -v "$freerdp_command" >/dev/null 2>&1; then
        echo "Missing $freerdp_command; install the FreeRDP build dependencies listed in README.md." >&2
        exit 1
    fi
done
pkg-config --exists openh264 xcursor || {
    echo "OpenH264 and Xcursor development libraries are required; see README.md." >&2
    exit 1
}

mkdir -p "$freerdp_cache"
if [[ ! -f "$freerdp_archive" ]]; then
    curl --fail --location --retry 3 \
        "https://pub.freerdp.com/releases/freerdp-$freerdp_version.tar.gz" \
        --output "$freerdp_archive.part"
    mv -- "$freerdp_archive.part" "$freerdp_archive"
fi
printf '%s  %s\n' "$freerdp_sha256" "$freerdp_archive" | sha256sum --check
if [[ ! -f "$freerdp_source/CMakeLists.txt" ]]; then
    tar -xzf "$freerdp_archive" -C "$freerdp_cache"
fi

cmake -S "$freerdp_source" -B "$freerdp_build" -G 'Unix Makefiles' \
    -DCMAKE_BUILD_TYPE=Release \
    -DCMAKE_INSTALL_PREFIX="$freerdp_install" \
    -DCMAKE_INSTALL_LIBDIR=lib \
    -DWITH_CLIENT=ON -DWITH_X11=ON -DWITH_XCURSOR=ON \
    -DWITH_OPENH264=ON -DWITH_FFMPEG=OFF -DWITH_SWSCALE=OFF \
    -DWITH_PULSE=ON -DWITH_ALSA=ON \
    -DWITH_CLIENT_SDL=OFF -DWITH_SERVER=OFF -DWITH_SAMPLE=OFF \
    -DWITH_WAYLAND=OFF -DWITH_MANPAGES=OFF -DBUILD_TESTING=OFF \
    -DWITH_CUPS=OFF -DWITH_FUSE=OFF -DWITH_PCSC=OFF \
    -DWITH_SMARTCARD_PCSC=OFF -DWITH_SMARTCARD_EMULATE=OFF \
    -DWITH_PKCS11=OFF -DWITH_KRB5=OFF -DWITH_AAD=OFF \
    -DCHANNEL_URBDRC=OFF -DCHANNEL_SMARTCARD=OFF -DCHANNEL_PRINTER=OFF
cmake --build "$freerdp_build" --parallel "${FREERDP_JOBS:-4}"
cmake --install "$freerdp_build" --strip

freerdp_config=$("$freerdp_install/bin/xfreerdp" /buildconfig)
for freerdp_feature in WITH_GFX_H264 WITH_OPENH264 WITH_XCURSOR; do
    if [[ "$freerdp_config" != *"$freerdp_feature=ON"* &&
          "$freerdp_config" != *"$freerdp_feature=TRUE"* ]]; then
        echo "Built client is missing $freerdp_feature support." >&2
        exit 1
    fi
done
echo "Built $freerdp_install/bin/xfreerdp with OpenH264 and Xcursor support."
