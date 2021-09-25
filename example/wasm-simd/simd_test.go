package main

import (
	"reflect"
	"testing"
	"unsafe"
)

func mulArraySIMD(out, a, b unsafe.Pointer, n int64)
func mulArrayFloat(out, a, b unsafe.Pointer, n int64)

func BenchmarkFloat32VecMul(b *testing.B) {
	const maxVectorLen = 10000000

	data := func(n int) (a, b, c, expected [][4]float32) {
		a = make([][4]float32, 0, n)
		b = make([][4]float32, 0, n)
		c = make([][4]float32, 0, n)
		expected = make([][4]float32, 0, n)
		for i := 0; i < n; i++ {
			f := float32(i)
			a = append(a, [4]float32{f + 0, f + 1, f + 2, f + 3})
			b = append(b, [4]float32{2, 2, 3, 3})
			c = append(c, [4]float32{})
			expected = append(expected, [4]float32{
				(f + 0) * 2,
				(f + 1) * 2,
				(f + 2) * 3,
				(f + 3) * 3,
			})
		}
		return a, b, c, expected
	}

	for name, fn := range map[string]func(a, b, c [][4]float32, n int64){
		"Naive Go": func(a, b, c [][4]float32, n int64) {
			for i := int64(0); i < n; i++ {
				c[i][0], c[i][1], c[i][2], c[i][3] =
					a[i][0]*b[i][0], a[i][1]*b[i][1], a[i][2]*b[i][2], a[i][3]*b[i][3]
			}
		},
		"Asm SIMD": func(a, b, c [][4]float32, n int64) {
			mulArraySIMD(
				unsafe.Pointer(&c[0][0]),
				unsafe.Pointer(&a[0][0]),
				unsafe.Pointer(&b[0][0]),
				n,
			)
		},
		"Asm Float": func(a, b, c [][4]float32, n int64) {
			mulArrayFloat(
				unsafe.Pointer(&c[0][0]),
				unsafe.Pointer(&a[0][0]),
				unsafe.Pointer(&b[0][0]),
				n*4,
			)
		},
	} {
		fn := fn
		b.Run(name, func(b *testing.B) {
			b.SetBytes(4 * 4)

			dataSize, repeat := b.N, 1
			if dataSize > maxVectorLen {
				repeat = dataSize / maxVectorLen
				dataSize /= repeat
			}
			inA, inB, out, expected := data(dataSize)

			b.ResetTimer()
			for i := 0; i < repeat; i++ {
				fn(inA, inB, out, int64(dataSize))
			}
			b.StopTimer()

			if !reflect.DeepEqual(expected, out) {
				b.Fatalf("Unexpected result\n%v\n%v", expected[0], out[0])
			}
		})
	}
}
