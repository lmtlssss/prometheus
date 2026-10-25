package remote

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

// shortFrameWriter deliberately violates io.Writer's short-write contract.
// The codec must fail closed rather than report success for an incomplete frame.
type shortFrameWriter struct {
	dest bytes.Buffer
	shortAt int
	calls int
}

func (w *shortFrameWriter) Write(p []byte) (int, error) {
	w.calls++
	if w.calls == w.shortAt {
		if len(p) == 0 {
			return 0, nil
		}
		return w.dest.Write(p[:len(p)-1])
	}
	return w.dest.Write(p)
}

func TestChunkedWriterRejectsShortFrameWrites(t *testing.T) {
	for _, tc := range []struct{name string; call int}{
		{"length-delimiter", 1},
		{"payload", 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := &shortFrameWriter{shortAt: tc.call}
			cw := NewChunkedWriter(w, &mockedFlusher{})
			n, err := cw.Write([]byte("hello"))
			require.ErrorIs(t, err, io.ErrShortWrite)
			if tc.call == 1 {
				require.Zero(t, n)
				require.Equal(t, 1, w.calls)
			} else {
				require.Equal(t, 4, n)
				require.Equal(t, 3, w.calls)
			}
		})
	}
}

func TestChunkedWriterFullFrameStillRoundTrips(t *testing.T) {
	w := &shortFrameWriter{}
	cw := NewChunkedWriter(w, &mockedFlusher{})
	msg := []byte("complete-frame")
	n, err := cw.Write(msg)
	require.NoError(t, err)
	require.Equal(t, len(msg), n)
	got, err := NewChunkedReader(&w.dest, 100, nil).Next()
	require.NoError(t, err)
	require.Equal(t, msg, got)
}
