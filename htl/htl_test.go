package htl

import (
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct{ in, want string }{
		{"",
			""},
		{"(a :href http://foo.bar/{{user}} \"안녕\")",
			"<a href=\"http://foo.bar/{{user}}\">안녕</a>"},
		{"(a :href foo)",
			"<a href=\"foo\"></a>"},
		{"(br)",
			"<br/>"},
		{"(a :href foo )",
			"<a href=\"foo\"></a>"},
		{"(a b)",
			"<a>b</a>"},
		{"(a b(c d))",
			"<a>b<c>d</c></a>"},
		{"(a (b (c)))",
			"<a><b><c></c></b></a>"},
		{"(a(b(c)))",
			"<a><b><c></c></b></a>"},
		{"(a(b(c))", // missing a closing bracket.
			""},
		{"(a(b(c))))", // extra closing bracket.
			""},
		{"(a \"x\\ty\\nz\")",
			"<a>x\ty\nz</a>"},
		{"(a :x 1 (b :z 2 :y 3 (c \"foo bar\" \"baz\")))",
			"<a x=\"1\"><b y=\"3\" z=\"2\"><c>foo barbaz</c></b></a>"},
		{"(a :x \"\\\\<>'\\\"\" \"content\")",
			"<a x=\"\\&lt;&gt;&apos;&quot;\">content</a>"},
		{"(a ;comments\n :x ;comments\n\"\\\\<>'\\\"\" ;comments \n\"content\")",
			"<a x=\"\\&lt;&gt;&apos;&quot;\">content</a>"},
		{"(a \"b\" ; \"c\"\n ;; \"d\"\n)",
			"<a>b</a>"},
	}
	for _, c := range cases {
		parsedTree, _ := Parse(c.in)
		if got := parsedTree.String(); got != c.want {
			t.Errorf("Parse(x).String():\ninput: %q\n  got: %q\n want: %q",
				c.in, got, c.want)
		}
	}
}
