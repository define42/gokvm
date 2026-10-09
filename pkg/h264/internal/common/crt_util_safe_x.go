package common

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Port of codec/common/src/crt_util_safe_x.cpp (POSIX branch).

// cFormatToGo converts a C printf format string to a Go fmt format string:
// length modifiers (h, hh, l, ll, L, j, z, t, I64, I32) are dropped and the
// %u / %i conversions become %d. Formats that are already Go syntax pass
// through unchanged.
func cFormatToGo(f string) string {
	if !strings.Contains(f, "%") {
		return f
	}
	isDigit := func(c byte) bool { return c >= '0' && c <= '9' }
	var b strings.Builder
	b.Grow(len(f))
	for i := 0; i < len(f); i++ {
		c := f[i]
		if c != '%' {
			b.WriteByte(c)
			continue
		}
		j := i + 1
		if j < len(f) && f[j] == '%' {
			b.WriteString("%%")
			i = j
			continue
		}
		start := j
		for j < len(f) && strings.IndexByte("-+ #0", f[j]) >= 0 {
			j++
		}
		for j < len(f) && (isDigit(f[j]) || f[j] == '*') {
			j++
		}
		if j < len(f) && f[j] == '.' {
			j++
			for j < len(f) && (isDigit(f[j]) || f[j] == '*') {
				j++
			}
		}
		spec := f[start:j]
		for j < len(f) {
			if strings.IndexByte("hlLjz", f[j]) >= 0 {
				j++
			} else if f[j] == 't' && j+1 < len(f) && strings.IndexByte("diouxX", f[j+1]) >= 0 {
				j++
			} else if strings.HasPrefix(f[j:], "I64") || strings.HasPrefix(f[j:], "I32") {
				j += 3
			} else {
				break
			}
		}
		if j >= len(f) {
			b.WriteByte('%')
			b.WriteString(spec)
			break
		}
		verb := f[j]
		switch verb {
		case 'u', 'i':
			verb = 'd'
		}
		b.WriteByte('%')
		b.WriteString(spec)
		b.WriteByte(verb)
		i = j
	}
	return b.String()
}

// truncateC returns s truncated the way a C buffer of iSize bytes (including
// the terminating NUL) would hold it.
func truncateC(s string, iSize int) string {
	if iSize <= 0 {
		return ""
	}
	if len(s) > iSize-1 {
		return s[:iSize-1]
	}
	return s
}

// WelsSnprintf is snprintf: the formatted text, truncated to
// iSizeOfBuffer-1 bytes, is stored in *pBuffer (if non nil); the return
// value is the length of the untruncated output.
func WelsSnprintf(pBuffer *string, iSizeOfBuffer int32, kpFormat string, args ...any) int32 {
	return WelsVsnprintf(pBuffer, iSizeOfBuffer, kpFormat, args)
}

// WelsStrncpy copies kpSrc into *pDest, truncated to iSizeInBytes-1 bytes.
func WelsStrncpy(pDest *string, iSizeInBytes int32, kpSrc string) string {
	if i := strings.IndexByte(kpSrc, 0); i >= 0 {
		kpSrc = kpSrc[:i]
	}
	s := truncateC(kpSrc, int(iSizeInBytes))
	if pDest != nil {
		*pDest = s
	}
	return s
}

// WelsVsnprintf is vsnprintf with the va_list given as []any.
func WelsVsnprintf(pBuffer *string, iSizeOfBuffer int32, kpFormat string, argptr []any) int32 {
	s := fmt.Sprintf(cFormatToGo(kpFormat), argptr...)
	if pBuffer != nil {
		*pBuffer = truncateC(s, int(iSizeOfBuffer))
	}
	return int32(len(s))
}

// WelsStrcat appends kpSrc to *pDest, keeping the total within
// uiSizeInBytes-1 bytes.
func WelsStrcat(pDest *string, uiSizeInBytes uint32, kpSrc string) string {
	uiCurLen := uint32(len(*pDest))
	if uiSizeInBytes > uiCurLen {
		var tail string
		WelsStrncpy(&tail, int32(uiSizeInBytes-uiCurLen), kpSrc)
		*pDest += tail
	}
	return *pDest
}

// WelsFopen is fopen; it returns nil on failure.
func WelsFopen(kpFilename string, kpMode string) *WelsFileHandle {
	mode := strings.ReplaceAll(kpMode, "b", "")
	var flag int
	switch mode {
	case "r":
		flag = os.O_RDONLY
	case "r+":
		flag = os.O_RDWR
	case "w":
		flag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	case "w+":
		flag = os.O_RDWR | os.O_CREATE | os.O_TRUNC
	case "a":
		flag = os.O_WRONLY | os.O_CREATE | os.O_APPEND
	case "a+":
		flag = os.O_RDWR | os.O_CREATE | os.O_APPEND
	default:
		return nil
	}
	fp, err := os.OpenFile(kpFilename, flag, 0o666)
	if err != nil {
		return nil
	}
	return fp
}

// WelsFclose is fclose (0 on success, -1 (EOF) on failure).
func WelsFclose(pFp *WelsFileHandle) int32 {
	if pFp == nil {
		return -1
	}
	if pFp.Close() != nil {
		return -1
	}
	return 0
}

// WelsFread is fread: it returns the number of complete items read.
func WelsFread(pBuffer []byte, iSize int32, iCount int32, pFp *WelsFileHandle) int32 {
	if pFp == nil || iSize <= 0 || iCount <= 0 {
		return 0
	}
	n, _ := io.ReadFull(pFp, pBuffer[:int(iSize)*int(iCount)])
	return int32(n / int(iSize))
}

// WelsFwrite is fwrite: it returns the number of complete items written.
func WelsFwrite(kpBuffer []byte, iSize int32, iCount int32, pFp *WelsFileHandle) int32 {
	if pFp == nil || iSize <= 0 || iCount <= 0 {
		return 0
	}
	n, _ := pFp.Write(kpBuffer[:int(iSize)*int(iCount)])
	return int32(n / int(iSize))
}

// WelsFseek is fseek (0 on success, -1 on failure).
func WelsFseek(fp *WelsFileHandle, offset int32, origin int32) int32 {
	if fp == nil {
		return -1
	}
	if _, err := fp.Seek(int64(offset), int(origin)); err != nil {
		return -1
	}
	return 0
}

// WelsFflush is fflush (os.File is unbuffered, so this only validates fp).
func WelsFflush(pFp *WelsFileHandle) int32 {
	if pFp == nil {
		return -1
	}
	return 0
}

// WelsGetTimeOfDay fills pTp with the current time.
func WelsGetTimeOfDay(pTp *SWelsTime) int32 {
	now := time.Now()
	pTp.Time = now.Unix()
	// (uint16_t)sTv.tv_usec / 1000 : the cast binds before the division in C.
	pTp.Millitm = uint16(now.Nanosecond()/1000) / 1000
	return 0
}

// strftimeGo implements the subset of strftime conversions used by the
// codec (%y %Y %m %d %H %M %S %j %p %%); others are copied verbatim.
func strftimeGo(kpFormat string, t time.Time) string {
	var b strings.Builder
	for i := 0; i < len(kpFormat); i++ {
		c := kpFormat[i]
		if c != '%' || i+1 >= len(kpFormat) {
			b.WriteByte(c)
			continue
		}
		i++
		switch kpFormat[i] {
		case 'y':
			fmt.Fprintf(&b, "%02d", t.Year()%100)
		case 'Y':
			fmt.Fprintf(&b, "%d", t.Year())
		case 'm':
			fmt.Fprintf(&b, "%02d", int(t.Month()))
		case 'd':
			fmt.Fprintf(&b, "%02d", t.Day())
		case 'H':
			fmt.Fprintf(&b, "%02d", t.Hour())
		case 'M':
			fmt.Fprintf(&b, "%02d", t.Minute())
		case 'S':
			fmt.Fprintf(&b, "%02d", t.Second())
		case 'j':
			fmt.Fprintf(&b, "%03d", t.YearDay())
		case 'p':
			if t.Hour() < 12 {
				b.WriteString("AM")
			} else {
				b.WriteString("PM")
			}
		case '%':
			b.WriteByte('%')
		default:
			b.WriteByte('%')
			b.WriteByte(kpFormat[i])
		}
	}
	return b.String()
}

// WelsStrftime formats kpTp (local time) into *pBuffer. As strftime it
// returns 0 (and stores "") when the result does not fit in iSize bytes
// including the terminating NUL.
func WelsStrftime(pBuffer *string, iSize int32, kpFormat string, kpTp *SWelsTime) int32 {
	s := strftimeGo(kpFormat, time.Unix(kpTp.Time, 0).Local())
	if len(s)+1 > int(iSize) {
		if pBuffer != nil {
			*pBuffer = ""
		}
		return 0
	}
	if pBuffer != nil {
		*pBuffer = s
	}
	return int32(len(s))
}

// WelsGetMillisecond returns kpTp.Millitm.
func WelsGetMillisecond(kpTp *SWelsTime) uint16 {
	return kpTp.Millitm
}
