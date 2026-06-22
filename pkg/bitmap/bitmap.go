package bitmap

import (
	"github.com/RoaringBitmap/roaring/v2"
)

type Bitmap struct {
	rb *roaring.Bitmap
}

func New() *Bitmap {
	return &Bitmap{rb: roaring.New()}
}

func (b *Bitmap) Add(id uint32) {
	b.rb.Add(id)
}

func (b *Bitmap) Remove(id uint32) {
	b.rb.Remove(id)
}

func (b *Bitmap) Contains(id uint32) bool {
	return b.rb.Contains(id)
}

func (b *Bitmap) And(other *Bitmap) *Bitmap {
	result := roaring.New()
	result.Or(b.rb)
	result.And(other.rb)
	return &Bitmap{rb: result}
}

func (b *Bitmap) Or(other *Bitmap) *Bitmap {
	result := roaring.New()
	result.Or(b.rb)
	result.Or(other.rb)
	return &Bitmap{rb: result}
}

func (b *Bitmap) Clone() *Bitmap {
	return &Bitmap{rb: b.rb.Clone()}
}

func (b *Bitmap) Cardinality() uint64 {
	return b.rb.GetCardinality()
}

func (b *Bitmap) ToArray() []uint32 {
	return b.rb.ToArray()
}

func (b *Bitmap) IsEmpty() bool {
	return b.rb.IsEmpty()
}
