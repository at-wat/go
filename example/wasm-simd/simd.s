// func mulArraySIMD(out, a, b unsafe.Pointer, n int)
TEXT ·mulArraySIMD(SB),$0-32
	Get SP
	MOVW out+0(FP), R0
	MOVW a+8(FP), R1
	MOVW b+16(FP), R2
	MOVD n+24(FP), R3

loop:
	Loop
		Get R3
		I64Eqz
		If
			RET
		End

		Get R0
		I32WrapI64
		V128Load (R2)
		V128Load (R1)
		F32x4Mul
		V128Store

		Get R0
		I64Const $16
		I64Add
		Set R0

		Get R1
		I64Const $16
		I64Add
		Set R1

		Get R2
		I64Const $16
		I64Add
		Set R2

		Get R3
		I64Const $1
		I64Sub
		Set R3

		Br loop
	End

// func mulArrayFloat(out, a, b unsafe.Pointer, n int)
TEXT ·mulArrayFloat(SB),$0-32
	Get SP
	MOVW out+0(FP), R0
	MOVW a+8(FP), R1
	MOVW b+16(FP), R2
	MOVD n+24(FP), R3

loop:
	Loop
		Get R3
		I64Eqz
		If
			RET
		End

		Get R0
		I32WrapI64
		F32Load (R2)
		F32Load (R1)
		F32Mul
		F32Store $0

		Get R0
		I64Const $4
		I64Add
		Set R0

		Get R1
		I64Const $4
		I64Add
		Set R1

		Get R2
		I64Const $4
		I64Add
		Set R2

		Get R3
		I64Const $1
		I64Sub
		Set R3

		Br loop
	End
