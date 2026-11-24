//go:build windows

package agent

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

// procJob 封装一个 Windows Job 对象。设置了 JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE 后，
// 句柄关闭时（包括 agent 被硬杀/崩溃时 OS 自动回收句柄）会把绑定的插件子进程一并杀掉，
// 从根上避免"父进程没了、插件成为占着监听端口的孤儿"。
type procJob struct {
	handle windows.Handle
}

func newProcJob() (*procJob, error) {
	h, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, err
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(h, windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		windows.CloseHandle(h)
		return nil, err
	}
	return &procJob{handle: h}, nil
}

// assign 把子进程加入本 Job。失败仅放弃保护（不致命），正常退出仍有 stopAll 兜底。
func (j *procJob) assign(p *os.Process) {
	if j == nil || j.handle == 0 || p == nil {
		return
	}
	h, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false,
		uint32(p.Pid))
	if err != nil {
		return
	}
	defer windows.CloseHandle(h)
	_ = windows.AssignProcessToJobObject(j.handle, h)
}

func (j *procJob) close() {
	if j != nil && j.handle != 0 {
		windows.CloseHandle(j.handle)
		j.handle = 0
	}
}
