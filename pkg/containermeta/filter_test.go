// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package containermeta

import "testing"

const (
	fooSource = "foo"
	barSource = "bar"
)

func TestFilterMatch(t *testing.T) {
	ev := Event{
		Source: fooSource,
		Entity: EntityID{
			Kind: KindContainer,
		},
	}

	tests := []struct {
		name     string
		filter   *Filter
		event    Event
		expected bool
	}{
		{
			name:     "nil filter",
			filter:   nil,
			event:    ev,
			expected: true,
		},

		{
			name: "matching single kind",
			filter: &Filter{
				Kinds: []Kind{KindContainer},
			},
			event:    ev,
			expected: true,
		},
		{
			name: "matching one of kinds",
			filter: &Filter{
				Kinds: []Kind{KindContainer, KindKubernetesPod},
			},
			event:    ev,
			expected: true,
		},
		{
			name: "matching no kind",
			filter: &Filter{
				Kinds: []Kind{KindKubernetesPod},
			},
			event:    ev,
			expected: false,
		},

		{
			name: "matching single source",
			filter: &Filter{
				Sources: []string{fooSource},
			},
			event:    ev,
			expected: true,
		},
		{
			name: "matching one of sources",
			filter: &Filter{
				Sources: []string{fooSource, barSource},
			},
			event:    ev,
			expected: true,
		},
		{
			name: "matching no source",
			filter: &Filter{
				Sources: []string{barSource},
			},
			event:    ev,
			expected: false,
		},

		{
			name: "matching source but not kind",
			filter: &Filter{
				Kinds:   []Kind{KindKubernetesPod},
				Sources: []string{fooSource},
			},
			event:    ev,
			expected: false,
		},
		{
			name: "matching kind but not source",
			filter: &Filter{
				Kinds:   []Kind{KindContainer},
				Sources: []string{barSource},
			},
			event:    ev,
			expected: false,
		},
		{
			name: "matching both kind and source",
			filter: &Filter{
				Kinds:   []Kind{KindContainer},
				Sources: []string{fooSource},
			},
			event:    ev,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.filter.Match(tt.event)
			if actual != tt.expected {
				t.Errorf("expected filter.Match() to be %t, got %t instead", tt.expected, actual)
			}
		})
	}
}
