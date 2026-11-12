package web

import (
	"net/url"
	"testing"
)

func TestTextDropsMarkupAndScripts(t *testing.T) {
	page := `<html><head><style>p{}</style><script>var x=1</script></head>
<body><h1>Blur</h1><p>Сайт закривається&nbsp;12.11</p><div>Blend <b>теж</b></div></body></html>`

	if got := Text(page); got != "Blur\n\nСайт закривається 12.11\n\nBlend теж" {
		t.Errorf("got %q", got)
	}
}

func TestLinksKeepTheirAddressesInTheText(t *testing.T) {
	base, _ := url.Parse("https://blog.example.org/")
	page := `<p><a class="x" href="/blog/first-post">First <b>post</b></a> and <a href="https://t.me/s/chan#top">channel</a>,
<a href="mailto:a@b.c">mail</a>, <a href="#menu">menu</a></p>`

	want := "First post (https://blog.example.org/blog/first-post) and channel (https://t.me/s/chan),\n\nmail, menu"
	if got := Text(linked(page, base)); got != want {
		t.Errorf("got %q", got)
	}
}
