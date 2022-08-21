package frame

import (
	"image"

	"github.com/rjkroege/edwood/draw"
)

func (f *frameimpl) drawtext(pt image.Point, text draw.Image, back draw.Image) {
	// log.Println("DrawText at", pt, "NoRedraw", f.NoRedraw, text)
	for _, b := range f.box {
		pt = f.cklinewrap(pt, b)
		// log.Printf("box [%d] %#v pt %v NoRedraw %v nrune %d\n",  nb, string(b.Ptr), pt, f.NoRedraw, b.Nrune)

		if !f.noredraw && b.Nrune >= 0 {
			f.background.Bytes(pt, text, image.Point{}, f.font, b.Ptr)
		}
		pt.X += b.Wid
	}
}

// drawBox is a helpful debugging utility that wraps each box with a
// rectangle to show its extent.
func (f *frameimpl) drawBox(r image.Rectangle, col, back draw.Image, qt image.Point) {
	f.background.Draw(r, col, nil, qt)
	r = r.Inset(1)
	f.background.Draw(r, back, nil, qt)
}

func (f *frameimpl) DrawSel(pt image.Point, p0, p1 int, highlighted bool) {
	// log.Printf("Frame.DrawSel start pt=%v p0=%d p1=%d highlighted=%v\n", pt, p0, p1, highlighted)
	// defer log.Println("Frame.DrawSel end")
	f.lk.Lock()
	defer f.lk.Unlock()
	f.drawselimpl(pt, p0, p1, highlighted)
}

func (f *frameimpl) drawselimpl(pt image.Point, p0, p1 int, highlighted bool) {
	// log.Println("Frame DrawSel Start", p0, p1, highlighted, f.sp0, f.sp1, f.ticked)
	// defer log.Println("Frame DrawSel End",  f.sp0, f.sp1, f.ticked)
	if p0 > p1 {
		panic("Drawsel0: p0 and p1 must be ordered")
	}

	if f.ticked {
		f.Tick(f.ptofcharptb(f.sp0, f.rect.Min, 0), false)
	}

	if f.sp0 != f.sp1 && f.highlighton {
		// Clear the selection so that subsequent code can
		// update correctly.
		back := f.cols[ColBack]
		text := f.cols[ColText]
		f.Drawsel0(f.ptofcharptb(f.sp0, f.rect.Min, 0), f.sp0, f.sp1, back, text)

		// Avoid multiple draws.
		f.highlighton = false
	}

	// We've already done everything necessary above if not
	// highlighting so simply return.
	if !highlighted {
		// This has to be updated here so that select can
		// correctly update the selection during a drag loop.
		f.sp0 = p0
		f.sp1 = p1
		return
	}

	// If we should just show the tick, do that and return.
	if p0 == p1 {
		f.Tick(pt, highlighted)
		f.display.Flush() // To show the tick.
		f.sp0 = p0
		f.sp1 = p1
		return
	}

	// Need to use the highlight colour.
	back := f.cols[ColHigh]
	text := f.cols[ColHText]

	f.Drawsel0(pt, p0, p1, back, text)
	f.sp0 = p0
	f.sp1 = p1
	f.highlighton = true
}

// TODO(rjk): This function is convoluted.
// Drawsel0 is a lower-level routine, taking as arguments a background
// color back and text color text. It assumes that the tick is being
// handled (removed beforehand, replaced afterwards, as required) by its
// caller. The selection is delimited by character positions p0 and p1.
// The point pt0 is the geometrical location of p0 on the screen and must
// be a value generated by Ptofchar.
//
// Commentary: this function should conceivably not be part of the public API
//
// TODO(rjk): Figure out if this is a true or false statement.
// Function does not mutate f.p0, f.p1 (well... actually, it does.)
func (f *frameimpl) Drawsel0(pt image.Point, p0, p1 int, back draw.Image, text draw.Image) image.Point {
	// log.Println("Frame Drawsel0 Start", p0, p1,  f.P0, f.P1)
	// defer log.Println("Frame Drawsel0 End", f.P0, f.P1 )
	p := 0
	trim := false
	x := 0

	if p0 > p1 {
		panic("Drawsel0: p0 and p1 must be ordered")
	}

	nb := 0
	var w int
	for ; nb < len(f.box) && p < p1; nb++ {
		b := f.box[nb]
		nr := nrune(b)
		if p+nr <= p0 {
			// This box doesn't need to be modified.
			p += nr
			continue
		}
		if p >= p0 {
			// Fills in the end of the previous line with selection highlight when the line has
			// has been wrapped.
			qt := pt
			pt = f.cklinewrap(pt, b)
			if pt.Y > qt.Y {
				if qt.X > f.rect.Max.X {
					qt.X = f.rect.Max.X
				}
				//f.drawBox(image.Rect(qt.X, qt.Y, f.Rect.Max.X, pt.Y), text, back,qt)
				f.background.Draw(image.Rect(qt.X, qt.Y, f.rect.Max.X, pt.Y), back, nil, qt)
			}
		}
		ptr := b.Ptr
		if p < p0 {
			// beginning of region: advance into box
			ptr = ptr[runeindex(ptr, p0-p):]
			nr -= p0 - p
			p = p0
		}
		trim = false
		if p+nr > p1 {
			// end of region: trim box
			nr -= (p + nr) - p1
			trim = true
		}

		if b.Nrune < 0 || nr == b.Nrune {
			w = b.Wid
		} else {
			w = f.font.BytesWidth(ptr[0:runeindex(ptr, nr)])
		}
		x = pt.X + w
		if x > f.rect.Max.X {
			x = f.rect.Max.X
		}
		// f.drawBox(image.Rect(pt.X, pt.Y, x, pt.Y+f.Font.DefaultHeight()), text, back, pt)
		f.background.Draw(image.Rect(pt.X, pt.Y, x, pt.Y+f.defaultfontheight), back, nil, pt)
		if b.Nrune >= 0 {
			f.background.Bytes(pt, text, image.Point{}, f.font, ptr[0:runeindex(ptr, nr)])
		}
		pt.X += w
		p += nr
	}

	if p1 > p0 && nb > 0 && nb < len(f.box) && f.box[nb-1].Nrune > 0 && !trim {
		qt := pt
		pt = f.cklinewrap(pt, f.box[nb])
		if pt.Y > qt.Y {
			f.drawBox(image.Rect(qt.X, qt.Y, f.rect.Max.X, pt.Y), f.cols[ColHigh], back, qt)
			// f.Background.Draw(image.Rect(qt.X, qt.Y, f.Rect.Max.X, pt.Y), back, nil, qt)
		}
	}

	return pt
}

// Redraw redraws the background of the Frame where the Frame is inside
// enclosing. Frame is responsible for drawing all of the pixels inside
// enclosing though may fill less than enclosing with text. (In particular,
// a margin may be added and the rectangle occupied by text is always
// a multiple of the fixed line height.)
// TODO(rjk): Modify this function to redraw the text as well and stop having
// the drawing of text strings be a side-effect of Insert, Delete, etc.
// TODO(rjk): Draw text to the bottom of enclosing as opposed to filling the
// bottom partial text row with blank.
//
// Note: this function is not part of the documented libframe entrypoints and
// was not invoked from Edwood code. Consequently, I am repurposing the name.
// Future changes will have this function able to clear the Frame and draw the
// entire box model.
func (f *frameimpl) Redraw(enclosing image.Rectangle) {
	f.lk.Lock()
	defer f.lk.Unlock()
	// log.Printf("Redraw %v %v", f.Rect, enclosing)
	f.background.Draw(enclosing, f.cols[ColBack], nil, image.Point{})
}

func (f *frameimpl) tick(pt image.Point, ticked bool) {
	//	log.Println("_tick")
	if f.ticked == ticked || f.tickimage == nil || !pt.In(f.rect) {
		return
	}

	pt.X -= f.tickscale
	r := image.Rect(pt.X, pt.Y, pt.X+frtickw*f.tickscale, pt.Y+f.defaultfontheight)

	if r.Max.X > f.rect.Max.X {
		r.Max.X = f.rect.Max.X
	}

	if ticked {
		f.tickback.Draw(f.tickback.R(), f.background, nil, pt)
		f.background.Draw(r, f.display.Black(), f.tickimage, image.Point{}) // draws an alpha-blended box
	} else {
		// There is an issue with tick management
		f.background.Draw(r, f.tickback, nil, image.Point{})
	}
	f.ticked = ticked
}

// Tick draws (if up is non-zero) or removes (if up is zero) the tick
// at the screen position indicated by pt.
//
// Commentary: because this code ignores selections, it is conceivably
// undesirable to use it in the public API.
func (f *frameimpl) Tick(pt image.Point, ticked bool) {
	if f.tickscale != f.display.ScaleSize(1) {
		if f.ticked {
			f.tick(pt, false)
		}
		f.InitTick()
	}

	f.tick(pt, ticked)
}

func (f *frameimpl) _draw(pt image.Point) image.Point {
	// f.Logboxes("_draw -- start")
	for nb := 0; nb < len(f.box); nb++ {
		b := f.box[nb]
		if b == nil {
			f.Logboxes("-- Frame._draw has invalid box mode --")
			panic("-- Frame._draw has invalid box mode --")
		}
		pt = f.cklinewrap0(pt, b)
		if pt.Y == f.rect.Max.Y {
			f.lastlinefull = true
			f.nchars -= f.strlen(nb)
			f.delbox(nb, len(f.box)-1)
			break
		}

		if b.Nrune > 0 {
			n, fits := f.canfit(pt, b)
			if !fits {
				break
			}
			if n != b.Nrune {
				f.splitbox(nb, n)
				b = f.box[nb]
			}
			pt.X += b.Wid
		} else {
			if b.Bc == '\n' {
				pt.X = f.rect.Min.X
				pt.Y += f.defaultfontheight
			} else {
				pt.X += f.newwid(pt, b)
			}
		}
	}
	// f.Logboxes("_draw -- end")
	return pt
}

func (f *frameimpl) strlen(nb int) int {
	n := 0
	for _, b := range f.box[nb:] {
		n += nrune(b)
	}
	return n
}
