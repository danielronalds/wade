package terminals

import "testing"

func TestOutputBufferReturnsDefensiveCopy(t *testing.T) {
	buffer := newOutputBuffer(4)
	buffer.Write([]byte("abcdef"))
	got := buffer.Bytes()
	if string(got) != "cdef" {
		t.Fatalf("Bytes() = %q, want cdef", got)
	}
	got[0] = 'x'
	if string(buffer.Bytes()) != "cdef" {
		t.Fatal("Bytes() exposed the internal buffer")
	}
}

func TestControlMessageValidation(t *testing.T) {
	resize, ok := parseControlMessage([]byte(`{"type":"resize","cols":120,"rows":40}`))
	if !ok || !resize.IsResize() || resize.IsActivate() {
		t.Fatalf("resize = %#v, parsed = %v", resize, ok)
	}
	activate, ok := parseControlMessage([]byte(`{"type":"activate"}`))
	if !ok || !activate.IsActivate() || activate.IsResize() {
		t.Fatalf("activate = %#v, parsed = %v", activate, ok)
	}
}
