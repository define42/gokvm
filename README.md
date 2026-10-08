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

The RDP session uses a fixed 1024×768 desktop, scaling other guest framebuffer
sizes to fit. By default it sends uncompressed bitmap updates with 16-, 24-, or
32-bit color. The optional OpenH264 build adds compressed AVC420 graphics.
It includes the same serial/VGA/VESA fallbacks as VNC. Both `-rdp` and `-vnc` can
be supplied to view the same guest at once; connected viewers share its keyboard
and mouse. Clipboard, audio, drive redirection, dynamic resolution changes, and
multiple monitors are not implemented. FreeRDP 3.32.1 has been tested with TinyCore
desktop rendering, pointer positioning, menu interaction, and application launch;
Microsoft Remote Desktop has not yet been validated.

In TinyCore, right-click the desktop to open its menu. The application dock
appears along the bottom of the screen.

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

For the X11 client, also check for `WITH_XCURSOR=ON`. The guest draws its
cursor into the framebuffer, and gokvm asks the client to hide its local
cursor. FreeRDP builds without Xcursor support ignore that request, leaving
two cursors visible; during movement they can appear separated or offset.

The server logs `using OpenH264 AVC420 graphics` after negotiation. Clients
without AVC420 support receive bitmap updates automatically. The tagged binary
links to the OpenH264 shared library, which must remain installed at runtime.
Normal builds need no OpenH264 dependency and report a clear error if
`-rdp-h264` is requested.

AVC420 uses lossy YUV 4:2:0 compression and software encoding, so small colored
text can be softer than bitmap output. AVC444 and hardware encoding are not
implemented. TLS and authentication behavior are the same as for bitmap RDP.

The H.264 path captures the linear framebuffer and sends changed frames at up
to 60 fps. Frame notifications avoid waiting for a separate polling cycle,
and slow clients receive the latest available frame when they are ready.
Actual frame rate depends on guest rendering, host CPU, and client decoding.
Bitmap RDP remains capped at 30 fps.

Run the protocol, input, framebuffer, and listener tests without booting a VM:

```bash
go test ./internal/rdp ./virtio ./flag ./vmm -short
go test -race ./internal/rdp ./virtio ./vmm -run 'Test(RDP|Framebuffer|SerialMirror)'
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
