package linux

import (
	"worker/internal/worker/store"
)

type OutputWriter struct {
	jobId string
	store store.Store
}

func NewWrite(store store.Store, jobId string) *OutputWriter {
	return &OutputWriter{store: store, jobId: jobId}
}

// Write implements the io.Writer interface
func (w *OutputWriter) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	// Create a copy of the data to prevent races
	// The underlying buffer p might be reused by the caller
	chunk := make([]byte, len(p))
	copy(chunk, p)

	w.store.WriteToBuffer(w.jobId, chunk)

	// Return the number of bytes written (always successful)
	return len(p), nil
}
