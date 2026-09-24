// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package extract_test

import (
	"testing"

	"github.com/lemon4ksan/aoni-contrib/extract"
)

func FuzzExtract(f *testing.F) {
	f.Add(
		[]byte(`<div class="target" data-token="secret123">hello &amp; world</div>`),
		`<div class="target" data-token="`,
		`"`,
		`#target`,
		`data-token`,
	)
	f.Add([]byte(``), ``, ``, ``, ``)
	f.Add([]byte(`prefix middle suffix`), `prefix `, ` suffix`, `a`, `href`)

	f.Fuzz(func(t *testing.T, src []byte, prefix, suffix, css, attrName string) {
		_, _ = extract.Between(src, prefix, suffix)
		_ = extract.BetweenResult(src, prefix, suffix)
		_ = extract.BetweenString(src, prefix, suffix)
		_ = extract.BetweenOptional(src, prefix, suffix)

		if css != "" && attrName != "" {
			_, _ = extract.Attr(src, css, attrName)
			_ = extract.AttrResult(src, css, attrName)
			_ = extract.AttrString(src, css, attrName)
			_ = extract.AttrOptional(src, css, attrName)
		}

		_ = extract.HTMLUnescape(src)
		_ = extract.AppendHTMLUnescape(nil, src)
	})
}

func FuzzHTMLUnescape(f *testing.F) {
	f.Add([]byte("&lt;div class=&quot;main&quot;&gt;Hello &amp; welcome!&lt;/div&gt;"))
	f.Add([]byte("&#39;&#x2F;&#x3c;&#x3e;"))
	f.Add([]byte("&amp;&amp;&amp;"))
	f.Add([]byte("&nonexistent;"))
	f.Add([]byte("no entities here"))
	f.Add([]byte(""))

	f.Fuzz(func(t *testing.T, src []byte) {
		res := extract.Unescape(src)
		_ = res
	})
}
