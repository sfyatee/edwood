package frame

import (
	"image"
	"log"
	"unicode/utf8"
)

// canfit measures the b's string contents and determines if it fits
// in the region of the screen between pt and the right edge of the
// text-containing region. Returned values have several cases.
//
// If b has width, returns the index of the first  rune known
// to not fit and true if more thna 0 runes fit.
// If b has no width, use minwidth instead of width.
func (f *Frame) canfit(pt image.Point, b *frbox) (int, bool) {
	left := f.Rect.Max.X - pt.X
	if b.Nrune < 0 {
		if int(b.Minwid) <= left {
			return 1, true
		} else {
			return 0, false
		}
	}

	if left > b.Wid {
		return b.Nrune, (b.Nrune != 0)
	}

	w := 0
	o := 0
	for nr := 0; nr < b.Nrune; nr++ {
		_, w = utf8.DecodeRune(b.Ptr[o:])
		left -= f.Font.StringWidth(string(b.Ptr[o : o+w]))
		if left < 0 {
			return nr, nr != 0
		}
		o += w
	}
	return 0, false
}

func (f *Frame) cklinewrap(p image.Point, b *frbox) (ret image.Point) {
	ret = p
	if b.Nrune < 0 {
		if int(b.Minwid) > f.Rect.Max.X-p.X {
			ret.X = f.Rect.Min.X
			ret.Y = p.Y + f.Font.DefaultHeight()
		}
	} else {
		if b.Wid > f.Rect.Max.X-p.X {
			ret.X = f.Rect.Min.X
			ret.Y = p.Y + f.Font.DefaultHeight()
		}
	}
	if ret.Y > f.Rect.Max.Y {
		ret.Y = f.Rect.Max.Y
	}
	return ret
}

func (f *Frame) cklinewrap0(p image.Point, b *frbox) (ret image.Point) {
	ret = p
	if _, ok := f.canfit(p, b); !ok {
		ret.X = f.Rect.Min.X
		ret.Y = p.Y + f.Font.DefaultHeight()
		if ret.Y > f.Rect.Max.Y {
			ret.Y = f.Rect.Max.Y
		}
	}
	return ret
}

func (f *Frame) advance(p *image.Point, b *frbox) {
	if b.Nrune < 0 && b.Bc == '\n' {
		p.X = f.Rect.Min.X
		p.Y += f.Font.DefaultHeight()
	} else {
		p.X += b.Wid
	}
}

func (f *Frame) newwid(pt image.Point, b *frbox) int {
	b.Wid = f.newwid0(pt, b)
	return b.Wid
}

func (f *Frame) newwid0(pt image.Point, b *frbox) int {
	c := f.Rect.Max.X
	x := pt.X
	if b.Nrune >= 0 || b.Bc != '\t' {
		return b.Wid
	}
	if x+int(b.Minwid) > c {
		pt.X = f.Rect.Min.X
		x = pt.X
	}
	x += f.MaxTab
	x -= (x - f.Rect.Min.X) % f.MaxTab
	if x-pt.X < int(b.Minwid) || x > c {
		x = pt.X + int(b.Minwid)
	}
	return x - pt.X
}

// TODO(rjk): broken. does not fix up the world correctly?
// clean merges boxes where possible over boxes [n0, n1)
func (f *Frame) clean(pt image.Point, n0, n1 int) {
	//log.Println("clean", pt, n0, n1, f.Rect.Max.X)
	//	f.Logboxes("--- clean: starting ---")
	c := f.Rect.Max.X
	nb := 0
	for nb = n0; nb < n1-1; nb++ {
		b := f.box[nb]
		pt = f.cklinewrap(pt, b)
		for f.box[nb].Nrune >= 0 &&
			nb < n1-1 &&
			f.box[nb+1].Nrune >= 0 &&
			pt.X+f.box[nb].Wid+f.box[nb+1].Wid < c {
			f.mergebox(nb)
			n1--
			b = f.box[nb]
		}
		f.advance(&pt, f.box[nb])
	}

	for ; nb < f.nbox; nb++ {
		b := f.box[nb]
		pt = f.cklinewrap(pt, b)
		f.advance(&pt, f.box[nb])
	}
	f.LastLineFull = 0
	if pt.Y >= f.Rect.Max.Y {
		f.LastLineFull = 1
	}
	//	f.Logboxes("--- clean: end")
}

func nbyte(f *frbox) int {
	return len(f.Ptr)
}

func nrune(b *frbox) int {
	if b.Nrune < 0 {
		return 1
	} else {
		return b.Nrune
	}
}

func Rpt(min, max image.Point) image.Rectangle {
	return image.Rectangle{Min: min, Max: max}
}

// Logboxes shows the box model to the log for debugging convenience.
func (f *Frame) Logboxes(message string, args ...interface{}) {
	log.Printf(message, args...)
	log.Printf("nbox=%d nalloc=%d", f.nbox, f.nalloc)
	for i, b := range f.box {
		if b != nil {
			if b.Nrune == -1 && b.Bc == '\n' {
				log.Printf("	box[%d] -> newline\n")
			} else if b.Nrune == -1 && b.Bc == '\t' {
				log.Printf("	box[%d] -> tab\n")
			} else {
				log.Printf("	box[%d] -> %#v width %d\n", i, string(b.Ptr), b.Wid)
			}
		}
	}
}
