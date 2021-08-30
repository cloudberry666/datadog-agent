// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package containermeta

// Filter allows a subscriber to filter events by entity kind or event source.
type Filter struct {
	// TODO(juliogreff): making these into a map[Kind|string]struct{} will yield
	// better performance
	Kinds   []Kind
	Sources []string
}

// MatchKind returns true if the filter matches the passed Kind.
func (f *Filter) MatchKind(k Kind) bool {
	if f == nil || len(f.Kinds) == 0 {
		return true
	}

	for _, fk := range f.Kinds {
		if fk == k {
			return true
		}
	}

	return false
}

// MatchSource returns true if the filter matches the passed source.
func (f *Filter) MatchSource(s string) bool {
	if f == nil || len(f.Sources) == 0 {
		return true
	}

	for _, fs := range f.Sources {
		if fs == s {
			return true
		}
	}

	return false
}

// Match returns true if the filter matches an event.
func (f *Filter) Match(ev Event) bool {
	if f == nil {
		return true
	}

	return f.MatchKind(ev.Entity.GetID().Kind) && f.MatchSource(ev.Source)
}
