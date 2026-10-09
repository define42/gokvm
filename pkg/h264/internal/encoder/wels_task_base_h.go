// Port of codec/encoder/core/inc/wels_task_base.h (and the parts of
// codec/common/inc/WelsTask.h the encoder needs; the common threading
// classes are not ported).

package encoder

// ETaskType is CWelsBaseTask::ETaskType.
type ETaskType = int32

const (
	WELS_ENC_TASK_ENCODING                     ETaskType = 0
	WELS_ENC_TASK_ENCODE_FIXED_SLICE           ETaskType = WELS_ENC_TASK_ENCODING
	WELS_ENC_TASK_ENCODE_SLICE_LOADBALANCING   ETaskType = WELS_ENC_TASK_ENCODING
	WELS_ENC_TASK_ENCODE_SLICE_SIZECONSTRAINED ETaskType = WELS_ENC_TASK_ENCODING
	WELS_ENC_TASK_UPDATEMBMAP                  ETaskType = 1
	WELS_ENC_TASK_PREPROCESS                   ETaskType = 2
	WELS_ENC_TASK_ALL                          ETaskType = 3
)

// IWelsTaskSink is WelsCommon::IWelsTaskSink.
type IWelsTaskSink interface {
	OnTaskExecuted() int32
	OnTaskCancelled() int32
}

// IWelsBaseTask is the virtual interface of CWelsBaseTask
// (WelsCommon::IWelsTask + GetTaskType). Destruct is the virtual destructor.
type IWelsBaseTask interface {
	Execute() WelsErrorType
	GetTaskType() uint32
	GetSink() IWelsTaskSink
	Destruct()
}

// CWelsBaseTask is the base of all encoder tasks.
type CWelsBaseTask struct {
	m_pSink IWelsTaskSink // protected (WelsCommon::IWelsTask)
}

// ctorCWelsBaseTask is the (inline) constructor CWelsBaseTask (pSink): IWelsTask (pSink).
func (p *CWelsBaseTask) ctorCWelsBaseTask(pSink IWelsTaskSink) {
	p.m_pSink = pSink
}

// GetSink is WelsCommon::IWelsTask::GetSink.
func (p *CWelsBaseTask) GetSink() IWelsTaskSink {
	return p.m_pSink
}
