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
	EV_SYN = 0
	EV_KEY = 1
	EV_ABS = 3
	ABS_X               = 0
	ABS_Y               = 1
	ABS_MT_POSITION_X = 53
	ABS_MT_POSITION_Y = 54
	ABS_MT_SLOT       = 47
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
	fds := syscall.FdSet{}
	fdSet(&fds, fd)
	n, err := syscall.Select(fd+1, &fds, nil, nil, tv)
	if err != nil || n == 0 {
		return nil
	}
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
			switch ev.Code {
			case ABS_X, ABS_MT_POSITION_X:
				r.x = int(ev.Value)
			case ABS_Y, ABS_MT_POSITION_Y:
				r.y = int(ev.Value)
			}
		} else if ev.Type == EV_KEY && ev.Code == BTN_TOUCH {
			press := ev.Value != 0
			log.Printf("[INPUT] touch x=%d y=%d press=%v", r.x, r.y, press)
			if press {
				return &TouchEvent{X: r.x, Y: r.y, Press: true}
			}
			return &TouchEvent{X: r.x, Y: r.y, Press: false}
		} else if ev.Type == EV_SYN {
			// SYN event may indicate end of MT frame, but we handle via BTN_TOUCH
		}
		// check if more data pending
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

// PowerWatcher watches power button double-press on all event devices
type PowerWatcher struct {
	files     []*os.File
	eventSize int
	bufs      [][]byte
	last      time.Time
	double    bool
	simulate  bool
}

func NewPowerWatcher(path string) *PowerWatcher {
	// Try all likely power devices: event0, event1, event2
	candidates := []string{"/dev/input/event0", "/dev/input/event1", "/dev/input/event2"}
	if path != "" && path != "/dev/input/event0" {
		candidates = append([]string{path}, candidates...)
	}
	var files []*os.File
	var bufs [][]byte
	sz := 16
	for _, p := range candidates {
		f, err := os.OpenFile(p, os.O_RDONLY, 0)
		if err != nil {
			continue
		}
		sz = detectEventSize(f)
		_ = syscall.SetNonblock(int(f.Fd()), true)
		log.Printf("[POWER] watching %s size=%d", p, sz)
		files = append(files, f)
		bufs = append(bufs, make([]byte, sz))
	}
	if len(files) == 0 {
		return &PowerWatcher{simulate: true, eventSize: 16}
	}
	return &PowerWatcher{files: files, eventSize: sz, bufs: bufs}
}

func (p *PowerWatcher) Poll() {
	if p.simulate || len(p.files) == 0 {
		return
	}
	for idx, f := range p.files {
		fd := int(f.Fd())
		buf := p.bufs[idx]
		for {
			n, err := syscall.Read(fd, buf)
			if err != nil {
				if err == syscall.EAGAIN {
					break
				}
				break
			}
			if n < p.eventSize {
				break
			}
			ev := parseEvent(buf, p.eventSize)
			if ev == nil {
				continue
			}
			// Log all KEY events for debug
			if ev.Type == EV_KEY {
				log.Printf("[POWER] KEY code=%d value=%d", ev.Code, ev.Value)
			}
			if ev.Type == EV_KEY && ev.Value == 1 && (ev.Code == KEY_POWER || ev.Code == KEY_SLEEP || ev.Code == 116 || ev.Code == 142) {
				now := time.Now()
				elapsed := now.Sub(p.last)
				log.Printf("[POWER] power press elapsed=%v", elapsed)
				if elapsed < 800*time.Millisecond {
					p.double = true
					log.Printf("[POWER] DOUBLE detected!")
				}
				p.last = now
			}
			// Also treat long hold (2s) as exit via IsDouble check? handled via timeout in main
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
	for _, f := range p.files {
		if f != nil {
			_ = f.Close()
		}
	}
}

// CombinedReader avoids fd conflict: single fd for touch+power if same device
type CombinedReader struct {
	touch *Reader
	power *PowerWatcher
}

func NewCombined() (*Reader, *PowerWatcher) {
	r := NewReader(nil)
	p := NewPowerWatcher("/dev/input/event0")
	return r, p
}
