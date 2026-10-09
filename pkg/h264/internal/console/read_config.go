// Package console is a port of codec/console/common: the configuration file
// reader (CReadConfig) shared by the h264dec and h264enc console applications,
// plus the C library helpers (atoi, atof) whose exact parsing behaviour the
// applications depend on.
package console

import (
	"bufio"
	"os"
)

// CReadConfig reads parameter settings from a configuration file
// (codec/console/common/src/read_config.cpp).
type CReadConfig struct {
	m_pCfgFile       *os.File
	m_pReader        *bufio.Reader
	m_bEof           bool // feof (m_pCfgFile)
	m_strCfgFileName string
	m_iLines         uint32
}

// NewCReadConfig is CReadConfig (const char* kpConfigFileName).
func NewCReadConfig(kpConfigFileName string) *CReadConfig {
	p := &CReadConfig{m_strCfgFileName: kpConfigFileName}
	if len(kpConfigFileName) > 0 {
		p.open(kpConfigFileName)
	}
	return p
}

func (p *CReadConfig) open(kpStrFile string) {
	f, err := os.Open(kpStrFile)
	if err != nil {
		return
	}
	// fopen of a directory succeeds in C but every read fails at once.
	p.m_pCfgFile = f
	p.m_pReader = bufio.NewReader(f)
	p.m_bEof = false
}

// Openf opens the configuration file kpStrFile.
func (p *CReadConfig) Openf(kpStrFile string) {
	if len(kpStrFile) > 0 {
		p.m_strCfgFileName = kpStrFile
		p.open(kpStrFile)
	}
}

// Close is the destructor (~CReadConfig).
func (p *CReadConfig) Close() {
	if p.m_pCfgFile != nil {
		p.m_pCfgFile.Close()
		p.m_pCfgFile = nil
		p.m_pReader = nil
	}
}

// fgetc returns the next byte; at end of file (or on a read error) it sets
// the end-of-file indicator and returns EOF cast to char (0xff).
func (p *CReadConfig) fgetc() byte {
	c, err := p.m_pReader.ReadByte()
	if err != nil {
		p.m_bEof = true
		return 0xff
	}
	return c
}

// ReadLine reads one line and splits it into at most kiValSize white-space
// separated tags (strings after '#' are ignored). pVal must hold at least
// kiValSize strings (the C default is 4). It returns 1 + the number of tag
// separators seen, or 0 if no file is open.
func (p *CReadConfig) ReadLine(pVal []string, kiValSize int) int32 {
	if p.m_pCfgFile == nil || pVal == nil || kiValSize <= 1 {
		return 0
	}

	nTagNum, n := 0, 0
	bCommentFlag := false

	for n < kiValSize {
		pVal[n] = ""
		n++
	}

	for {
		kCh := p.fgetc()

		if kCh == '\n' || p.m_bEof {
			p.m_iLines++
			break
		}
		if kCh == '#' {
			bCommentFlag = true
		}
		if !bCommentFlag {
			if kCh == '\t' || kCh == ' ' {
				if nTagNum >= kiValSize-1 {
					break
				}
				if pVal[nTagNum] != "" {
					nTagNum++
				}
			} else {
				pVal[nTagNum] += string([]byte{kCh})
			}
		}
	}

	return int32(1 + nTagNum)
}

// EndOfFile reports whether the end of the file has been reached (or no
// file is open).
func (p *CReadConfig) EndOfFile() bool {
	if p.m_pCfgFile == nil {
		return true
	}
	return p.m_bEof
}

// GetLines returns the number of lines read so far.
func (p *CReadConfig) GetLines() int32 {
	return int32(p.m_iLines)
}

// ExistFile reports whether the configuration file could be opened.
func (p *CReadConfig) ExistFile() bool {
	return p.m_pCfgFile != nil
}

// GetFileName returns the configuration file name.
func (p *CReadConfig) GetFileName() string {
	return p.m_strCfgFileName
}
