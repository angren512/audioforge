package task

import "audioforge/internal/media"

// State 是任务状态。
type State string

const (
	StateQueued     State = "queued"
	StateDecrypting State = "decrypting"
	StateProbing    State = "probing"
	StateConverting State = "converting"
	StateDone       State = "done"
	StateFailed     State = "failed"
	StateCanceled   State = "canceled"
	StateSkipped    State = "skipped"
)

// Task 是单个文件的处理任务。
type Task struct {
	ID        string      `json:"id"`
	Src       string      `json:"src"`
	Name      string      `json:"name"`
	Ext       string      `json:"ext"`
	Encrypted bool        `json:"encrypted"`
	State     State       `json:"state"`
	Progress  float64     `json:"progress"`
	Message   string      `json:"message"`
	Output    string      `json:"output"`
	Info      *media.Info `json:"info"`
}

// Stats 汇总当前队列情况。
type Stats struct {
	Total    int `json:"total"`
	Queued   int `json:"queued"`
	Running  int `json:"running"`
	Done     int `json:"done"`
	Failed   int `json:"failed"`
	Canceled int `json:"canceled"`
	Skipped  int `json:"skipped"`
}

func (s State) IsFinished() bool {
	switch s {
	case StateDone, StateFailed, StateCanceled, StateSkipped:
		return true
	}
	return false
}
