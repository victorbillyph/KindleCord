package input

import (
	"encoding/binary"
	"log"
	"os"
	"syscall"
	"time"
	"unsafe"
)

const (
	EV_ABS = 3
	EV_KEY = 1
	ABS_MT_POSITION_X = 53
	ABS_MT_POSITION_Y = 54
	BTN_TOUCH         = 330
	KEY_POWER         = 116
	KEY_SLEEP         = 142
)

// TouchEvent represents a touch press/release
type TouchEvent struct {
	X, Y   int
	Press  bool // true=press, false=release
}

type inputEvent struct {
	Time  struct{ Sec, Usec int64 }
	Type  uint16
	Code  uint16
	Value int32
}

// Reader handles touch input with correct struct size handling
type Reader struct {
	file     *os.File
	simulate bool
	x, y     int
	buf      []byte
	eventSize int
}

func NewReader(paths []string) *Reader {
	if paths == nil {
		paths = []string{"/dev/input/event1", "/dev/input/event0"}
	}
	for _, p := range paths {
		f, err := os.OpenFile(p, os.O_RDONLY, 0)
		if err == nil {
			// detect event size: 16 bytes on 32-bit Kindle, 24 on x64
			sz := detectEventSize(f)
			log.Printf("[INPUT] using %s eventSize=%d", p, sz)
			return &Reader{file: f, eventSize: sz, buf: make([]byte, sz)}
		}
	}
	log.Printf("[INPUT SIM] No input device, touch disabled")
	return &Reader{simulate: true, eventSize: 16}
}

func detectEventSize(f *os.File) int {
	if unsafe.Sizeof(uintptr(0)) == 8 {
		return 24 // x64 host (PC)
	}
	return 16 // 32-bit Kindle
}

// Poll waits up to timeout and returns TouchEvent or nil
func (r *Reader) Poll(timeout time.Duration) *TouchEvent {
	if r.simulate || r.file == nil {
		time.Sleep(timeout)
		return nil
	}
	fd := int(r.file.Fd())
	var tv *syscall.Timeval
	if timeout >= 0 {
		t := syscall.NsecToTimeval(timeout.Nanoseconds())
		tv = &t
	}
	// select with timeout
	fds := syscall.FdSet{}
	// FD_SET
	fdSet(&fds, fd)
	n, err := syscall.Select(fd+1, &fds, nil, nil, tv)
	if err != nil || n == 0 {
		return nil
	}
	// read multiple events available
	for {
		nr, err := syscall.Read(fd, r.buf)
		if err != nil || nr < r.eventSize {
			return nil
		}
		ev := parseEvent(r.buf, r.eventSize)
		if ev == nil {
			return nil
		}
		if ev.Type == EV_ABS {
			if ev.Code == ABS_MT_POSITION_X {
				r.x = int(ev.Value)
			} else if ev.Code == ABS_MT_POSITION_Y {
				r.y = int(ev.Value)
			}
		} else if ev.Type == EV_KEY && ev.Code == BTN_TOUCH {
			return &TouchEvent{X: r.x, Y: r.y, Press: ev.Value != 0}
		}
		// check if more data pending without blocking
		// peek with select 0
		fds2 := syscall.FdSet{}
		fdSet(&fds2, fd)
		tv0 := syscall.Timeval{}
		nn, _ := syscall.Select(fd+1, &fds2, nil, nil, &tv0)
		if nn == 0 {
			return nil
		}
	}
}

func parseEvent(buf []byte, sz int) *inputEvent {
	if len(buf) < sz {
		return nil
	}
	var ev inputEvent
	if sz == 24 {
		// 64-bit: tv_sec 8, tv_usec 8, type 2, code 2, value 4
		ev.Time.Sec = int64(binary.LittleEndian.Uint64(buf[0:8]))
		ev.Time.Usec = int64(binary.LittleEndian.Uint64(buf[8:16]))
		ev.Type = binary.LittleEndian.Uint16(buf[16:18])
		ev.Code = binary.LittleEndian.Uint16(buf[18:20])
		ev.Value = int32(binary.LittleEndian.Uint32(buf[20:24]))
	} else {
		// 32-bit: tv_sec 4, tv_usec 4, type 2, code 2, value 4
		// Use int64 conversion via separate assignment to handle 32-bit Timeval
		sec := int64(binary.LittleEndian.Uint32(buf[0:4]))
		usec := int64(binary.LittleEndian.Uint32(buf[4:8]))
		ev.Time.Sec = sec
		ev.Time.Usec = usec
		ev.Type = binary.LittleEndian.Uint16(buf[8:10])
		ev.Code = binary.LittleEndian.Uint16(buf[10:12])
		ev.Value = int32(binary.LittleEndian.Uint32(buf[12:16]))
	}
	return &ev
}

func fdSet(fds *syscall.FdSet, fd int) {
	fds.Bits[fd/64] |= 1 << (uint(fd) % 64)
}

func (r *Reader) Close() {
	if r.file != nil {
		_ = r.file.Close()
	}
}

// PowerWatcher watches power button double-press, sharing logic without stealing touch fd
type PowerWatcher struct {
	file       *os.File
	eventSize  int
	buf        []byte
	last       time.Time
	double     bool
	simulate   bool
}

func NewPowerWatcher(path string) *PowerWatcher {
	if path == "" {
		path = "/dev/input/event0"
	}
	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return &PowerWatcher{simulate: true, eventSize: 16}
	}
	sz := detectEventSize(f)
	// make non-blocking
	_ = syscall.SetNonblock(int(f.Fd()), true)
	log.Printf("[POWER] watching %s size=%d", path, sz)
	return &PowerWatcher{file: f, eventSize: sz, buf: make([]byte, sz)}
}

func (p *PowerWatcher) Poll() {
	if p.simulate || p.file == nil {
		return
	}
	fd := int(p.file.Fd())
	for {
		n, err := syscall.Read(fd, p.buf)
		if err != nil {
			if err == syscall.EAGAIN {
				return
			}
			return
		}
		if n < p.eventSize {
			return
		}
		ev := parseEvent(p.buf, p.eventSize)
		if ev == nil {
			continue
		}
		if ev.Type == EV_KEY && ev.Value == 1 && (ev.Code == KEY_POWER || ev.Code == KEY_SLEEP) {
			now := time.Now()
			if now.Sub(p.last) < 500*time.Millisecond {
				p.double = true
			}
			p.last = now
		}
	}
}

func (p *PowerWatcher) IsDouble() bool {
	if p.double {
		p.double = false
		return true
	}
	return false
}

func (p *PowerWatcher) Close() {
	if p.file != nil {
		_ = p.file.Close()
	}
}

// CombinedReader avoids fd conflict: single fd for touch+power if same device
type CombinedReader struct {
	touch *Reader
	power *PowerWatcher
}

func NewCombined() (*Reader, *PowerWatcher) {
	// If event0 is power and event1 is touch, keep separate but set power non-blocking so it doesn't steal
	r := NewReader(nil)
	p := NewPowerWatcher("/dev/input/event0")
	// If touch used event0, disable separate power watcher to avoid double open
	if r.file != nil && p.file != nil {
		// check if same path (both opened event0)
		// we can't easily detect, but log
	}
	return r, p
}
