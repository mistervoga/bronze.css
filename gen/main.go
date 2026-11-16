// Command gen writes bronze.css: a utility-first CSS framework in which every
// length, font size, ratio and breakpoint is a power of the bronze ratio
// B = h(3,0) = [3; 3, 3, …] = (3+√13)/2. B is derived from the closed form of
// the Hurwitz continued fraction h(a,n) = [a, a(1+n), a(1+2n), …] (see README).
//
//	go run ./gen -o bronze.css
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"sort"
	"strings"
)

// H is the p-th convergent of h(a,n), closed form (1) of the manuscript:
//
//	Σ_{k≤⌊(p+1)/2⌋} a^(p−2k+1) C(p−k+1,k) Π_{i=k}^{p−k}(1+in)
//	─────────────────────────────────────────────────────────
//	Σ_{k≤⌊p/2⌋}     a^(p−2k)   C(p−k,k)   Π_{i=k+1}^{p−k}(1+in)
func H(a, n *big.Rat, p int) *big.Rat {
	sum := func(top, shift, lo int) *big.Rat {
		s := new(big.Rat)
		for k := 0; k <= top; k++ {
			t := new(big.Rat).SetInt(new(big.Int).Binomial(int64(p-k+shift), int64(k)))
			for e := 0; e < p-2*k+shift; e++ {
				t.Mul(t, a)
			}
			for i := k + lo; i <= p-k; i++ {
				f := new(big.Rat).Mul(n, big.NewRat(int64(i), 1))
				t.Mul(t, f.Add(f, big.NewRat(1, 1)))
			}
			s.Add(s, t)
		}
		return s
	}
	return new(big.Rat).Quo(sum((p+1)/2, 1, 0), sum(p/2, 0, 1))
}

// Bronze returns lim H_p(3,0), stopping when convergents differ by < 1e-13.
func Bronze() float64 {
	three, zero := big.NewRat(3, 1), new(big.Rat)
	prev, _ := H(three, zero, 0).Float64()
	for p := 1; ; p++ {
		cur, _ := H(three, zero, p).Float64()
		if math.Abs(cur-prev) < 1e-13 {
			return cur
		}
		prev = cur
	}
}

var B = Bronze()

// pow returns B^e as a short decimal.
func pow(e float64) string { return num(math.Pow(B, e)) }
func num(v float64) string {
	s := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.6f", v), "0"), ".")
	if s == "-0" {
		return "0"
	}
	return s
}

// Spacing steps: step n (and n.5) is u·B^(n−2). Step 2 = u.
type step struct {
	name string  // class suffix, e.g. "2.5"
	tok  string  // custom property, e.g. --s-2-5
	exp  float64 // exponent of B; NaN for zero
}

func steps() []step {
	out := []step{{"0", "--s-0", math.NaN()}}
	for n := 1.0; n <= 5; n += 0.5 {
		name := num(n)
		out = append(out, step{name, "--s-" + strings.ReplaceAll(name, ".", "-"), n - 2})
	}
	return out
}

// esc escapes a class name for a selector ("p-2.5" -> "p-2\.5", "md:x" -> "md\:x").
func esc(c string) string {
	return strings.NewReplacer(".", `\.`, ":", `\:`, "/", `\/`).Replace(c)
}

type rule struct{ class, decl string }

// utilities are emitted once plainly and once per breakpoint with a prefix.
func utilities() []rule {
	var r []rule
	add := func(c, d string) { r = append(r, rule{c, d}) }

	// spacing: margin, padding, gap
	sides := []struct{ k, props string }{
		{"", "%s"}, {"t", "%s-top"}, {"r", "%s-right"}, {"b", "%s-bottom"}, {"l", "%s-left"},
		{"x", "%s-inline"}, {"y", "%s-block"},
	}
	for _, s := range steps() {
		v := "var(" + s.tok + ")"
		for _, side := range sides {
			add("m"+side.k+"-"+s.name, fmt.Sprintf(side.props, "margin")+":"+v)
			add("p"+side.k+"-"+s.name, fmt.Sprintf(side.props, "padding")+":"+v)
		}
		add("gap-"+s.name, "gap:"+v)
		add("gap-x-"+s.name, "column-gap:"+v)
		add("gap-y-"+s.name, "row-gap:"+v)
		add("stack-"+s.name, "display:flex;flex-direction:column;gap:"+v)
		add("cluster-"+s.name, "display:flex;flex-wrap:wrap;align-items:center;gap:"+v)
	}
	for _, side := range []struct{ k, p string }{{"", "margin"}, {"x", "margin-inline"}, {"y", "margin-block"}, {"t", "margin-top"}, {"b", "margin-bottom"}, {"l", "margin-left"}, {"r", "margin-right"}} {
		add("m"+side.k+"-auto", side.p+":auto")
	}

	// display, flex, grid
	for _, d := range []string{"block", "inline", "inline-block", "flex", "inline-flex", "grid", "contents"} {
		add(d, "display:"+d)
	}
	add("hidden", "display:none")
	add("flex-row", "flex-direction:row")
	add("flex-col", "flex-direction:column")
	add("flex-wrap", "flex-wrap:wrap")
	add("flex-1", "flex:1 1 0%;min-width:0")
	add("flex-none", "flex:none")
	add("grow", "flex-grow:1")
	for _, a := range []string{"start", "center", "end", "baseline", "stretch"} {
		v := a
		if a == "start" || a == "end" {
			v = "flex-" + a
		}
		add("items-"+a, "align-items:"+v)
		add("self-"+a, "align-self:"+v)
	}
	for _, j := range []string{"start", "center", "end", "between", "around"} {
		v := j
		switch j {
		case "start", "end":
			v = "flex-" + j
		case "between", "around":
			v = "space-" + j
		}
		add("justify-"+j, "justify-content:"+v)
	}
	for n := 1; n <= 6; n++ {
		add(fmt.Sprintf("cols-%d", n), fmt.Sprintf("display:grid;grid-template-columns:repeat(%d,minmax(0,1fr))", n))
		add(fmt.Sprintf("span-%d", n), fmt.Sprintf("grid-column:span %d/span %d", n, n))
	}
	add("span-full", "grid-column:1/-1")
	// bronze grids (each cols-* is itself a grid): B = 3 + 1/B, B² = 3B + 1
	b1 := num(B - 1)
	add("cols-b", "display:grid;grid-template-columns:minmax(0,"+num(B)+"fr) minmax(0,1fr)")        // B : 1
	add("cols-b-rev", "display:grid;grid-template-columns:minmax(0,1fr) minmax(0,"+num(B)+"fr)")    // 1 : B
	add("cols-b1", "display:grid;grid-template-columns:minmax(0,"+b1+"fr) minmax(0,1fr)")           // (B−1) : 1
	add("cols-b1-rev", "display:grid;grid-template-columns:minmax(0,1fr) minmax(0,"+b1+"fr)")       // 1 : (B−1)
	add("cols-b3", "display:grid;grid-template-columns:minmax(0,"+b1+"fr) repeat(2,minmax(0,1fr))") // (B−1) : 1 : 1
	add("cols-3b", "display:grid;grid-template-columns:repeat(3,minmax(0,1fr)) minmax(0,"+pow(-1)+"fr)")
	add("auto-fill", "display:grid;grid-template-columns:repeat(auto-fill,minmax(min(100%,var(--s-4-5)),1fr))")

	// sizes
	add("w-full", "width:100%")
	add("w-auto", "width:auto")
	add("w-major", "width:calc(100% - 100% / var(--b))") // (B−1)/B
	add("w-minor", "width:calc(100% / var(--b))")        // 1/B
	add("w-minor-2", "width:calc(100% / var(--b2))")     // 1/B²
	add("h-full", "height:100%")
	add("h-screen", "height:100dvh")
	add("min-h-screen", "min-height:100dvh")
	add("min-w-0", "min-width:0")
	add("min-h-0", "min-height:0")
	add("max-w-measure", "max-width:var(--measure)")
	add("max-w-container", "max-width:var(--container)")
	add("max-w-none", "max-width:none")
	add("aspect-b", "aspect-ratio:"+num(B))
	add("aspect-b1", "aspect-ratio:"+b1)
	add("aspect-sqrt-b", "aspect-ratio:"+pow(0.5))
	add("aspect-square", "aspect-ratio:1")

	// type scale: font-size 1rem·B^(k/4)
	for _, t := range typeScale {
		add("text-"+t.name, "font-size:var(--t-"+t.name+")")
	}
	add("leading-tight", "line-height:"+pow(0.25))
	add("leading-normal", "line-height:"+pow(1.0/3))
	add("leading-loose", "line-height:"+pow(0.5))
	for _, a := range []string{"left", "center", "right"} {
		add("text-"+a, "text-align:"+a)
	}
	add("font-normal", "font-weight:400")
	add("font-bold", "font-weight:700")
	add("font-mono", "font-family:var(--mono)")
	add("font-sans", "font-family:var(--sans)")
	add("uppercase", "text-transform:uppercase;letter-spacing:calc(1em / var(--b3))")
	add("truncate", "overflow:hidden;text-overflow:ellipsis;white-space:nowrap")
	add("nowrap", "white-space:nowrap")
	add("tabular", "font-variant-numeric:tabular-nums")

	// position, overflow
	for _, p := range []string{"static", "relative", "absolute", "sticky"} {
		add(p, "position:"+p)
	}
	add("top-0", "top:0")
	for _, o := range []string{"auto", "hidden"} {
		add("overflow-"+o, "overflow:"+o)
	}
	return r
}

// colour, border, radius and helper classes are not responsive.
func plain() []rule {
	var r []rule
	add := func(c, d string) { r = append(r, rule{c, d}) }
	for _, c := range []string{"ink", "ink-2", "muted", "accent", "page", "line"} {
		add("text-"+c, "color:var(--"+c+")")
		add("bg-"+c, "background-color:var(--"+c+")")
	}
	add("border", "border:1px solid var(--line)")
	for _, s := range []struct{ k, p string }{{"t", "top"}, {"r", "right"}, {"b", "bottom"}, {"l", "left"}} {
		add("border-"+s.k, "border-"+s.p+":1px solid var(--line)")
	}
	add("border-0", "border:0")
	add("border-accent", "border-color:var(--accent)")
	add("rounded-0", "border-radius:0")
	add("rounded-1", "border-radius:var(--s-1)")
	add("rounded-2", "border-radius:var(--s-2)")
	add("rounded-full", "border-radius:9999px")
	add("underline", "text-decoration:underline;text-underline-offset:calc(1em / var(--b2) * 2)")
	add("no-underline", "text-decoration:none")
	add("sr-only", "position:absolute;width:1px;height:1px;padding:0;margin:-1px;overflow:hidden;clip:rect(0,0,0,0);white-space:nowrap;border:0")
	add("container", "width:100%;max-width:var(--container);margin-inline:auto;padding-inline:var(--s-2-5)")
	add("measure", "max-width:var(--measure)")
	return r
}

var typeScale = []struct {
	name string
	exp  float64 // exponent of B (in quarters)
}{
	{"xs", -0.5}, {"sm", -0.25}, {"base", 0}, {"lg", 0.25}, {"xl", 0.5}, {"2xl", 0.75}, {"3xl", 1},
}

// Breakpoints: u·B^3.5 and u·B^4 (rem, since media queries cannot read custom properties).
const uRem = 0.5625 // u = 9px at 16px root

var breakpoints = []struct {
	name string
	exp  float64
}{{"md", 3.5}, {"lg", 4}}

func write(w io.Writer) {
	p := func(f string, a ...any) { fmt.Fprintf(w, f, a...) }
	p("/*! bronze.css v%s · MIT License · utility CSS in the bronze ratio B = (3+√13)/2 = %s\n", Version, num(B))
	p("   B = h(3,0) = [3; 3, 3, …], derived from the Hurwitz continued fraction (README.md).\n")
	p("   Every length is u·B^k, every font size 1rem·B^(k/4). Generated by gen/main.go (go run ./gen -o bronze.css) — do not edit. */\n\n")

	// tokens
	p(":root{\n  --b:%s;--b2:%s;--b3:%s;--b-inv:%s;\n  --u:%srem;\n", num(B), num(B*B), num(B*B*B), num(1/B), num(uRem))
	for _, s := range steps() {
		if math.IsNaN(s.exp) {
			p("  %s:0;\n", s.tok)
			continue
		}
		p("  %s:calc(var(--u) * %s);\n", s.tok, pow(s.exp))
	}
	p("  --measure:calc(var(--u) * %s);--container:calc(var(--u) * %s);\n", pow(3.5), pow(4))
	for _, t := range typeScale {
		p("  --t-%s:%srem;\n", t.name, pow(t.exp))
	}
	p("  --sans:Verdana,Geneva,system-ui,sans-serif;--mono:ui-monospace,\"Cascadia Mono\",\"JetBrains Mono\",monospace;\n")
	p("  --page:#ebe3d6;--ink:#111;--ink-2:#444;--muted:#767676;--line:#d9cfc0;--accent:#3a050c;\n  color-scheme:light;\n}\n\n")

	// base
	p("*,*::before,*::after{box-sizing:border-box}\n")
	p("html{-webkit-text-size-adjust:100%%;text-size-adjust:100%%}\n")
	p("body{margin:0;background:var(--page);color:var(--ink);font:var(--t-base)/%s var(--sans)}\n", pow(1.0/3))
	p("h1,h2,h3,h4,p,figure,blockquote,dl,dd,ul,ol{margin:0}\n")
	p("h1{font-size:var(--t-3xl);line-height:%s}h2{font-size:var(--t-xl);line-height:%s}h3{font-size:var(--t-lg)}h4{font-size:var(--t-base)}\n", pow(0.25), pow(0.25))
	p("img,svg,video,canvas{display:block;max-width:100%%}\n")
	p("input,select,textarea,button{font:inherit;color:inherit}\n")
	p("a{color:inherit}\n:focus-visible{outline:1px solid var(--ink);outline-offset:var(--s-1)}\n")
	p("@media (prefers-reduced-motion:reduce){*{transition:none!important;animation:none!important}}\n\n")

	// owl rhythm: .flow-N > * + *
	for _, s := range steps() {
		p(".flow-%s>*+*{margin-top:var(%s)}\n", esc(s.name), s.tok)
	}
	p("\n")
	for _, r := range plain() {
		p(".%s{%s}\n", esc(r.class), r.decl)
	}
	p("\n")
	u := utilities()
	for _, r := range u {
		p(".%s{%s}\n", esc(r.class), r.decl)
	}
	for _, bp := range breakpoints {
		p("\n@media (min-width:%srem){\n", num(uRem*math.Pow(B, bp.exp)))
		for _, r := range u {
			p(".%s{%s}\n", esc(bp.name+":"+r.class), r.decl)
		}
		p("}\n")
	}
}

// classNames lists every class the stylesheet defines (used by the test and README).
func classNames() []string {
	var out []string
	for _, r := range append(plain(), utilities()...) {
		out = append(out, r.class)
	}
	sort.Strings(out)
	return out
}

// Version is printed in the stylesheet header.
const Version = "1.0.0"

// main writes to -o (default stdout). -o avoids shell redirection, which in
// Windows PowerShell 5.1 would write UTF-16 instead of UTF-8.
func main() {
	out := flag.String("o", "", "output file (default: stdout)")
	flag.Parse()
	w := os.Stdout
	if *out != "" {
		f, err := os.Create(*out)
		if err != nil {
			fmt.Fprintln(os.Stderr, "bronze.css:", err)
			os.Exit(1)
		}
		defer f.Close()
		w = f
	}
	bw := bufio.NewWriter(w)
	write(bw)
	if err := bw.Flush(); err != nil {
		fmt.Fprintln(os.Stderr, "bronze.css:", err)
		os.Exit(1)
	}
	if *out != "" {
		fmt.Fprintf(os.Stderr, "bronze.css v%s written: %s\n", Version, *out)
	}
}
