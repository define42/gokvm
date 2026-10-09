# gokvm [![CI](https://github.com/bobuhiro11/gokvm/actions/workflows/ci.yml/badge.svg)](https://github.com/bobuhiro11/gokvm/actions/workflows/ci.yml) [![Coverage Status](https://coveralls.io/repos/github/bobuhiro11/gokvm/badge.svg?branch=main)](https://coveralls.io/github/bobuhiro11/gokvm?branch=main) [![code lines](https://sloc.xyz/github/bobuhiro11/gokvm?category=code)](https://sloc.xyz/github/bobuhiro11/gokvm?category=code) [![Go Reference](https://pkg.go.dev/badge/github.com/bobuhiro11/gokvm.svg)](https://pkg.go.dev/github.com/bobuhiro11/gokvm) [![Go Report Card](https://goreportcard.com/badge/github.com/bobuhiro11/gokvm)](https://goreportcard.com/report/github.com/bobuhiro11/gokvm)


gokvm is a hypervisor that uses KVM as an acceleration.
It is implemented completely in the Go language.
With **only 1.5k lines of code**, it can **boot Linux 5.10**, the latest version at the time, without any modifications
(see [v0.0.1](https://github.com/bobuhiro11/gokvm/releases/tag/v0.0.1)).
It includes naive and simple device emulation for serial console, virtio-net, and virtio-blk.
The execution environment is limited to the x86-64 Linux environment.
This should be useful for those who are interested in how to use KVM from userland.
The latest version supports the following features:

- [x] kvm acceleration
- [x] multi processors
- [x] serial console
- [x] virtio-net (virtio 1.0, modern PCI transport)
- [x] virtio-blk (virtio 1.0, modern PCI transport)
- [x] virtio-gpu (virtio 1.0, 2D; frames written to PNG via `-g`)
- [x] VNC server for virtio-gpu with keyboard and mouse input (`-vnc`)
- [x] Built-in TLS RDP console with keyboard and mouse input (`-rdp`)
- [x] Optional OpenH264 AVC420 compression for RDP (`-rdp-h264`)
- [x] RDP initial resolution and live single-monitor resizing with virtio-gpu hotplug
- [x] Virtio-snd playback through the built-in RDP server (`-audio rdp`)
- [x] Built-in user-mode networking with DHCP, DNS, and outbound TCP/UDP (`-net user`)
- [x] PVH Boot Protocol
- [x] ISO boot via the El Torito boot catalog (no SeaBIOS/UEFI firmware required)

**This is an experimental project, so please do not use it in production.**

![demo](https://raw.githubusercontent.com/bobuhiro11/gokvm/main/demo.gif)

## CLI

Extract the latest release from [the Github Release tab](https://github.com/bobuhiro11/gokvm/releases) and run it.
Before running, make sure /dev/kvm exists.
You can use existing bzImage and initrd, or you can create them using the Makefile of this project.

```bash
tar zxvf gokvm*.tar.gz
./gokvm boot -k ./bzImage -i ./initrd  # To exit, press Ctrl-a x.
./gokvm boot -k ./bzImage -i ./initrd -vnc :5900  # Enable virtio-gpu over VNC.
./gokvm boot -iso ./TinyCore-current.iso -vnc :5900  # Boot kernel/initrd from an ISO.
./gokvm boot -iso ./TinyCore-current.iso -rdp 127.0.0.1:3389 -m 512M  # RDP desktop.
./gokvm boot -iso http://www.tinycorelinux.net/17.x/x86/release/TinyCore-current.iso -vnc :5900
```

The bundled kernel config enables virtio-gpu and framebuffer console support, so
VNC shows the Linux framebuffer console once the guest probes the GPU. If you
use a different kernel, make sure it has `CONFIG_DRM_VIRTIO_GPU`,
`CONFIG_DRM_FBDEV_EMULATION`, and `CONFIG_FRAMEBUFFER_CONSOLE` built in.

ISO boot support reads the El Torito boot catalog when present, uses the boot
image location to find common syslinux/isolinux or GRUB configs, then loads the
Linux kernel/initrd through gokvm's direct Linux loader. It does not emulate the
BIOS/UEFI bootloader code, so no SeaBIOS firmware is required. The raw ISO is
attached to the guest as a read-only virtio-blk device so the booted kernel can
mount its live media.

When booting a TinyCore ISO with `-vnc` or `-rdp`, gokvm injects an autostart overlay into
the initrd so the guest brings up the FLWM desktop (Xvesa) on the remote display
instead of a text login. Both servers forward keyboard and mouse input back
to the guest. Try it with `make tinycore`, which builds gokvm and boots
`TinyCore-current.iso` on `127.0.0.1:5900`.

Slax can boot directly from its ISO with its normal graphical startup:

```bash
./gokvm boot -iso ./slax.iso -m 2G -rdp 127.0.0.1:3390
```

The Slackware-based Slax image with Linux 6.1.38 has been tested through to the
desktop. Keep the ISO's default boot parameters; no extracted kernel, custom
initrd, or driver blacklist is needed. For AVC420, build with the `openh264` tag
and add `-rdp-h264` as described below.

### Networking

Add `-net user` for networking inside the gokvm process. It needs no TAP device,
root privileges, host firewall rules, or helper daemon:

```bash
./gokvm boot -iso ./slax.iso -m 2G -net user \
  -rdp 127.0.0.1:3390 -rdp-h264
```

Use at least 2 GB for browser workloads. A 1 GB Slax guest can reach the desktop,
but complex websites can fill both RAM and Slax's compressed swap, making the
whole desktop unresponsive while RDP remains connected. Check `free -m` and
`vmstat 1` inside the guest when this happens; additional tabs may need more RAM.

Omit `-rdp-h264` when using the standard build. Slax requests an address
automatically: the first guest receives `10.0.2.15/24`, with gateway and DNS
server `10.0.2.2`. DNS uses the host's configured resolvers. Guest TCP/UDP
connections to `10.0.2.2` reach host loopback services, except the built-in DNS
and DHCP ports.

This backend supports outbound IPv4 TCP/UDP. Incoming port forwarding and IPv6
are not implemented. External ICMP/ping is not forwarded; check connectivity
with a web browser or `wget https://github.com` in the guest. The network stack
is built in Go and works with `CGO_ENABLED=0`; only optional OpenH264 requires cgo.

The tested Slax image lacks its default CA certificate bundle. If its HTTPS
tools report certificate errors, run these commands inside Slax, then retry:

```bash
update-ca-certificates
ln -s certs/ca-certificates.crt /etc/ssl/cert.pem
```

Networking remains disabled by default (`-net none`). The existing `-t tap0`
option attaches a host-managed TAP interface; it cannot be combined with `-net`.

### RDP console

RDP is implemented in Go inside gokvm; no xrdp daemon or guest RDP server is
needed. Build and start the bundled TinyCore desktop:

```bash
go build -o gokvm .
./gokvm boot -iso ./TinyCore-current.iso -rdp 127.0.0.1:3389 -m 512M
```

Connect using FreeRDP (the executable may be named `xfreerdp3` on your system):

```bash
xfreerdp /v:127.0.0.1:3389 /sec:tls /u:console /p:console /cert:ignore
```

The example accepts the temporary self-signed certificate for local testing.
To supply a persistent certificate, add `-rdp-cert console.crt -rdp-key console.key`
to the gokvm command; both files must be PEM encoded. In clients with a security
selector, choose TLS. NLA/CredSSP is not supported in this initial implementation.

**The console has no user authentication.** The example credentials are dummy
values and are not checked. TLS encrypts the connection; it does not restrict who
can control the VM. Keep the listener on loopback and use an authenticated SSH
tunnel for access from another machine.

The RDP session uses the client's requested desktop size and supports live
single-monitor resizing. By default it sends uncompressed bitmap updates with
16-, 24-, or 32-bit color. The optional OpenH264 build adds compressed AVC420 graphics.
It includes the same serial/VGA/VESA fallbacks as VNC. Both `-rdp` and `-vnc` can
be supplied to view the same guest at once; connected viewers share its keyboard
and mouse. Optional audio playback uses `-audio rdp`. Clipboard, microphone input,
drive redirection, and multiple monitors are not implemented. FreeRDP 3.32.1 has
been tested with TinyCore and Slax desktop rendering, pointer positioning, menu
interaction, and application launch; Microsoft Remote Desktop has not yet been validated.

In TinyCore, right-click the desktop to open its menu. The application dock
appears along the bottom of the screen.

#### Screen resolution

Set an initial size with `/size` and enable window-driven resizing with
`/dynamic-resolution` in FreeRDP:

```bash
xfreerdp /v:127.0.0.1:3390 /sec:tls /u:console /p \
  /cert:ignore /size:1280x720 /dynamic-resolution /gfx:AVC420
```

Omit `/gfx:AVC420` for bitmap clients. Both paths resize without reconnecting;
audio and input remain on the same connection. Rapid window changes are combined
over 150 ms. Each dimension must be 200–4096 pixels, with at most 4096×2160 total
pixels; odd dimensions round down to even pixels for AVC420. Unsupported layouts
leave the current size active. Only one monitor is supported.

The first connected RDP viewer controls the shared GPU's preferred mode. Other
viewers scale that desktop to their own requested sizes. When the controlling
viewer disconnects, the next connected viewer takes over. Mouse coordinates
follow the current framebuffer size throughout a mode change.

The guest must use `virtio_gpu` and apply its DRM/RandR hotplug notification to
change the actual desktop resolution. A desktop that does not apply the mode
continues rendering at its previous size, scaled into the RDP window. The fixed
1024×768 VESA fallback used by older guests also scales; it cannot change modes.
Linux may round a preferred mode's width to an 8-pixel boundary.

For X11 desktops without automatic RandR mode handling, such as Fluxbox, copy
[`scripts/gokvm-resize`](scripts/gokvm-resize) into the guest and run it from a
terminal in the graphical session:

```bash
sh ./gokvm-resize --once  # Apply the current preferred mode once.
sh ./gokvm-resize &       # Follow future changes (checks once per second).
```

The helper requires `xrandr` and handles one connected `Virtual-*` output. To
start it with Slax's desktop, put the script at `~/.local/bin/gokvm-resize` and
add `sh "$HOME/.local/bin/gokvm-resize" &` before the line that starts Fluxbox in
`~/.fluxbox/startup`. Desktops that already apply preferred modes need no helper.

#### OpenH264 graphics

Build with the `openh264` tag to enable H.264/AVC420 over the RDP graphics
pipeline. This requires cgo, a C compiler, `pkg-config`, and the
[OpenH264](https://github.com/cisco/openh264) development library. On Debian or
Ubuntu, the build packages are `build-essential pkg-config libopenh264-dev`.

```bash
CGO_ENABLED=1 go build -tags openh264 -o gokvm .
./gokvm boot -iso ./TinyCore-current.iso -rdp 127.0.0.1:3389 -rdp-h264 -m 512M
```

Use a FreeRDP build with H.264 decoding enabled:

```bash
xfreerdp /v:127.0.0.1:3389 /sec:tls /u:console /p:console /cert:ignore /gfx:AVC420
```

Check `xfreerdp /buildconfig` for `WITH_GFX_H264=ON`. A build with
`WITH_GFX_H264=OFF` rejects `/gfx:AVC420` during command-line parsing, even if
its help lists that option. Installing the OpenH264 library alone does not
enable codec support in an already-built FreeRDP client. Use a client built
with H.264 decoding, or omit `/gfx:AVC420` to connect using bitmap updates.
The interoperability tests used a separate H.264-enabled FreeRDP build.

For the X11 client, also check for `WITH_XCURSOR=ON`. Virtio-gpu hardware cursors
are sent as separate RDP pointer updates, so the client draws them immediately
without waiting for a video frame. Shapes and hotspots are cached; moving the
hardware cursor alone does not encode desktop video. Transparent padding is
trimmed, and shapes larger than the supported 32×32 RDP pointer size are reduced.
Older clients receive a compatible color cursor without alpha blending.

Guest software cursors, including the VESA fallback, remain in the desktop
image and require the client's local cursor to be hidden. FreeRDP builds without
Xcursor support can show a second pointer in this case. VNC and PNG outputs
continue receiving an image with the guest cursor composed into it, including
when used alongside RDP.

The server logs `using OpenH264 AVC420 graphics` after negotiation. Clients
without AVC420 support receive bitmap updates automatically. The tagged binary
links to the OpenH264 shared library, which must remain installed at runtime.
Normal builds need no OpenH264 dependency and report a clear error if
`-rdp-h264` is requested.

OpenH264 can encode slices of the same frame on multiple CPU cores. The default
`-rdp-h264-threads 0` selects up to two workers per client, taking the Go CPU
limit, guest vCPU count, and number of active AVC clients into account. Explicit
values from 1 to 16 set a per-client worker limit; `-rdp-h264-threads 1` forces
serial encoding. Clients share an encoding CPU budget that reserves capacity
for guest vCPUs and one CPU for host work, including audio, with a minimum budget
of one encoding worker. Frames remain ordered, and slow
clients continue receiving the latest frame without building a queue.

Use `-rdp-stats` to log performance measurements every five seconds while frames
are active, plus a summary when the client disconnects. Statistics include frame
and compressed payload byte counts, actual worker count, average/maximum time
for framebuffer copying, color conversion, H.264 encoding and network writes,
and time spent waiting for client acknowledgements
or available encoding workers. Statistics are disabled by default and also
work with bitmap RDP. For example:

```bash
./gokvm boot -iso ./slax.iso -m 2G -net user \
  -rdp 127.0.0.1:3390 -rdp-h264 -rdp-h264-threads 2 -rdp-stats
```

Try worker limits of 1, 2, and 4 while scrolling the same page, then compare
encoding time and responsiveness. More workers help when encoding is the
bottleneck; guest rendering, color conversion, and client acknowledgement waits
can also limit frame rate. These options require no changes to the RDP client
command. `-rdp-h264-threads` requires `-rdp-h264`; `-rdp-stats` requires `-rdp`.

AVC420 uses lossy YUV 4:2:0 compression and software encoding, so small colored
text can be softer than bitmap output. AVC444 and hardware encoding are not
implemented. TLS and authentication behavior are the same as for bitmap RDP.

The H.264 path sends changed frames at up to 60 fps. Virtio-gpu dirty rectangles
limit pixel conversion and framebuffer copying to changed regions; each client
retains its own image and catches up safely after skipping intermediate frames.
Bitmap output updates changed 64×64 tiles, while H.264 updates a persistent YUV
image and sends repaint rectangles with its compressed video. H.264 still
encodes a complete video picture, using references to earlier frames.

Frame notifications avoid waiting for a separate polling cycle, and slow clients
receive the latest available image when ready. Resizing and refresh requests
repaint the whole desktop. Serial/VGA/VESA fallbacks still publish full images.
Actual frame rate depends on guest rendering, host CPU, and client decoding;
bitmap RDP remains capped at 30 fps.

Virtio-gpu currently provides a 2D framebuffer without guest 3D acceleration.
Complex browser pages can therefore be limited by software rendering inside
the guest even when AVC420 is active. For choppy scrolling, first check the
server's `using OpenH264 AVC420 graphics` message to confirm the session is
using the faster graphics path.

#### Audio playback

Add `-audio rdp` to expose a virtio-snd card and play guest audio through the
connected RDP client. The tested Slax ISO already includes the Linux
`virtio_snd` driver. Audio itself needs no native library or special build tag
in gokvm; `-rdp-h264` still requires the OpenH264 build described above.

```bash
./gokvm boot -iso ./slax.iso -m 2G -net user \
  -rdp 127.0.0.1:3390 -rdp-h264 -audio rdp

xfreerdp /v:127.0.0.1:3390 /sec:tls /u:console /p \
  /cert:ignore /gfx:AVC420 /sound:sys:pulse,latency:40
```

The client must be built with a working audio output backend. Check
`xfreerdp /buildconfig` for `WITH_PULSE=ON` when using `sys:pulse`, or
`WITH_ALSA=ON` for `sys:alsa`. A client that logs `Loaded fake backend for rdpsnd`
cannot produce sound. PulseAudio-compatible PipeWire servers work with the
Pulse backend. Audio negotiation logs `client using PCM audio` on the server.
The 40 ms client buffer helps absorb scheduling jitter during playback.
The tested Pulse backend can briefly rebuffer when playback resumes after a
long silence.

Playback is stereo, signed 16-bit PCM at 48 kHz; guest ALSA/PulseAudio can convert
application formats. Use `aplay -l` in the guest to check that the card is present,
and `aplay -D plughw:0,0 example.wav` to test it directly. Microphone capture is
not implemented. Audio is off by default, and `-audio rdp` requires `-rdp`.
Disconnected or slow clients cannot stop the guest audio clock; old queued
samples are discarded to keep buffering bounded. VNC carries no audio.

Run the protocol, input, framebuffer, and listener tests without booting a VM:

```bash
go test ./internal/rdp ./virtio ./flag ./vmm -short
go test -race ./internal/rdp ./virtio ./vmm -run 'Test(RDP|GPU|Modern|Framebuffer|SerialMirror)'
```

With the OpenH264 development library installed, also run the native encoder
roundtrip and graphics pipeline tests:

```bash
go test -tags openh264 -short ./internal/rdp/... ./virtio ./vmm ./flag
```

## Go package

This project includes a thin wrapper for the KVM API using ioctl. Please refer to the following link to use it.

https://pkg.go.dev/github.com/bobuhiro11/gokvm

## Reference

Thanks to the many useful resources on KVM, this project was able to boot Linux on a virtual machine.

- [The Definitive KVM API Documentation](https://docs.kernel.org/virt/kvm/api.html#)
- [Using the KVM API, lwn.net](https://lwn.net/Articles/658511/)
- [kvmtest.c, lwn.net](https://lwn.net/Articles/658512/)
- [KVM tool](https://git.kernel.org/pub/scm/linux/kernel/git/will/kvmtool.git/about/)
- [kvm-hello-world](https://github.com/dpw/kvm-hello-world)
- [linux kvm-api: types,structures, consts](https://github.com/torvalds/linux/blob/master/include/uapi/linux/kvm.h)
- [aghosn/kvm.go](https://gist.github.com/aghosn/f72c8e8f53bf99c3c4117f49677ab0b9)
- [KVM HOST IN A FEW LINES OF CODE](https://zserge.com/posts/kvm/)
- [zserge/kvm-host.c](https://gist.github.com/zserge/ae9098a75b2b83a1299d19b79b5fe488)
- [CS 695: Virtualization and Cloud Computing, cse.iitb.ac.in](https://www.cse.iitb.ac.in/~cs695/)
- [The Linux/x86 Boot Protocol, kernel.org](https://www.kernel.org/doc/html/latest/x86/boot.html)
- [Build and run minimal Linux / Busybox systems in Qemu](https://gist.github.com/chrisdone/02e165a0004be33734ac2334f215380e)
- [kvm_cost.go, google/gvisor](https://github.com/google/gvisor/blob/master/pkg/sentry/platform/kvm/kvm_const.go)
- [Serial UART information, www.lammertbies.nl](https://www.lammertbies.nl/comm/info/serial-uart)
- [Virtual I/O Device (VIRTIO) Version 1.1](https://docs.oasis-open.org/virtio/virtio/v1.1/csprd01/virtio-v1.1-csprd01.html)
- [rust-vmm/vm-virtio](https://github.com/rust-vmm/vm-virtio/tree/main/crates/virtio-queue)
- [ハイパーバイザの作り方～ちゃんと理解する仮想化技術～ 第１１回 virtioによる準仮想化デバイス その１「virtioの概要とVirtio PCI」](https://syuu1228.github.io/howto_implement_hypervisor/part11.html)
- [ハイパーバイザの作り方～ちゃんと理解する仮想化技術～ 第１２回 virtioによる準仮想化デバイス その２「Virtqueueとvirtio-netの実現」](https://syuu1228.github.io/howto_implement_hypervisor/part12.html)
- [Xen PVH boot protocol](https://github.com/mirage/xen/blob/master/docs/misc/hvmlite.markdown)
- [Cloud Hypervisor](https://github.com/cloud-hypervisor/cloud-hypervisor)
