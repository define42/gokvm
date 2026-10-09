// Port of codec/encoder/core/src/wels_task_management.cpp.
//
// The thread pool is not ported: the queued tasks of a list are executed
// sequentially, in list order, on the calling goroutine, and the sink is
// notified after each task exactly as the pool would do.

package encoder

import (
	"github.com/define42/gokvm/pkg/h264/api"
)

// CreateTaskManage is the static IWelsTaskManage::CreateTaskManage.
func CreateTaskManage(pCtx *sWelsEncCtx, iSpatialLayer int32, bNeedLock bool) IWelsTaskManage {
	if nil == pCtx {
		return nil
	}

	pTaskManage := NewCWelsTaskManageBase()

	if ENC_RETURN_SUCCESS != pTaskManage.Init(pCtx) {
		pTaskManage.Uninit()
		pTaskManage.Destruct()
		return nil
	}
	return pTaskManage
}

// ctorCWelsTaskManageBase is the constructor body CWelsTaskManageBase::CWelsTaskManageBase.
func (p *CWelsTaskManageBase) ctorCWelsTaskManageBase() {
	p.m_pEncCtx = nil
	p.m_iWaitTaskNum = 0

	for iDid := 0; iDid < MAX_DEPENDENCY_LAYER; iDid++ {
		p.m_iTaskNum[iDid] = 0
		p.m_cEncodingTaskList[iDid] = new(TASKLIST_TYPE)
		p.m_cPreEncodingTaskList[iDid] = new(TASKLIST_TYPE)
	}
}

// Destruct is the destructor CWelsTaskManageBase::~CWelsTaskManageBase.
func (p *CWelsTaskManageBase) Destruct() {
	p.Uninit()
}

func (p *CWelsTaskManageBase) Init(pEncCtx *sWelsEncCtx) WelsErrorType {
	p.m_pEncCtx = pEncCtx
	p.m_iThreadNum = int32(p.m_pEncCtx.pSvcParam.IMultipleThreadIdc)
	// CWelsThreadPool::SetThreadNum: a non-positive count becomes 1.
	if p.m_iThreadNum <= 0 {
		p.m_iThreadNum = 1
	}

	iReturn := int32(ENC_RETURN_SUCCESS)
	for iDid := 0; iDid < MAX_DEPENDENCY_LAYER; iDid++ {
		p.m_pcAllTaskList[WELS_ENC_TASK_ENCODING][iDid] = p.m_cEncodingTaskList[iDid]
		p.m_pcAllTaskList[WELS_ENC_TASK_UPDATEMBMAP][iDid] = p.m_cPreEncodingTaskList[iDid]
		iReturn |= p.CreateTasks(pEncCtx, int32(iDid))
	}

	return iReturn
}

func (p *CWelsTaskManageBase) Uninit() {
	p.DestroyTasks()

	for iDid := 0; iDid < MAX_DEPENDENCY_LAYER; iDid++ {
		p.m_cEncodingTaskList[iDid] = nil
		p.m_cPreEncodingTaskList[iDid] = nil
	}
}

func (p *CWelsTaskManageBase) CreateTasks(pEncCtx *sWelsEncCtx, kiCurDid int32) WelsErrorType {
	var pTask IWelsBaseTask
	var kiTaskCount int32
	uiSliceMode := pEncCtx.pSvcParam.SSpatialLayers[kiCurDid].SSliceArgument.UiSliceMode

	if uiSliceMode != api.SM_SIZELIMITED_SLICE {
		p.m_iTaskNum[kiCurDid] = int32(pEncCtx.pSvcParam.SSpatialLayers[kiCurDid].SSliceArgument.UiSliceNum)
	} else {
		p.m_iTaskNum[kiCurDid] = int32(pEncCtx.iActiveThreadsNum)
	}
	kiTaskCount = p.m_iTaskNum[kiCurDid]

	for idx := int32(0); idx < kiTaskCount; idx++ {
		pTask = NewCWelsUpdateMbMapTask(p, pEncCtx, idx)
		*p.m_cPreEncodingTaskList[kiCurDid] = append(*p.m_cPreEncodingTaskList[kiCurDid], pTask)
	}

	for idx := int32(0); idx < kiTaskCount; idx++ {
		if uiSliceMode == api.SM_SIZELIMITED_SLICE {
			pTask = NewCWelsConstrainedSizeSlicingEncodingTask(p, pEncCtx, idx)
		} else {
			if pEncCtx.pSvcParam.BUseLoadBalancing {
				pTask = NewCWelsLoadBalancingSlicingEncodingTask(p, pEncCtx, idx)
			} else {
				pTask = NewCWelsSliceEncodingTask(p, pEncCtx, idx)
			}
		}
		*p.m_cEncodingTaskList[kiCurDid] = append(*p.m_cEncodingTaskList[kiCurDid], pTask)
	}

	return ENC_RETURN_SUCCESS
}

func (p *CWelsTaskManageBase) DestroyTaskList(pTargetTaskList *TASKLIST_TYPE) {
	if nil == pTargetTaskList {
		return
	}
	for len(*pTargetTaskList) > 0 {
		pTask := (*pTargetTaskList)[0]
		pTask.Destruct()
		*pTargetTaskList = (*pTargetTaskList)[1:]
	}
	*pTargetTaskList = nil
}

func (p *CWelsTaskManageBase) DestroyTasks() {
	for iDid := 0; iDid < MAX_DEPENDENCY_LAYER; iDid++ {
		if p.m_iTaskNum[iDid] > 0 {
			p.DestroyTaskList(p.m_cEncodingTaskList[iDid])
			p.DestroyTaskList(p.m_cPreEncodingTaskList[iDid])
			p.m_iTaskNum[iDid] = 0
			p.m_pcAllTaskList[WELS_ENC_TASK_ENCODING][iDid] = nil
		}
	}
}

// OnTaskMinusOne: WelsEventSignal decrements the wait counter (and signals
// the waiting thread when it reaches 0; nothing waits in the sequential port).
func (p *CWelsTaskManageBase) OnTaskMinusOne() {
	p.m_iWaitTaskNum--
}

func (p *CWelsTaskManageBase) OnTaskCancelled() WelsErrorType {
	p.OnTaskMinusOne()
	return ENC_RETURN_SUCCESS
}

func (p *CWelsTaskManageBase) OnTaskExecuted() WelsErrorType {
	p.OnTaskMinusOne()
	return ENC_RETURN_SUCCESS
}

// ExecuteTaskList: TASKLIST_TYPE** pTaskList (one row of m_pcAllTaskList) ->
// *[MAX_DEPENDENCY_LAYER]*TASKLIST_TYPE. The tasks are run in queue order.
func (p *CWelsTaskManageBase) ExecuteTaskList(pTaskList *[MAX_DEPENDENCY_LAYER]*TASKLIST_TYPE) WelsErrorType {
	p.m_iWaitTaskNum = p.m_iTaskNum[p.m_iCurDid]
	pTargetTaskList := pTaskList[p.m_iCurDid]
	if 0 == p.m_iWaitTaskNum {
		return ENC_RETURN_SUCCESS
	}

	iCurrentTaskCount := p.m_iWaitTaskNum //if directly use m_iWaitTaskNum in the loop make cause sync problem
	iIdx := int32(0)
	for iIdx < iCurrentTaskCount {
		// m_pThreadPool->QueueTask (pTargetTaskList->getNode (iIdx)): run it now.
		pTask := (*pTargetTaskList)[iIdx]
		pTask.Execute()
		if pSink := pTask.GetSink(); pSink != nil {
			pSink.OnTaskExecuted()
		}
		iIdx++
	}

	// WelsEventWait (&m_hTaskEvent, &m_hEventMutex, m_iWaitTaskNum): all tasks are done.

	return ENC_RETURN_SUCCESS
}

func (p *CWelsTaskManageBase) InitFrame(kiCurDid int32) {
	p.m_iCurDid = kiCurDid
	if p.m_pEncCtx.pCurDqLayer.bNeedAdjustingSlicing {
		p.ExecuteTaskList(&p.m_pcAllTaskList[WELS_ENC_TASK_UPDATEMBMAP])
	}
}

func (p *CWelsTaskManageBase) ExecuteTasks(iTaskType ETaskType) WelsErrorType {
	return p.ExecuteTaskList(&p.m_pcAllTaskList[iTaskType])
}

// GetThreadPoolThreadNum returns the thread count the (not ported) thread
// pool would have been configured with.
func (p *CWelsTaskManageBase) GetThreadPoolThreadNum() int32 {
	return p.m_iThreadNum
}

// CWelsTaskManageOne is for test

func (p *CWelsTaskManageOne) Init(pEncCtx *sWelsEncCtx) WelsErrorType {
	p.m_pEncCtx = pEncCtx

	return p.CreateTasks(pEncCtx, pEncCtx.iMaxSliceCount)
}

func (p *CWelsTaskManageOne) ExecuteTasks(iTaskType ETaskType) WelsErrorType {
	for len(*p.m_cEncodingTaskList[0]) > 0 {
		(*p.m_cEncodingTaskList[0])[0].Execute()
		*p.m_cEncodingTaskList[0] = (*p.m_cEncodingTaskList[0])[1:]
	}
	return ENC_RETURN_SUCCESS
}
