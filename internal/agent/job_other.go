//go:build !windows

package agent

import "os"

// procJob 在非 Windows 平台是空实现：类 Unix 下 agent 若作为进程组/会话首进程，
// 子进程多半随父退出；同时 Run() 退出时的 stopAll() 会主动停掉所有插件。
type procJob struct{}

func newProcJob() (*procJob, error) { return &procJob{}, nil }

func (j *procJob) assign(p *os.Process) {}

func (j *procJob) close() {}
