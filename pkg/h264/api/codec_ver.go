// Port of codec/api/wels/codec_ver.h.

package api

// G_stCodecVersion is the version of the codec (static const g_stCodecVersion).
var G_stCodecVersion = OpenH264Version{2, 6, 0, 2502}

// G_strCodecVer is the version string (static const g_strCodecVer).
const G_strCodecVer = "OpenH264 version:2.6.0.2502"

const (
	OPENH264_MAJOR    = 2
	OPENH264_MINOR    = 6
	OPENH264_REVISION = 0
	OPENH264_RESERVED = 2502
)
