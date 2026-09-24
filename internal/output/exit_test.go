package output

import (
	"errors"
	"fmt"
	"testing"
)

type fakeExitCoder struct{ code int }

func (f *fakeExitCoder) Error() string { return "boom" }
func (f *fakeExitCoder) ExitCode() int { return f.code }

func TestCodeFor(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"nil maps to OK", nil, ExitOK},
		{"plain error maps to generic", errors.New("boom"), ExitError},
		{"coded error", WithCode(errors.New("boom"), ExitConfig), ExitConfig},
		{"exit coder", &fakeExitCoder{ExitNetwork}, ExitNetwork},
		{"wrapped coded error", fmt.Errorf("wrap: %w", WithCode(errors.New("boom"), ExitAuth)), ExitAuth},
		{"wrapped exit coder", fmt.Errorf("wrap: %w", &fakeExitCoder{ExitNotFound}), ExitNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CodeFor(tt.err); got != tt.want {
				t.Errorf("CodeFor(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}

func TestWithCodeNil(t *testing.T) {
	if err := WithCode(nil, ExitConfig); err != nil {
		t.Errorf("WithCode(nil) = %v, want nil", err)
	}
}

func TestCodedErrorUnwrap(t *testing.T) {
	base := errors.New("boom")
	err := WithCode(base, ExitConfig)
	if !errors.Is(err, base) {
		t.Error("WithCode error should unwrap to its base error")
	}
	if got := CodeFor(err); got != ExitConfig {
		t.Errorf("CodeFor = %d, want %d", got, ExitConfig)
	}
}
