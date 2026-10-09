package processing

// Port of codec/processing/src/common/WelsFrameWorkEx.cpp (C style interface).

func Init(pCtx any, iType int32, pCfg any) EResult {
	if p, ok := pCtx.(IWelsVP); ok && p != nil {
		return p.Init(iType, pCfg)
	}
	return RET_INVALIDPARAM
}
func Uninit(pCtx any, iType int32) EResult {
	if p, ok := pCtx.(IWelsVP); ok && p != nil {
		return p.Uninit(iType)
	}
	return RET_INVALIDPARAM
}
func Flush(pCtx any, iType int32) EResult {
	if p, ok := pCtx.(IWelsVP); ok && p != nil {
		return p.Flush(iType)
	}
	return RET_INVALIDPARAM
}
func Process(pCtx any, iType int32, pSrc *SPixMap, dst *SPixMap) EResult {
	if p, ok := pCtx.(IWelsVP); ok && p != nil {
		return p.Process(iType, pSrc, dst)
	}
	return RET_INVALIDPARAM
}
func Get(pCtx any, iType int32, pParam any) EResult {
	if p, ok := pCtx.(IWelsVP); ok && p != nil {
		return p.Get(iType, pParam)
	}
	return RET_INVALIDPARAM
}
func Set(pCtx any, iType int32, pParam any) EResult {
	if p, ok := pCtx.(IWelsVP); ok && p != nil {
		return p.Set(iType, pParam)
	}
	return RET_INVALIDPARAM
}
func SpecialFeature(pCtx any, iType int32, pIn any, pOut any) EResult {
	if p, ok := pCtx.(IWelsVP); ok && p != nil {
		return p.SpecialFeature(iType, pIn, pOut)
	}
	return RET_INVALIDPARAM
}

///////////////////////////////////////////////////////////////////////////////

// CreateSpecificVpInterfaceC is the C++ overload
// CreateSpecificVpInterface (IWelsVPc** pCtx).
func CreateSpecificVpInterfaceC(pCtx **IWelsVPc) EResult {
	var pWelsVP IWelsVP

	ret := CreateSpecificVpInterface(&pWelsVP)
	if ret == RET_SUCCESS {
		pVPc := &IWelsVPc{}
		pVPc.Init = Init
		pVPc.Uninit = Uninit
		pVPc.Flush = Flush
		pVPc.Process = Process
		pVPc.Get = Get
		pVPc.Set = Set
		pVPc.SpecialFeature = SpecialFeature
		pVPc.PCtx = pWelsVP
		*pCtx = pVPc
	}

	return ret
}

// DestroySpecificVpInterfaceC is the C++ overload
// DestroySpecificVpInterface (IWelsVPc* pCtx).
func DestroySpecificVpInterfaceC(pCtx *IWelsVPc) EResult {
	if pCtx != nil {
		p, _ := pCtx.PCtx.(IWelsVP)
		DestroySpecificVpInterface(p)
		pCtx.PCtx = nil
	}

	return RET_SUCCESS
}
