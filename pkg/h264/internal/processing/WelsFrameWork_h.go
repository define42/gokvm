package processing

// Port of codec/processing/src/common/WelsFrameWork.h.

const MAX_STRATEGY_NUM = METHOD_MASK - 1

// IStrategy is the abstract base class of every processing method. Concrete
// strategies embed IStrategyBase (which provides the C++ default
// implementations of Init/Uninit/Flush/Get/Set/SpecialFeature and the data
// members) and implement Process (and override Get/Set where the C++ does).
type IStrategy interface {
	IWelsVP
	strategyBase() *IStrategyBase
}

// IStrategyBase holds the data members and default virtual methods of the C++
// IStrategy class.
type IStrategyBase struct {
	m_eMethod EMethods
	m_eFormat EVideoFormat
	m_iIndex  int32
	m_bInit   bool
}

// initIStrategyBase is the IStrategy constructor.
func (s *IStrategyBase) initIStrategyBase() {
	s.m_eMethod = METHOD_NULL
	s.m_eFormat = VIDEO_FORMAT_I420
	s.m_iIndex = 0
	s.m_bInit = false
}

func (s *IStrategyBase) strategyBase() *IStrategyBase { return s }

func (s *IStrategyBase) Init(iType int32, pCfg any) EResult {
	return RET_SUCCESS
}
func (s *IStrategyBase) Uninit(iType int32) EResult {
	return RET_SUCCESS
}
func (s *IStrategyBase) Flush(iType int32) EResult {
	return RET_SUCCESS
}
func (s *IStrategyBase) Get(iType int32, pParam any) EResult {
	return RET_SUCCESS
}
func (s *IStrategyBase) Set(iType int32, pParam any) EResult {
	return RET_SUCCESS
}
func (s *IStrategyBase) SpecialFeature(iType int32, pIn any, pOut any) EResult {
	return RET_SUCCESS
}

// CVpFrameWork is the IWelsVP implementation returned by
// WelsCreateVpInterface. The C++ mutex is dropped (single-threaded port).
type CVpFrameWork struct {
	m_pStgChain [MAX_STRATEGY_NUM]IStrategy
}
