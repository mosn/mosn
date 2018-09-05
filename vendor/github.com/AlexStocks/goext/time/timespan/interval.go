// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// Package gxtime encapsulates some golang.time functions
// refers from https://github.com/senseyeio/spaniel/blob/master/interval.go
package gxtimespan

import (
	"sort"
	"time"
)

// EndPointType represents whether the start or end of an interval is Closed or Open.
type EndPointType int

const (
	// Open means that the interval does not include a value
	Open EndPointType = iota
	// Closed means that the interval does include a value
	Closed
)

// EndPoint represents an extreme of an interval, and whether it is inclusive or exclusive (Closed or Open)
type EndPoint struct {
	Element time.Time
	Type    EndPointType
}

// Span represents a basic span, with a start and end time.
type Span interface {
	Start() time.Time
	StartType() EndPointType
	End() time.Time
	EndType() EndPointType
}

// Spans represents a list of spans, on which other functions operate.
type Spans []Span

// ByStart sorts a list of spans by their start point
type ByStart Spans

func (s ByStart) Len() int           { return len(s) }
func (s ByStart) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s ByStart) Less(i, j int) bool { return s[i].Start().Before(s[j].Start()) }

// UnionHandlerFunc is used by UnionWithHandler to allow for custom functionality when two spans are merged.
// It is passed the two spans to be merged, and span which will result from the union.
type UnionHandlerFunc func(mergeInto, mergeFrom, mergeSpan Span) Span

// IntersectionHandlerFunc is used by IntersectionWithHandler to allow for custom functionality when two spans
// intersect. It is passed the two spans that intersect, and span representing the intersection.
type IntersectionHandlerFunc func(intersectingEvent1, intersectingEvent2, intersectionSpan Span) Span

func getLoosestIntervalType(x, y EndPointType) EndPointType {
	if x > y {
		return x
	}
	return y
}

func getTightestIntervalType(x, y EndPointType) EndPointType {
	if x < y {
		return x
	}
	return y
}

func getMin(a, b EndPoint) EndPoint {
	if a.Element.Before(b.Element) {
		return a
	}
	return b
}

func getMax(a, b EndPoint) EndPoint {
	if a.Element.After(b.Element) {
		return a
	}
	return b
}

func filter(spans Spans, filterFunc func(Span) bool) Spans {
	filtered := Spans{}
	for _, span := range spans {
		if !filterFunc(span) {
			filtered = append(filtered, span)
		}
	}
	return filtered
}

// IsInstant returns true if the interval is deemed instantaneous
func IsInstant(a Span) bool {
	return a.Start().Equal(a.End())
}

// Returns true if two spans are side by side
func contiguous(a, b Span) bool {
	// [1,2,3,4] [4,5,6,7] - not contiguous
	// [1,2,3,4) [4,5,6,7] - contiguous
	// [1,2,3,4] (4,5,6,7] - contiguous
	// [1,2,3,4) (4,5,6,7] - not contiguous
	// [1,2,3] [5,6,7] - not contiguous
	// [1] (1,2,3] - contiguous

	// Two instants can't be contiguous
	if IsInstant(a) && IsInstant(b) {
		return false
	}

	if b.Start().Before(a.Start()) {
		a, b = b, a
	}

	aStartType := a.StartType()
	aEndType := a.EndType()
	bStartType := b.StartType()

	if IsInstant(a) {
		aEndType = Closed
		aStartType = Closed
	}
	if IsInstant(b) {
		bStartType = Closed
	}

	// If a and b start at the same point, just check that their start types are different.
	if a.Start().Equal(b.Start()) {
		return aStartType != bStartType
	}

	// To be contiguous the ranges have to overlap on the first/last point
	if !(a.End().Equal(b.Start())) {
		return false
	}

	if aEndType == bStartType {
		return false
	}
	return true
}

// Returns true if two spans overlap
func overlap(a, b Span) bool {
	// [1,2,3,4] [4,5,6,7] - intersects
	// [1,2,3,4) [4,5,6,7] - doesn't intersect
	// [1,2,3,4] (4,5,6,7] - doesn't intersect
	// [1,2,3,4) (4,5,6,7] - doesn't intersect

	aStartType := a.StartType()
	aEndType := a.EndType()
	bStartType := b.StartType()
	bEndType := b.EndType()

	if IsInstant(a) {
		aStartType = Closed
		aEndType = Closed
	}
	if IsInstant(b) {
		bStartType = Closed
		bEndType = Closed
	}

	// Given [a_s,a_e] and [b_s,b_e]
	// If a_s > b_e || a_e < b_s, overlap == false

	c1 := false // is a_s after b_e
	if a.Start().After(b.End()) {
		c1 = true
	} else if a.Start().Equal(b.End()) {
		c1 = (aStartType == Open || bEndType == Open)
	}

	c2 := false // is a_e before b_s
	if a.End().Before(b.Start()) {
		c2 = true
	} else if a.End().Equal(b.Start()) {
		c2 = (aEndType == Open || bStartType == Open)
	}

	if c1 || c2 {
		return false
	}

	return true
}

// UnionWithHandler returns a list of Spans representing the union of all of the spans.
// For example, given a list [A,B] where A and B overlap, a list [C] would be returned, with the span C spanning
// both A and B. The provided handler is passed the source and destination spans, and the currently merged empty span.
func (s Spans) UnionWithHandler(unionHandlerFunc UnionHandlerFunc) Spans {

	if len(s) < 2 {
		return s
	}

	var sorted Spans
	sorted = append(sorted, s...)
	sort.Stable(ByStart(sorted))

	result := Spans{sorted[0]}

	for _, b := range sorted[1:] {
		// A: current span in merged array; B: current span in sorted array
		// If B overlaps with A, it can be merged with A.
		a := result[len(result)-1]
		if overlap(a, b) || contiguous(a, b) {

			spanStart := getMin(EndPoint{a.Start(), a.StartType()}, EndPoint{b.Start(), b.StartType()})
			spanEnd := getMax(EndPoint{a.End(), a.EndType()}, EndPoint{b.End(), b.EndType()})

			if a.Start().Equal(b.Start()) {
				spanStart.Type = getLoosestIntervalType(a.StartType(), b.StartType())
			}
			if a.End().Equal(b.End()) {
				spanEnd.Type = getLoosestIntervalType(a.EndType(), b.EndType())
			}

			span := NewWithTypes(spanStart.Element, spanEnd.Element, spanStart.Type, spanEnd.Type)
			result[len(result)-1] = unionHandlerFunc(a, b, span)

			continue
		}
		result = append(result, b)
	}

	return result
}

// Union returns a list of Spans representing the union of all of the spans.
// For example, given a list [A,B] where A and B overlap, a list [C] would be returned, with the span C spanning
// both A and B.
func (s Spans) Union() Spans {
	return s.UnionWithHandler(func(mergeInto, mergeFrom, mergeSpan Span) Span {
		return mergeSpan
	})
}

// IntersectionWithHandler returns a list of Spans representing the overlaps between the contained spans.
// For example, given a list [A,B] where A and B overlap, a list [C] would be returned, with the span C covering
// the intersection of the A and B. The provided handler function is notified of the two spans that have been found
// to overlap, and the span representing the overlap.
func (s Spans) IntersectionWithHandler(intersectHandlerFunc IntersectionHandlerFunc) Spans {
	var sorted Spans
	sorted = append(sorted, s...)
	sort.Stable(ByStart(sorted))

	actives := Spans{sorted[0]}

	intersections := Spans{}

	for _, b := range sorted[1:] {
		// Tidy up the active span list
		actives = filter(actives, func(t Span) bool {
			// If this value is identical to one in actives, don't filter it.
			if b.Start() == t.Start() && b.End() == t.End() {
				return false
			}
			// If this value starts after the one in actives finishes, filter the active.
			return b.Start().After(t.End())
		})

		for _, a := range actives {
			if overlap(a, b) {
				spanStart := getMax(EndPoint{a.Start(), a.StartType()}, EndPoint{b.Start(), b.StartType()})
				spanEnd := getMin(EndPoint{a.End(), a.EndType()}, EndPoint{b.End(), b.EndType()})

				if a.Start().Equal(b.Start()) {
					spanStart.Type = getTightestIntervalType(a.StartType(), b.StartType())
				}
				if a.End().Equal(b.End()) {
					spanEnd.Type = getTightestIntervalType(a.EndType(), b.EndType())
				}
				span := NewWithTypes(spanStart.Element, spanEnd.Element, spanStart.Type, spanEnd.Type)
				intersection := intersectHandlerFunc(a, b, span)
				intersections = append(intersections, intersection)
			}
		}
		actives = append(actives, b)
	}
	return intersections
}

// Intersection returns a list of Spans representing the overlaps between the contained spans.
// For example, given a list [A,B] where A and B overlap, a list [C] would be returned,
// with the span C covering the intersection of A and B.
func (s Spans) Intersection() Spans {
	return s.IntersectionWithHandler(func(intersectingEvent1, intersectingEvent2, intersectionSpan Span) Span {
		return intersectionSpan
	})
}
