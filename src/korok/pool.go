package korok

//import "fmt"
import "sync"
import "strconv"

type Buffer struct {
	bs   []byte
	next *Buffer
}

func (b *Buffer) AvailSize() int {
	return cap(b.bs) - len(b.bs)
}

func (b *Buffer) Length() int {
	return len(b.bs)
}

func (b *Buffer) AppendByteSlice(bs []byte) {
	b.bs = append(b.bs, bs...)
}

func (b *Buffer) AppendByte(v byte) {
	b.bs = append(b.bs, v)
}

func (b *Buffer) AppendString(s string) {
	b.bs = append(b.bs, s...)
}

func (b *Buffer) AppendInt(i int64) {
	b.bs = strconv.AppendInt(b.bs, i, 10)
}

func (b *Buffer) AppendUint(i uint64) {
	b.bs = strconv.AppendUint(b.bs, i, 10)
}

func (b *Buffer) Bytes() []byte {
	return b.bs
}

func (b *Buffer) String() string {
	return string(b.bs)
}

func (b *Buffer) Reset() {
	b.bs = b.bs[:0]
}

type BufPool struct {
	freeBuf *Buffer
	mu      sync.Mutex
}

func (bp *BufPool) get(size int32) *Buffer {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	buffer := bp.freeBuf

	if buffer == nil {
		buffer = &Buffer{
			bs:   make([]byte, 0, size),
			next: nil,
		}
	} else {
		bp.freeBuf = buffer.next
		buffer.next = nil
		buffer.Reset()
	}

	return buffer
}

func (bp *BufPool) free(buf *Buffer) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	buf.next = bp.freeBuf
	bp.freeBuf = buf
}

func NewBufPoolWithSize(size int32) *FixedSizeBufPool {
	return &FixedSizeBufPool{
		Size: size,
		Pool: BufPool{},
	}
}

type FixedSizeBufPool struct {
	Size int32
	Pool BufPool
}

func (sbp *FixedSizeBufPool) Get() *Buffer {
	return sbp.Pool.get(sbp.Size)
}

func (sbp *FixedSizeBufPool) Free(buf *Buffer) {
	sbp.Pool.free(buf)
}
