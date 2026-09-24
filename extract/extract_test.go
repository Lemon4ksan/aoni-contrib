// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package extract_test

import (
	"regexp"
	"slices"
	"testing"

	"github.com/lemon4ksan/foundation/generic"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"

	"github.com/lemon4ksan/aoni-contrib/extract"
)

func TestBetween(t *testing.T) {
	t.Parallel()

	src := []byte("prefix:hello world:suffix")

	res, err := extract.Between(src, "prefix:", ":suffix")
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(res))

	resResult := generic.ToResult(extract.Between(src, "prefix:", ":suffix"))
	require.True(t, resResult.IsSuccess())
	assert.Equal(t, "hello world", string(resResult.MustValue()))

	_, err = extract.Between(src, "missing:", ":suffix")
	assert.ErrorIs(t, err, extract.ErrBetweenNotFound)

	_, err = extract.Between(src, "prefix:", ":missing")
	assert.ErrorIs(t, err, extract.ErrBetweenNotFound)

	sVal := extract.BetweenString(src, "prefix:", ":suffix")
	require.True(t, sVal.IsSuccess())
	assert.Equal(t, "hello world", sVal.MustValue())

	optVal := extract.BetweenOptional(src, "prefix:", ":suffix")
	assert.True(t, optVal.IsPresent())
	assert.Equal(t, "hello world", optVal.MustValue())

	optMissing := extract.BetweenOptional(src, "missing:", ":suffix")
	assert.False(t, optMissing.IsPresent())

	// Edge cases: empty prefix and empty suffix
	resNoPrefix, err := extract.Between(src, "", ":suffix")
	require.NoError(t, err)
	assert.Equal(t, "prefix:hello world", string(resNoPrefix))

	resNoSuffix, err := extract.Between(src, "prefix:", "")
	require.NoError(t, err)
	assert.Equal(t, "hello world:suffix", string(resNoSuffix))
}

func TestAttr(t *testing.T) {
	t.Parallel()

	src := []byte(`<div id="test-id" data-token="secret-123" class="main"></div>`)

	val, err := extract.Attr(src, "#test-id", "data-token")
	require.NoError(t, err)
	assert.Equal(t, "secret-123", string(val))

	attrRes := generic.ToResult(extract.Attr(src, "#test-id", "data-token"))
	require.True(t, attrRes.IsSuccess())
	assert.Equal(t, "secret-123", string(attrRes.MustValue()))

	_, err = extract.Attr(src, "#missing-id", "data-token")
	assert.ErrorIs(t, err, extract.ErrElementNotFound)

	_, err = extract.Attr(src, "#test-id", "missing-attr")
	assert.ErrorIs(t, err, extract.ErrAttrNotFound)

	sAttr := extract.AttrString(src, "#test-id", "data-token")
	require.True(t, sAttr.IsSuccess())
	assert.Equal(t, "secret-123", sAttr.MustValue())

	optAttr := extract.AttrOptional(src, "#test-id", "data-token")
	assert.True(t, optAttr.IsPresent())
	assert.Equal(t, "secret-123", optAttr.MustValue())

	optMissing := extract.AttrOptional(src, "#test-id", "missing-attr")
	assert.False(t, optMissing.IsPresent())

	// Single quote id and attr
	singleQuoteSrc := []byte("<div id='single' attr='val'></div>")
	sqVal, err := extract.Attr(singleQuoteSrc, "#single", "attr")
	require.NoError(t, err)
	assert.Equal(t, "val", string(sqVal))

	// No selector (raw extraction)
	rawVal, err := extract.Attr(singleQuoteSrc, "", "attr")
	require.NoError(t, err)
	assert.Equal(t, "val", string(rawVal))
}

func TestRegex(t *testing.T) {
	t.Parallel()

	src := []byte("User session token: abcdef12345; Expires: tomorrow")

	val, err := extract.Regex(src, `token:\s*([a-z0-9]+);`)
	require.NoError(t, err)
	assert.Equal(t, "abcdef12345", string(val))

	regRes := extract.RegexResult(src, `token:\s*([a-z0-9]+);`)
	require.True(t, regRes.IsSuccess())
	assert.Equal(t, "abcdef12345", string(regRes.MustValue()))

	sRegex := extract.RegexString(src, `token:\s*([a-z0-9]+);`)
	require.True(t, sRegex.IsSuccess())
	assert.Equal(t, "abcdef12345", sRegex.MustValue())

	optRegex := extract.RegexOptional(src, `token:\s*([a-z0-9]+);`)
	assert.True(t, optRegex.IsPresent())
	assert.Equal(t, "abcdef12345", optRegex.MustValue())

	// No capture groups, falls back to full match
	valFull, err := extract.Regex(src, `Expires: tomorrow`)
	require.NoError(t, err)
	assert.Equal(t, "Expires: tomorrow", string(valFull))

	// Mismatch
	_, errMismatch := extract.Regex(src, `not_found_pattern`)
	assert.ErrorIs(t, errMismatch, extract.ErrRegexMismatch)

	// Invalid regex syntax
	_, errInvalid := extract.Regex(src, `(?P<invalid`)
	assert.Error(t, errInvalid)
}

func TestBetweenAll(t *testing.T) {
	t.Parallel()

	src := []byte("<item>first</item><item>second</item><item>third</item>")

	var items []string

	for item := range extract.BetweenAll(src, "<item>", "</item>") {
		items = append(items, string(item))
	}

	assert.Equal(t, []string{"first", "second", "third"}, items)

	// Empty boundaries exit immediately
	var emptyItems []string

	for item := range extract.BetweenAll(src, "", "") {
		emptyItems = append(emptyItems, string(item))
	}

	assert.Empty(t, emptyItems)
}

func TestBetweenAllString(t *testing.T) {
	t.Parallel()

	src := "<li>apple</li><li>banana</li><li>cherry</li>"

	var fruits []string

	for fruit := range extract.BetweenAllString(src, "<li>", "</li>") {
		fruits = append(fruits, fruit)
	}

	assert.Equal(t, []string{"apple", "banana", "cherry"}, fruits)
}

func TestRegexAll(t *testing.T) {
	t.Parallel()

	rx := regexp.MustCompile(`item-(\d+)`)
	src := []byte("item-10 item-20 item-30")

	var matches []string

	for m := range extract.RegexAll(src, rx) {
		matches = append(matches, string(m))
	}

	assert.Equal(t, []string{"10", "20", "30"}, matches)

	// Nil rx returns immediately
	var nilMatches []string

	for m := range extract.RegexAll(src, nil) {
		nilMatches = append(nilMatches, string(m))
	}

	assert.Empty(t, nilMatches)
}

func TestRegexAllSubmatch(t *testing.T) {
	t.Parallel()

	rx := regexp.MustCompile(`([a-z]+)=(\d+)`)
	src := []byte("a=1 b=2")

	var entries []string

	for sm := range extract.RegexAllSubmatch(src, rx) {
		entries = append(entries, string(sm[1])+"->"+string(sm[2]))
	}

	assert.Equal(t, []string{"a->1", "b->2"}, entries)

	var nilEntries [][]byte

	for sm := range extract.RegexAllSubmatch(src, nil) {
		nilEntries = append(nilEntries, sm[0])
	}

	assert.Empty(t, nilEntries)
}

func TestAttrsAll(t *testing.T) {
	t.Parallel()

	src := []byte(`<a href="/page1"></a><b href='/page2'></b><c href="/page3"></c>`)

	var hrefs []string

	for h := range extract.AttrsAll(src, "href") {
		hrefs = append(hrefs, string(h))
	}

	assert.True(t, slices.Equal([]string{"/page1", "/page2", "/page3"}, hrefs))

	// Empty attr name
	var emptyHrefs []string

	for h := range extract.AttrsAll(src, "") {
		emptyHrefs = append(emptyHrefs, string(h))
	}

	assert.Empty(t, emptyHrefs)
}
