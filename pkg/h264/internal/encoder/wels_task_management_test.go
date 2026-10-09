// Port of test/encoder/EncUT_EncoderTaskManagement.cpp (plus a check of the
// sequential task execution order).

package encoder

import "testing"

func newTaskMgmtTestCtx() *sWelsEncCtx {
	sCtx := &sWelsEncCtx{}
	sWelsSvcCodingParam := NewSWelsSvcCodingParam()
	sCtx.pSvcParam = sWelsSvcCodingParam
	sWelsSvcCodingParam.IMultipleThreadIdc = 4
	sCtx.iMaxSliceCount = 35
	return sCtx
}

func TestEncoderTaskManagement_CWelsTaskManageBase(t *testing.T) {
	sCtx := newTaskMgmtTestCtx()
	pTaskManage := CreateTaskManage(sCtx, 1, false)
	if pTaskManage == nil {
		t.Fatal("CreateTaskManage returned NULL")
	}
	pTaskManage.Destruct()
}

func TestEncoderTaskManagement_CWelsTaskManageParallel(t *testing.T) {
	sCtx := newTaskMgmtTestCtx()
	pTaskManage := CreateTaskManage(sCtx, 1, true)
	if pTaskManage == nil {
		t.Fatal("CreateTaskManage returned NULL")
	}
	pTaskManage.Destruct()
}

func TestEncoderTaskManagement_CWelsTaskManageMultiD(t *testing.T) {
	sCtx := newTaskMgmtTestCtx()
	sCtx.pSvcParam.SSpatialLayers[0].SSliceArgument.UiSliceNum = 35

	pTaskManage := CreateTaskManage(sCtx, 4, true)
	if pTaskManage == nil {
		t.Fatal("CreateTaskManage returned NULL")
	}
	pBase := pTaskManage.(*CWelsTaskManageBase)
	if pBase.m_iTaskNum[0] != 35 || len(*pBase.m_cEncodingTaskList[0]) != 35 || len(*pBase.m_cPreEncodingTaskList[0]) != 35 {
		t.Fatalf("unexpected task count %d", pBase.m_iTaskNum[0])
	}
	if pTaskManage.GetThreadPoolThreadNum() != 4 {
		t.Fatalf("thread num %d", pTaskManage.GetThreadPoolThreadNum())
	}
	pTaskManage.Destruct()
}

type testOrderTask struct {
	CWelsBaseTask
	id    int32
	order *[]int32
}

func (p *testOrderTask) Execute() WelsErrorType {
	*p.order = append(*p.order, p.id)
	return ENC_RETURN_SUCCESS
}
func (p *testOrderTask) GetTaskType() uint32 { return uint32(WELS_ENC_TASK_ENCODING) }

func TestEncoderTaskManagement_SequentialOrder(t *testing.T) {
	sCtx := newTaskMgmtTestCtx()
	sCtx.pSvcParam.SSpatialLayers[1].SSliceArgument.UiSliceNum = 5
	pTaskManage := CreateTaskManage(sCtx, 2, false)
	if pTaskManage == nil {
		t.Fatal("CreateTaskManage returned NULL")
	}
	pBase := pTaskManage.(*CWelsTaskManageBase)
	var order []int32
	list := TASKLIST_TYPE{}
	for i := int32(0); i < 5; i++ {
		task := &testOrderTask{id: i, order: &order}
		task.ctorCWelsBaseTask(pBase)
		list = append(list, task)
	}
	*pBase.m_cEncodingTaskList[1] = list
	sCtx.pCurDqLayer = &SDqLayer{}
	pTaskManage.InitFrame(1)
	if ret := pTaskManage.ExecuteTasks(WELS_ENC_TASK_ENCODING); ret != ENC_RETURN_SUCCESS {
		t.Fatalf("ExecuteTasks returned %d", ret)
	}
	if len(order) != 5 {
		t.Fatalf("executed %d tasks", len(order))
	}
	for i, id := range order {
		if int32(i) != id {
			t.Fatalf("order %v", order)
		}
	}
	if pBase.m_iWaitTaskNum != 0 {
		t.Fatalf("m_iWaitTaskNum %d", pBase.m_iWaitTaskNum)
	}
	pTaskManage.Destruct()
}
