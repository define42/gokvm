// Port of codec/encoder/core/inc/wels_task_management.h.
//
// The OS thread pool, events, and locks are replaced by bounded Go workers
// for the audited fixed-slice path. Other task lists execute sequentially.

package encoder

// IWelsTaskManage is the virtual interface of the task managers. Create one
// with CreateTaskManage (wels_task_management.go).
type IWelsTaskManage interface {
	Init(pEncCtx *sWelsEncCtx) WelsErrorType
	Uninit()
	InitFrame(kiCurDid int32)
	// ExecuteTasks: C++ default argument iTaskType = WELS_ENC_TASK_ENCODING
	// must be passed explicitly.
	ExecuteTasks(iTaskType ETaskType) WelsErrorType
	GetThreadPoolThreadNum() int32
	// Destruct is the virtual destructor (C++ delete / WELS_DELETE_OP).
	Destruct()
}

// TASKLIST_TYPE is CWelsNonDuplicatedList<CWelsBaseTask>: push_back ->
// append, begin -> [0] (nil when empty), pop_front -> [1:], getNode(i) -> [i].
type TASKLIST_TYPE []IWelsBaseTask

// CWelsTaskManageBase runs sequential tasks and bounded fixed-slice workers.
type CWelsTaskManageBase struct {
	// protected:
	m_pEncCtx *sWelsEncCtx

	m_pcAllTaskList        [WELS_ENC_TASK_ALL][MAX_DEPENDENCY_LAYER]*TASKLIST_TYPE
	m_cEncodingTaskList    [MAX_DEPENDENCY_LAYER]*TASKLIST_TYPE
	m_cPreEncodingTaskList [MAX_DEPENDENCY_LAYER]*TASKLIST_TYPE
	m_iTaskNum             [MAX_DEPENDENCY_LAYER]int32

	m_iThreadNum   int32 // thread count requested (the pool's thread count in C)
	m_iWaitTaskNum int32

	// private:
	m_iCurDid int32
}

// NewCWelsTaskManageBase is the C++ constructor (new CWelsTaskManageBase()).
func NewCWelsTaskManageBase() *CWelsTaskManageBase {
	p := &CWelsTaskManageBase{}
	p.ctorCWelsTaskManageBase()
	return p
}

// CWelsTaskManageOne runs all tasks in the calling thread (for test).
type CWelsTaskManageOne struct {
	CWelsTaskManageBase
}

// NewCWelsTaskManageOne is the C++ constructor (declared, never defined in
// C++; runs the base constructor).
func NewCWelsTaskManageOne() *CWelsTaskManageOne {
	p := &CWelsTaskManageOne{}
	p.ctorCWelsTaskManageBase()
	return p
}

func (p *CWelsTaskManageOne) GetThreadPoolThreadNum() int32 { return 1 }

var (
	_ IWelsTaskManage = (*CWelsTaskManageBase)(nil)
	_ IWelsTaskManage = (*CWelsTaskManageOne)(nil)
	_ IWelsTaskSink   = (*CWelsTaskManageBase)(nil)
)
