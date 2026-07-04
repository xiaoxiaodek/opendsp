package encoder

import (
	"context"

	"github.com/opendsp/opendsp/internal/mockadx/generator"
)

type BidResponseSpec struct {
	RequestID string
	BidID     string
	ImpID     string
	Price     float64
	CrID      string
	Adm       string
}

type Encoder interface {
	ContentType() string
	Encode(ctx context.Context, spec *generator.BidRequestSpec) ([]byte, error)
	Decode(raw []byte) (*BidResponseSpec, error)
}