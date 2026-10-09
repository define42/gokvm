package common

// Port of codec/common/src/memory_align.cpp (see memory_align_h.go).

// NewCMemoryAlign is the CMemoryAlign constructor.
func NewCMemoryAlign(kuiCacheLineSize uint32) *CMemoryAlign {
	m := &CMemoryAlign{}
	if (kuiCacheLineSize == 0) || (kuiCacheLineSize&0x0f) != 0 {
		m.m_nCacheLineSize = 0x10
	} else {
		m.m_nCacheLineSize = kuiCacheLineSize
	}
	return m
}

// memoryLength is the number of bytes CMemoryAlign::WelsMalloc accounts for
// a request of kuiSize bytes (payload + alignment slack + bookkeeping).
func (m *CMemoryAlign) memoryLength(kuiSize uint32) uint32 {
	const kiSizeOfVoidPointer = 8
	const kiSizeOfInt = 4
	return kuiSize + m.m_nCacheLineSize - 1 + kiSizeOfVoidPointer + kiSizeOfInt
}

// WelsMallocAccount records an allocation of kuiSize bytes made with
// make()/new() in the memory usage counter.
func (m *CMemoryAlign) WelsMallocAccount(kuiSize uint32, kpTag string) {
	m.m_nMemoryUsageInBytes += m.memoryLength(kuiSize)
}

// WelsFreeAccount records the release of an allocation of kuiSize bytes.
func (m *CMemoryAlign) WelsFreeAccount(kuiSize uint32, kpTag string) {
	m.m_nMemoryUsageInBytes -= m.memoryLength(kuiSize)
}

// WelsMallocz allocates kuiSize zeroed bytes.
func (m *CMemoryAlign) WelsMallocz(kuiSize uint32, kpTag string) []uint8 {
	return m.WelsMalloc(kuiSize, kpTag)
}

// WelsMalloc allocates kuiSize bytes (always zeroed in Go).
func (m *CMemoryAlign) WelsMalloc(kuiSize uint32, kpTag string) []uint8 {
	m.WelsMallocAccount(kuiSize, kpTag)
	return make([]uint8, kuiSize)
}

// WelsFree releases a buffer returned by WelsMalloc / WelsMallocz (pass the
// slice exactly as returned so its length matches the requested size).
func (m *CMemoryAlign) WelsFree(pPointer []uint8, kpTag string) {
	if pPointer != nil {
		m.WelsFreeAccount(uint32(len(pPointer)), kpTag)
	}
}

func (m *CMemoryAlign) WelsGetCacheLineSize() uint32 {
	return m.m_nCacheLineSize
}

func (m *CMemoryAlign) WelsGetMemoryUsage() uint32 {
	return m.m_nMemoryUsageInBytes
}

// WelsMallocz allocates kuiSize zeroed bytes (no accounting).
func WelsMallocz(kuiSize uint32, kpTag string) []uint8 {
	return make([]uint8, kuiSize)
}

// WelsFree is a no-op: the garbage collector releases the memory; callers
// should drop their reference (p = nil).
func WelsFree(pPtr []uint8, kpTag string) {}
