//go:build !onnx

package inference

// NewRunner (stub build) returns a disabled runner. The default build has no
// CGO/ONNX Runtime dependency, so the binary stays static and portable. Build
// with `-tags onnx` (and CGO_ENABLED=1) to compile the real backend.
func NewRunner(cfg Config) (Runner, error) {
	return disabledRunner{err: ErrUnavailable}, nil
}
