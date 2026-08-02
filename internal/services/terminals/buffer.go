package terminals

// TODO: Review properly

type outputBuffer struct {
	data  []byte
	limit int
}

func (b *outputBuffer) Write(data []byte) {
	b.data = append(b.data, data...)
	if len(b.data) <= b.limit {
		return
	}

	b.data = append([]byte(nil), b.data[len(b.data)-b.limit:]...)
}

func (b *outputBuffer) Bytes() []byte {
	return append([]byte(nil), b.data...)
}

func newOutputBuffer(limit int) outputBuffer {
	return outputBuffer{
		data:  make([]byte, 0, limit),
		limit: limit,
	}
}
