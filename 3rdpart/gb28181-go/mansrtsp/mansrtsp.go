package mansrtsp

import "fmt"

// Pause builds a MANSRTSP PAUSE body for SIP INFO.
func Pause(cseq int) string {
	return fmt.Sprintf("PAUSE RTSP/1.0\r\nCSeq: %d\r\nPauseTime: now\r\n", cseq)
}

// Resume builds a MANSRTSP PLAY (resume from now) body.
func Resume(cseq int) string {
	return fmt.Sprintf("PLAY RTSP/1.0\r\nCSeq: %d\r\nRange: npt=now-\r\n", cseq)
}

// Speed builds a MANSRTSP PLAY with Scale.
func Speed(cseq int, speed float64) string {
	return fmt.Sprintf("PLAY RTSP/1.0\r\nCSeq: %d\r\nScale: %.6f\r\n", cseq, speed)
}

// Seek builds a MANSRTSP PLAY with npt seek offset (seconds).
func Seek(cseq int, seekTime int64) string {
	return fmt.Sprintf("PLAY RTSP/1.0\r\nCSeq: %d\r\nRange: npt=%d-\r\n", cseq, seekTime)
}
