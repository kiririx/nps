package mux

import (
	"errors"
	"github.com/cnlh/nps/lib/common"
	"io"
	"math"
	"sync/atomic"
	"time"
	"unsafe"
)

type QueueOp struct {
	readOp  chan struct{}
	cleanOp chan struct{}
	popWait int32
}

func (Self *QueueOp) New() {
	Self.readOp = make(chan struct{})
	Self.cleanOp = make(chan struct{}, 2)
}

func (Self *QueueOp) allowPop() (closed bool) {
	if atomic.CompareAndSwapInt32(&Self.popWait, 1, 0) {
		select {
		case Self.readOp <- struct{}{}:
			return false
		case <-Self.cleanOp:
			return true
		}
	}
	return
}

func (Self *QueueOp) Clean() {
	Self.cleanOp <- struct{}{}
	Self.cleanOp <- struct{}{}
	close(Self.cleanOp)
}

type PriorityQueue struct {
	QueueOp
	highestChain *bufChain
	middleChain  *bufChain
	lowestChain  *bufChain
	hunger       uint8
}

func (Self *PriorityQueue) New() {
	Self.highestChain = new(bufChain)
	Self.highestChain.new(4)
	Self.middleChain = new(bufChain)
	Self.middleChain.new(32)
	Self.lowestChain = new(bufChain)
	Self.lowestChain.new(256)
	Self.QueueOp.New()
}

func (Self *PriorityQueue) Push(packager *common.MuxPackager) {
	switch packager.Flag {
	case common.MUX_PING_FLAG, common.MUX_PING_RETURN:
		Self.highestChain.pushHead(unsafe.Pointer(packager))
	// the ping package need highest priority
	// prevent ping calculation error
	case common.MUX_NEW_CONN, common.MUX_NEW_CONN_OK, common.MUX_NEW_CONN_Fail:
		// the new conn package need some priority too
		Self.middleChain.pushHead(unsafe.Pointer(packager))
	default:
		Self.lowestChain.pushHead(unsafe.Pointer(packager))
	}
	Self.allowPop()
	return
}

func (Self *PriorityQueue) Pop() (packager *common.MuxPackager) {
startPop:
	ptr, ok := Self.highestChain.popTail()
	if ok {
		packager = (*common.MuxPackager)(ptr)
		return
	}
	if Self.hunger < 100 {
		ptr, ok = Self.middleChain.popTail()
		if ok {
			packager = (*common.MuxPackager)(ptr)
			Self.hunger++
			return
		}
	}
	ptr, ok = Self.lowestChain.popTail()
	if ok {
		packager = (*common.MuxPackager)(ptr)
		if Self.hunger > 0 {
			Self.hunger = uint8(Self.hunger / 2)
		}
		return
	}
	// PriorityQueue is empty, notice Push method
	if atomic.CompareAndSwapInt32(&Self.popWait, 0, 1) {
		select {
		case <-Self.readOp:
			goto startPop
		case <-Self.cleanOp:
			return nil
		}
	}
	goto startPop
}

type ListElement struct {
	buf  []byte
	l    uint16
	part bool
}

func (Self *ListElement) New(buf []byte, l uint16, part bool) (err error) {
	if uint16(len(buf)) != l {
		return errors.New("ListElement: buf length not match")
	}
	Self.buf = buf
	Self.l = l
	Self.part = part
	return nil
}

type FIFOQueue struct {
	QueueOp
	chain   *bufChain
	length  uint32
	stopOp  chan struct{}
	timeout time.Time
}

func (Self *FIFOQueue) New() {
	Self.QueueOp.New()
	Self.chain = new(bufChain)
	Self.chain.new(64)
	Self.stopOp = make(chan struct{}, 1)
}

func (Self *FIFOQueue) Push(element *ListElement) {
	Self.chain.pushHead(unsafe.Pointer(element))
	Self.length += uint32(element.l)
	Self.allowPop()
	return
}

func (Self *FIFOQueue) Pop() (element *ListElement, err error) {
startPop:
	ptr, ok := Self.chain.popTail()
	if ok {
		element = (*ListElement)(ptr)
		Self.length -= uint32(element.l)
		return
	}
	if atomic.CompareAndSwapInt32(&Self.popWait, 0, 1) {
		t := Self.timeout.Sub(time.Now())
		if t <= 0 {
			t = time.Minute
		}
		timer := time.NewTimer(t)
		defer timer.Stop()
		select {
		case <-Self.readOp:
			goto startPop
		case <-Self.cleanOp:
			return
		case <-Self.stopOp:
			err = io.EOF
			return
		case <-timer.C:
			err = errors.New("mux.queue: read time out")
			return
		}
	}
	goto startPop
}

func (Self *FIFOQueue) Len() (n uint32) {
	return Self.length
}

func (Self *FIFOQueue) Stop() {
	Self.stopOp <- struct{}{}
}

func (Self *FIFOQueue) SetTimeOut(t time.Time) {
	Self.timeout = t
}

// https://golang.org/src/sync/poolqueue.go

type bufDequeue struct {
	// headTail packs together a 32-bit head index and a 32-bit
	// tail index. Both are indexes into vals modulo len(vals)-1.
	//
	// tail = index of oldest data in queue
	// head = index of next slot to fill
	//
	// Slots in the range [tail, head) are owned by consumers.
	// A consumer continues to own a slot outside this range until
	// it nils the slot, at which point ownership passes to the
	// producer.
	//
	// The head index is stored in the most-significant bits so
	// that we can atomically add to it and the overflow is
	// harmless.
	headTail uint64

	// vals is a ring buffer of interface{} values stored in this
	// dequeue. The size of this must be a power of 2.
	//
	// A slot is still in use until *both* the tail
	// index has moved beyond it and typ has been set to nil. This
	// is set to nil atomically by the consumer and read
	// atomically by the producer.
	vals []unsafe.Pointer
}

const dequeueBits = 32

// dequeueLimit is the maximum size of a bufDequeue.
//
// This must be at most (1<<dequeueBits)/2 because detecting fullness
// depends on wrapping around the ring buffer without wrapping around
// the index. We divide by 4 so this fits in an int on 32-bit.
const dequeueLimit = (1 << dequeueBits) / 4

func (d *bufDequeue) unpack(ptrs uint64) (head, tail uint32) {
	const mask = 1<<dequeueBits - 1
	head = uint32((ptrs >> dequeueBits) & mask)
	tail = uint32(ptrs & mask)
	return
}

func (d *bufDequeue) pack(head, tail uint32) uint64 {
	const mask = 1<<dequeueBits - 1
	return (uint64(head) << dequeueBits) |
		uint64(tail&mask)
}

// pushHead adds val at the head of the queue. It returns false if the
// queue is full.
func (d *bufDequeue) pushHead(val unsafe.Pointer) bool {
	var slot *unsafe.Pointer
	for {
		ptrs := atomic.LoadUint64(&d.headTail)
		head, tail := d.unpack(ptrs)
		if (tail+uint32(len(d.vals)))&(1<<dequeueBits-1) == head {
			// Queue is full.
			return false
		}
		ptrs2 := d.pack(head+1, tail)
		if atomic.CompareAndSwapUint64(&d.headTail, ptrs, ptrs2) {
			slot = &d.vals[head&uint32(len(d.vals)-1)]
			break
		}
	}
	// The head slot is free, so we own it.
	*slot = val
	return true
}

// popTail removes and returns the element at the tail of the queue.
// It returns false if the queue is empty. It may be called by any
// number of consumers.
func (d *bufDequeue) popTail() (unsafe.Pointer, bool) {
	ptrs := atomic.LoadUint64(&d.headTail)
	head, tail := d.unpack(ptrs)
	if tail == head {
		// Queue is empty.
		return nil, false
	}
	slot := &d.vals[tail&uint32(len(d.vals)-1)]
	for {
		typ := atomic.LoadPointer(slot)
		if typ != nil {
			break
		}
		// Another goroutine is still pushing data on the tail.
	}

	// We now own slot.
	val := *slot

	// Tell pushHead that we're done with this slot. Zeroing the
	// slot is also important so we don't leave behind references
	// that could keep this object live longer than necessary.
	//
	// We write to val first and then publish that we're done with
	atomic.StorePointer(slot, nil)
	// At this point pushHead owns the slot.
	if tail < math.MaxUint32 {
		atomic.AddUint64(&d.headTail, 1)
	} else {
		atomic.AddUint64(&d.headTail, ^uint64(math.MaxUint32-1))
	}
	return val, true
}

// bufChain is a dynamically-sized version of bufDequeue.
//
// This is implemented as a doubly-linked list queue of poolDequeues
// where each dequeue is double the size of the previous one. Once a
// dequeue fills up, this allocates a new one and only ever pushes to
// the latest dequeue. Pops happen from the other end of the list and
// once a dequeue is exhausted, it gets removed from the list.
type bufChain struct {
	// head is the bufDequeue to push to. This is only accessed
	// by the producer, so doesn't need to be synchronized.
	head *bufChainElt

	// tail is the bufDequeue to popTail from. This is accessed
	// by consumers, so reads and writes must be atomic.
	tail        *bufChainElt
	chainStatus int32
}

type bufChainElt struct {
	bufDequeue

	// next and prev link to the adjacent poolChainElts in this
	// bufChain.
	//
	// next is written atomically by the producer and read
	// atomically by the consumer. It only transitions from nil to
	// non-nil.
	//
	// prev is written atomically by the consumer and read
	// atomically by the producer. It only transitions from
	// non-nil to nil.
	next, prev *bufChainElt
}

func storePoolChainElt(pp **bufChainElt, v *bufChainElt) {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(pp)), unsafe.Pointer(v))
}

func loadPoolChainElt(pp **bufChainElt) *bufChainElt {
	return (*bufChainElt)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(pp))))
}

func (c *bufChain) new(initSize int) {
	// Initialize the chain.
	// initSize must be a power of 2
	d := new(bufChainElt)
	d.vals = make([]unsafe.Pointer, initSize)
	storePoolChainElt(&c.head, d)
	storePoolChainElt(&c.tail, d)
}

func (c *bufChain) pushHead(val unsafe.Pointer) {
	for {
		d := loadPoolChainElt(&c.head)

		if d.pushHead(val) {
			return
		}

		// The current dequeue is full. Allocate a new one of twice
		// the size.
		if atomic.CompareAndSwapInt32(&c.chainStatus, 0, 1) {
			newSize := len(d.vals) * 2
			if newSize >= dequeueLimit {
				// Can't make it any bigger.
				newSize = dequeueLimit
			}

			d2 := &bufChainElt{prev: d}
			d2.vals = make([]unsafe.Pointer, newSize)
			storePoolChainElt(&c.head, d2)
			storePoolChainElt(&d.next, d2)
			d2.pushHead(val)
			atomic.SwapInt32(&c.chainStatus, 0)
		}
	}
}

func (c *bufChain) popTail() (unsafe.Pointer, bool) {
	d := loadPoolChainElt(&c.tail)
	if d == nil {
		return nil, false
	}

	for {
		// It's important that we load the next pointer
		// *before* popping the tail. In general, d may be
		// transiently empty, but if next is non-nil before
		// the pop and the pop fails, then d is permanently
		// empty, which is the only condition under which it's
		// safe to drop d from the chain.
		d2 := loadPoolChainElt(&d.next)

		if val, ok := d.popTail(); ok {
			return val, ok
		}

		if d2 == nil {
			// This is the only dequeue. It's empty right
			// now, but could be pushed to in the future.
			return nil, false
		}

		// The tail of the chain has been drained, so move on
		// to the next dequeue. Try to drop it from the chain
		// so the next pop doesn't have to look at the empty
		// dequeue again.
		if atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&c.tail)), unsafe.Pointer(d), unsafe.Pointer(d2)) {
			// We won the race. Clear the prev pointer so
			// the garbage collector can collect the empty
			// dequeue and so popHead doesn't back up
			// further than necessary.
			storePoolChainElt(&d2.prev, nil)
		}
		d = d2
	}
}
