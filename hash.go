package main

import "math/rand/v2"

// hashAlphabet leaves out 0, 1, o and l, which are easy to misread.
const hashAlphabet = "23456789abcdefghijkmnpqrstuvwxyz"

// newHash returns a random hash not yet taken. It uses 3 characters
// until 10 draws in a row collide, then 4.
func newHash(taken func(string) bool, r *rand.Rand) string {
	for n := 3; ; n++ {
		for range 10 {
			b := make([]byte, n)
			for i := range b {
				b[i] = hashAlphabet[r.IntN(len(hashAlphabet))]
			}
			if h := string(b); !taken(h) {
				return h
			}
		}
	}
}
