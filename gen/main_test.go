package main

import (
	"math"
	"math/big"
	"strings"
	"testing"
)

// Backward recursion x = q_p, x ← q_k + 1/x with q_k = a(1+kn).
func convergent(a, n *big.Rat, p int) *big.Rat {
	q := func(k int) *big.Rat {
		f := new(big.Rat).Mul(n, big.NewRat(int64(k), 1))
		return f.Mul(a, f.Add(f, big.NewRat(1, 1)))
	}
	x := q(p)
	for k := p - 1; k >= 0; k-- {
		x = new(big.Rat).Add(q(k), new(big.Rat).Inv(x))
	}
	return x
}

func TestClosedFormIsTheConvergent(t *testing.T) {
	for _, a := range []*big.Rat{big.NewRat(1, 1), big.NewRat(3, 1), big.NewRat(1, 2)} {
		for _, n := range []*big.Rat{new(big.Rat), big.NewRat(2, 1), big.NewRat(1, 3)} {
			for p := 0; p <= 9; p++ {
				if H(a, n, p).Cmp(convergent(a, n, p)) != 0 {
					t.Fatalf("a=%v n=%v p=%d", a, n, p)
				}
			}
		}
	}
}

func TestBronze(t *testing.T) {
	if math.Abs(B-(3+math.Sqrt(13))/2) > 1e-12 || math.Abs(B-3-1/B) > 1e-12 {
		t.Fatalf("B = %v", B)
	}
}

func css() string {
	var sb strings.Builder
	write(&sb)
	return sb.String()
}

// Neighbouring spacing tokens differ by exactly √B; whole steps by B.
func TestSpacingIsGeometric(t *testing.T) {
	st := steps()[1:]
	for i := 1; i < len(st); i++ {
		if d := st[i].exp - st[i-1].exp; d != 0.5 {
			t.Fatalf("%s→%s: exponent step %v", st[i-1].name, st[i].name, d)
		}
	}
	if !strings.Contains(css(), "--s-2:calc(var(--u) * 1);") {
		t.Fatal("step 2 must be u")
	}
}

func TestClassesAndVariants(t *testing.T) {
	out := css()
	for _, want := range []string{
		`.p-2\.5{padding:var(--s-2-5)}`, `.md\:cols-b3{`, `.lg\:hidden{display:none}`,
		`.stack-3{display:flex;flex-direction:column;gap:var(--s-3)}`, `.flow-2>*+*{margin-top:var(--s-2)}`,
		`.text-3xl{font-size:var(--t-3xl)}`, `--t-3xl:3.302776rem;`, `.cols-b{display:grid;grid-template-columns:minmax(0,3.302776fr) minmax(0,1fr)}`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s", want)
		}
	}
	seen := map[string]bool{}
	for _, c := range classNames() {
		if seen[c] {
			t.Errorf("duplicate class %s", c)
		}
		seen[c] = true
	}
	if strings.Count(out, "{") != strings.Count(out, "}") {
		t.Fatal("unbalanced braces")
	}
}
