package bngsocket

func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		data: make([]byte, size),
		size: size,
	}
}

func (rb *RingBuffer) Write(data []byte) {
	for _, b := range data {
		rb.data[rb.head] = b
		rb.head = (rb.head + 1) % rb.size
	}
}

func (rb *RingBuffer) Read(n int) []byte {
	result := make([]byte, n)
	for i := 0; i < n; i++ {
		result[i] = rb.data[rb.tail]
		rb.tail = (rb.tail + 1) % rb.size
	}
	return result
}

func (rb *RingBuffer) Available() int {
	if rb.head >= rb.tail {
		return rb.head - rb.tail
	}
	return rb.size - rb.tail + rb.head
}

func (rb *RingBuffer) Reset() {
	rb.head = 0
	rb.tail = 0
}
