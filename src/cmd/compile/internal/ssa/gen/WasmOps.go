// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore
// +build ignore

package main

import "strings"

var regNamesWasm = []string{
	"R0",
	"R1",
	"R2",
	"R3",
	"R4",
	"R5",
	"R6",
	"R7",
	"R8",
	"R9",
	"R10",
	"R11",
	"R12",
	"R13",
	"R14",
	"R15",

	"F0",
	"F1",
	"F2",
	"F3",
	"F4",
	"F5",
	"F6",
	"F7",
	"F8",
	"F9",
	"F10",
	"F11",
	"F12",
	"F13",
	"F14",
	"F15",

	"F16",
	"F17",
	"F18",
	"F19",
	"F20",
	"F21",
	"F22",
	"F23",
	"F24",
	"F25",
	"F26",
	"F27",
	"F28",
	"F29",
	"F30",
	"F31",

	"V0",
	"V1",
	"V2",
	"V3",
	"V4",
	"V5",
	"V6",
	"V7",

	"SP",
	"g",

	// pseudo-registers
	"SB",
}

func init() {
	// Make map from reg names to reg integers.
	if len(regNamesWasm) > 64 {
		panic("too many registers")
	}
	num := map[string]int{}
	for i, name := range regNamesWasm {
		num[name] = i
	}
	buildReg := func(s string) regMask {
		m := regMask(0)
		for _, r := range strings.Split(s, " ") {
			if n, ok := num[r]; ok {
				m |= regMask(1) << uint(n)
				continue
			}
			panic("register " + r + " not found")
		}
		return m
	}

	var (
		gp     = buildReg("R0 R1 R2 R3 R4 R5 R6 R7 R8 R9 R10 R11 R12 R13 R14 R15")
		fp32   = buildReg("F0 F1 F2 F3 F4 F5 F6 F7 F8 F9 F10 F11 F12 F13 F14 F15")
		fp64   = buildReg("F16 F17 F18 F19 F20 F21 F22 F23 F24 F25 F26 F27 F28 F29 F30 F31")
		vp128  = buildReg("V0 V1 V2 V3 V4 V5 V6 V7")
		gpsp   = gp | buildReg("SP")
		gpspsb = gpsp | buildReg("SB")
		// The "registers", which are actually local variables, can get clobbered
		// if we're switching goroutines, because it unwinds the WebAssembly stack.
		callerSave = gp | fp32 | fp64 | buildReg("g")
	)

	// Common regInfo
	var (
		gp01       = regInfo{inputs: nil, outputs: []regMask{gp}}
		gp11       = regInfo{inputs: []regMask{gpsp}, outputs: []regMask{gp}}
		gp21       = regInfo{inputs: []regMask{gpsp, gpsp}, outputs: []regMask{gp}}
		gp31       = regInfo{inputs: []regMask{gpsp, gpsp, gpsp}, outputs: []regMask{gp}}
		fp32_01    = regInfo{inputs: nil, outputs: []regMask{fp32}}
		fp32_11    = regInfo{inputs: []regMask{fp32}, outputs: []regMask{fp32}}
		fp32_21    = regInfo{inputs: []regMask{fp32, fp32}, outputs: []regMask{fp32}}
		fp32_21gp  = regInfo{inputs: []regMask{fp32, fp32}, outputs: []regMask{gp}}
		fp64_01    = regInfo{inputs: nil, outputs: []regMask{fp64}}
		fp64_11    = regInfo{inputs: []regMask{fp64}, outputs: []regMask{fp64}}
		fp64_21    = regInfo{inputs: []regMask{fp64, fp64}, outputs: []regMask{fp64}}
		fp64_21gp  = regInfo{inputs: []regMask{fp64, fp64}, outputs: []regMask{gp}}
		vp128_01   = regInfo{inputs: nil, outputs: []regMask{vp128}}
		vp128_11   = regInfo{inputs: []regMask{vp128}, outputs: []regMask{vp128}}
		vp128_11gp = regInfo{inputs: []regMask{vp128}, outputs: []regMask{gp}}
		vp128_21   = regInfo{inputs: []regMask{vp128, vp128}, outputs: []regMask{vp128}}
		vp128_31   = regInfo{inputs: []regMask{vp128, vp128, vp128}, outputs: []regMask{vp128}}
		gpload     = regInfo{inputs: []regMask{gpspsb, 0}, outputs: []regMask{gp}}
		gpstore    = regInfo{inputs: []regMask{gpspsb, gpsp, 0}}
		fp32load   = regInfo{inputs: []regMask{gpspsb, 0}, outputs: []regMask{fp32}}
		fp32store  = regInfo{inputs: []regMask{gpspsb, fp32, 0}}
		fp64load   = regInfo{inputs: []regMask{gpspsb, 0}, outputs: []regMask{fp64}}
		fp64store  = regInfo{inputs: []regMask{gpspsb, fp64, 0}}
		vp128load  = regInfo{inputs: []regMask{gpspsb, 0}, outputs: []regMask{vp128}}
		vp128store = regInfo{inputs: []regMask{gpspsb, vp128, 0}}
	)

	var WasmOps = []opData{
		{name: "LoweredStaticCall", argLength: 1, reg: regInfo{clobbers: callerSave}, aux: "CallOff", call: true},                                // call static function aux.(*obj.LSym). arg0=mem, auxint=argsize, returns mem
		{name: "LoweredTailCall", argLength: 1, reg: regInfo{clobbers: callerSave}, aux: "CallOff", call: true},                                  // tail call static function aux.(*obj.LSym). arg0=mem, auxint=argsize, returns mem
		{name: "LoweredClosureCall", argLength: 3, reg: regInfo{inputs: []regMask{gp, gp, 0}, clobbers: callerSave}, aux: "CallOff", call: true}, // call function via closure. arg0=codeptr, arg1=closure, arg2=mem, auxint=argsize, returns mem
		{name: "LoweredInterCall", argLength: 2, reg: regInfo{inputs: []regMask{gp}, clobbers: callerSave}, aux: "CallOff", call: true},          // call fn by pointer. arg0=codeptr, arg1=mem, auxint=argsize, returns mem

		{name: "LoweredAddr", argLength: 1, reg: gp11, aux: "SymOff", rematerializeable: true, symEffect: "Addr"}, // returns base+aux+auxint, arg0=base
		{name: "LoweredMove", argLength: 3, reg: regInfo{inputs: []regMask{gp, gp}}, aux: "Int64"},                // large move. arg0=dst, arg1=src, arg2=mem, auxint=len/8, returns mem
		{name: "LoweredZero", argLength: 2, reg: regInfo{inputs: []regMask{gp}}, aux: "Int64"},                    // large zeroing. arg0=start, arg1=mem, auxint=len/8, returns mem

		{name: "LoweredGetClosurePtr", reg: gp01},                                                                          // returns wasm.REG_CTXT, the closure pointer
		{name: "LoweredGetCallerPC", reg: gp01, rematerializeable: true},                                                   // returns the PC of the caller of the current function
		{name: "LoweredGetCallerSP", reg: gp01, rematerializeable: true},                                                   // returns the SP of the caller of the current function
		{name: "LoweredNilCheck", argLength: 2, reg: regInfo{inputs: []regMask{gp}}, nilCheck: true, faultOnNilArg0: true}, // panic if arg0 is nil. arg1=mem
		{name: "LoweredWB", argLength: 3, reg: regInfo{inputs: []regMask{gp, gp}}, aux: "Sym", symEffect: "None"},          // invokes runtime.gcWriteBarrier. arg0=destptr, arg1=srcptr, arg2=mem, aux=runtime.gcWriteBarrier

		// LoweredConvert converts between pointers and integers.
		// We have a special op for this so as to not confuse GCCallOff
		// (particularly stack maps). It takes a memory arg so it
		// gets correctly ordered with respect to GC safepoints.
		// arg0=ptr/int arg1=mem, output=int/ptr
		//
		// TODO(neelance): LoweredConvert should not be necessary any more, since OpConvert does not need to be lowered any more (CL 108496).
		{name: "LoweredConvert", argLength: 2, reg: regInfo{inputs: []regMask{gp}, outputs: []regMask{gp}}},

		// The following are native WebAssembly instructions, see https://webassembly.github.io/spec/core/syntax/instructions.html

		{name: "Select", asm: "Select", argLength: 3, reg: gp31}, // returns arg0 if arg2 != 0, otherwise returns arg1

		{name: "I64Load8U", asm: "I64Load8U", argLength: 2, reg: gpload, aux: "Int64", typ: "UInt8"},    // read unsigned 8-bit integer from address arg0+aux, arg1=mem
		{name: "I64Load8S", asm: "I64Load8S", argLength: 2, reg: gpload, aux: "Int64", typ: "Int8"},     // read signed 8-bit integer from address arg0+aux, arg1=mem
		{name: "I64Load16U", asm: "I64Load16U", argLength: 2, reg: gpload, aux: "Int64", typ: "UInt16"}, // read unsigned 16-bit integer from address arg0+aux, arg1=mem
		{name: "I64Load16S", asm: "I64Load16S", argLength: 2, reg: gpload, aux: "Int64", typ: "Int16"},  // read signed 16-bit integer from address arg0+aux, arg1=mem
		{name: "I64Load32U", asm: "I64Load32U", argLength: 2, reg: gpload, aux: "Int64", typ: "UInt32"}, // read unsigned 32-bit integer from address arg0+aux, arg1=mem
		{name: "I64Load32S", asm: "I64Load32S", argLength: 2, reg: gpload, aux: "Int64", typ: "Int32"},  // read signed 32-bit integer from address arg0+aux, arg1=mem
		{name: "I64Load", asm: "I64Load", argLength: 2, reg: gpload, aux: "Int64", typ: "UInt64"},       // read 64-bit integer from address arg0+aux, arg1=mem
		{name: "I64Store8", asm: "I64Store8", argLength: 3, reg: gpstore, aux: "Int64", typ: "Mem"},     // store 8-bit integer arg1 at address arg0+aux, arg2=mem, returns mem
		{name: "I64Store16", asm: "I64Store16", argLength: 3, reg: gpstore, aux: "Int64", typ: "Mem"},   // store 16-bit integer arg1 at address arg0+aux, arg2=mem, returns mem
		{name: "I64Store32", asm: "I64Store32", argLength: 3, reg: gpstore, aux: "Int64", typ: "Mem"},   // store 32-bit integer arg1 at address arg0+aux, arg2=mem, returns mem
		{name: "I64Store", asm: "I64Store", argLength: 3, reg: gpstore, aux: "Int64", typ: "Mem"},       // store 64-bit integer arg1 at address arg0+aux, arg2=mem, returns mem

		{name: "F32Load", asm: "F32Load", argLength: 2, reg: fp32load, aux: "Int64", typ: "Float32"}, // read 32-bit float from address arg0+aux, arg1=mem
		{name: "F64Load", asm: "F64Load", argLength: 2, reg: fp64load, aux: "Int64", typ: "Float64"}, // read 64-bit float from address arg0+aux, arg1=mem
		{name: "F32Store", asm: "F32Store", argLength: 3, reg: fp32store, aux: "Int64", typ: "Mem"},  // store 32-bit float arg1 at address arg0+aux, arg2=mem, returns mem
		{name: "F64Store", asm: "F64Store", argLength: 3, reg: fp64store, aux: "Int64", typ: "Mem"},  // store 64-bit float arg1 at address arg0+aux, arg2=mem, returns mem

		{name: "I64Const", reg: gp01, aux: "Int64", rematerializeable: true, typ: "Int64"},        // returns the constant integer aux
		{name: "F32Const", reg: fp32_01, aux: "Float32", rematerializeable: true, typ: "Float32"}, // returns the constant float aux
		{name: "F64Const", reg: fp64_01, aux: "Float64", rematerializeable: true, typ: "Float64"}, // returns the constant float aux

		{name: "I64Eqz", asm: "I64Eqz", argLength: 1, reg: gp11, typ: "Bool"}, // arg0 == 0
		{name: "I64Eq", asm: "I64Eq", argLength: 2, reg: gp21, typ: "Bool"},   // arg0 == arg1
		{name: "I64Ne", asm: "I64Ne", argLength: 2, reg: gp21, typ: "Bool"},   // arg0 != arg1
		{name: "I64LtS", asm: "I64LtS", argLength: 2, reg: gp21, typ: "Bool"}, // arg0 < arg1 (signed)
		{name: "I64LtU", asm: "I64LtU", argLength: 2, reg: gp21, typ: "Bool"}, // arg0 < arg1 (unsigned)
		{name: "I64GtS", asm: "I64GtS", argLength: 2, reg: gp21, typ: "Bool"}, // arg0 > arg1 (signed)
		{name: "I64GtU", asm: "I64GtU", argLength: 2, reg: gp21, typ: "Bool"}, // arg0 > arg1 (unsigned)
		{name: "I64LeS", asm: "I64LeS", argLength: 2, reg: gp21, typ: "Bool"}, // arg0 <= arg1 (signed)
		{name: "I64LeU", asm: "I64LeU", argLength: 2, reg: gp21, typ: "Bool"}, // arg0 <= arg1 (unsigned)
		{name: "I64GeS", asm: "I64GeS", argLength: 2, reg: gp21, typ: "Bool"}, // arg0 >= arg1 (signed)
		{name: "I64GeU", asm: "I64GeU", argLength: 2, reg: gp21, typ: "Bool"}, // arg0 >= arg1 (unsigned)

		{name: "F32Eq", asm: "F32Eq", argLength: 2, reg: fp32_21gp, typ: "Bool"}, // arg0 == arg1
		{name: "F32Ne", asm: "F32Ne", argLength: 2, reg: fp32_21gp, typ: "Bool"}, // arg0 != arg1
		{name: "F32Lt", asm: "F32Lt", argLength: 2, reg: fp32_21gp, typ: "Bool"}, // arg0 < arg1
		{name: "F32Gt", asm: "F32Gt", argLength: 2, reg: fp32_21gp, typ: "Bool"}, // arg0 > arg1
		{name: "F32Le", asm: "F32Le", argLength: 2, reg: fp32_21gp, typ: "Bool"}, // arg0 <= arg1
		{name: "F32Ge", asm: "F32Ge", argLength: 2, reg: fp32_21gp, typ: "Bool"}, // arg0 >= arg1

		{name: "F64Eq", asm: "F64Eq", argLength: 2, reg: fp64_21gp, typ: "Bool"}, // arg0 == arg1
		{name: "F64Ne", asm: "F64Ne", argLength: 2, reg: fp64_21gp, typ: "Bool"}, // arg0 != arg1
		{name: "F64Lt", asm: "F64Lt", argLength: 2, reg: fp64_21gp, typ: "Bool"}, // arg0 < arg1
		{name: "F64Gt", asm: "F64Gt", argLength: 2, reg: fp64_21gp, typ: "Bool"}, // arg0 > arg1
		{name: "F64Le", asm: "F64Le", argLength: 2, reg: fp64_21gp, typ: "Bool"}, // arg0 <= arg1
		{name: "F64Ge", asm: "F64Ge", argLength: 2, reg: fp64_21gp, typ: "Bool"}, // arg0 >= arg1

		{name: "I64Add", asm: "I64Add", argLength: 2, reg: gp21, typ: "Int64"},                    // arg0 + arg1
		{name: "I64AddConst", asm: "I64Add", argLength: 1, reg: gp11, aux: "Int64", typ: "Int64"}, // arg0 + aux
		{name: "I64Sub", asm: "I64Sub", argLength: 2, reg: gp21, typ: "Int64"},                    // arg0 - arg1
		{name: "I64Mul", asm: "I64Mul", argLength: 2, reg: gp21, typ: "Int64"},                    // arg0 * arg1
		{name: "I64DivS", asm: "I64DivS", argLength: 2, reg: gp21, typ: "Int64"},                  // arg0 / arg1 (signed)
		{name: "I64DivU", asm: "I64DivU", argLength: 2, reg: gp21, typ: "Int64"},                  // arg0 / arg1 (unsigned)
		{name: "I64RemS", asm: "I64RemS", argLength: 2, reg: gp21, typ: "Int64"},                  // arg0 % arg1 (signed)
		{name: "I64RemU", asm: "I64RemU", argLength: 2, reg: gp21, typ: "Int64"},                  // arg0 % arg1 (unsigned)
		{name: "I64And", asm: "I64And", argLength: 2, reg: gp21, typ: "Int64"},                    // arg0 & arg1
		{name: "I64Or", asm: "I64Or", argLength: 2, reg: gp21, typ: "Int64"},                      // arg0 | arg1
		{name: "I64Xor", asm: "I64Xor", argLength: 2, reg: gp21, typ: "Int64"},                    // arg0 ^ arg1
		{name: "I64Shl", asm: "I64Shl", argLength: 2, reg: gp21, typ: "Int64"},                    // arg0 << (arg1 % 64)
		{name: "I64ShrS", asm: "I64ShrS", argLength: 2, reg: gp21, typ: "Int64"},                  // arg0 >> (arg1 % 64) (signed)
		{name: "I64ShrU", asm: "I64ShrU", argLength: 2, reg: gp21, typ: "Int64"},                  // arg0 >> (arg1 % 64) (unsigned)

		{name: "F32Neg", asm: "F32Neg", argLength: 1, reg: fp32_11, typ: "Float32"}, // -arg0
		{name: "F32Add", asm: "F32Add", argLength: 2, reg: fp32_21, typ: "Float32"}, // arg0 + arg1
		{name: "F32Sub", asm: "F32Sub", argLength: 2, reg: fp32_21, typ: "Float32"}, // arg0 - arg1
		{name: "F32Mul", asm: "F32Mul", argLength: 2, reg: fp32_21, typ: "Float32"}, // arg0 * arg1
		{name: "F32Div", asm: "F32Div", argLength: 2, reg: fp32_21, typ: "Float32"}, // arg0 / arg1

		{name: "F64Neg", asm: "F64Neg", argLength: 1, reg: fp64_11, typ: "Float64"}, // -arg0
		{name: "F64Add", asm: "F64Add", argLength: 2, reg: fp64_21, typ: "Float64"}, // arg0 + arg1
		{name: "F64Sub", asm: "F64Sub", argLength: 2, reg: fp64_21, typ: "Float64"}, // arg0 - arg1
		{name: "F64Mul", asm: "F64Mul", argLength: 2, reg: fp64_21, typ: "Float64"}, // arg0 * arg1
		{name: "F64Div", asm: "F64Div", argLength: 2, reg: fp64_21, typ: "Float64"}, // arg0 / arg1

		{name: "I64TruncSatF64S", asm: "I64TruncSatF64S", argLength: 1, reg: regInfo{inputs: []regMask{fp64}, outputs: []regMask{gp}}, typ: "Int64"}, // truncates the float arg0 to a signed integer (saturating)
		{name: "I64TruncSatF64U", asm: "I64TruncSatF64U", argLength: 1, reg: regInfo{inputs: []regMask{fp64}, outputs: []regMask{gp}}, typ: "Int64"}, // truncates the float arg0 to an unsigned integer (saturating)
		{name: "I64TruncSatF32S", asm: "I64TruncSatF32S", argLength: 1, reg: regInfo{inputs: []regMask{fp32}, outputs: []regMask{gp}}, typ: "Int64"}, // truncates the float arg0 to a signed integer (saturating)
		{name: "I64TruncSatF32U", asm: "I64TruncSatF32U", argLength: 1, reg: regInfo{inputs: []regMask{fp32}, outputs: []regMask{gp}}, typ: "Int64"}, // truncates the float arg0 to an unsigned integer (saturating)
		{name: "F32ConvertI64S", asm: "F32ConvertI64S", argLength: 1, reg: regInfo{inputs: []regMask{gp}, outputs: []regMask{fp32}}, typ: "Float32"}, // converts the signed integer arg0 to a float
		{name: "F32ConvertI64U", asm: "F32ConvertI64U", argLength: 1, reg: regInfo{inputs: []regMask{gp}, outputs: []regMask{fp32}}, typ: "Float32"}, // converts the unsigned integer arg0 to a float
		{name: "F64ConvertI64S", asm: "F64ConvertI64S", argLength: 1, reg: regInfo{inputs: []regMask{gp}, outputs: []regMask{fp64}}, typ: "Float64"}, // converts the signed integer arg0 to a float
		{name: "F64ConvertI64U", asm: "F64ConvertI64U", argLength: 1, reg: regInfo{inputs: []regMask{gp}, outputs: []regMask{fp64}}, typ: "Float64"}, // converts the unsigned integer arg0 to a float
		{name: "F32DemoteF64", asm: "F32DemoteF64", argLength: 1, reg: regInfo{inputs: []regMask{fp64}, outputs: []regMask{fp32}}, typ: "Float32"},
		{name: "F64PromoteF32", asm: "F64PromoteF32", argLength: 1, reg: regInfo{inputs: []regMask{fp32}, outputs: []regMask{fp64}}, typ: "Float64"},

		{name: "I64Extend8S", asm: "I64Extend8S", argLength: 1, reg: gp11, typ: "Int64"},   // sign-extend arg0 from 8 to 64 bit
		{name: "I64Extend16S", asm: "I64Extend16S", argLength: 1, reg: gp11, typ: "Int64"}, // sign-extend arg0 from 16 to 64 bit
		{name: "I64Extend32S", asm: "I64Extend32S", argLength: 1, reg: gp11, typ: "Int64"}, // sign-extend arg0 from 32 to 64 bit

		{name: "F32Sqrt", asm: "F32Sqrt", argLength: 1, reg: fp32_11, typ: "Float32"},         // sqrt(arg0)
		{name: "F32Trunc", asm: "F32Trunc", argLength: 1, reg: fp32_11, typ: "Float32"},       // trunc(arg0)
		{name: "F32Ceil", asm: "F32Ceil", argLength: 1, reg: fp32_11, typ: "Float32"},         // ceil(arg0)
		{name: "F32Floor", asm: "F32Floor", argLength: 1, reg: fp32_11, typ: "Float32"},       // floor(arg0)
		{name: "F32Nearest", asm: "F32Nearest", argLength: 1, reg: fp32_11, typ: "Float32"},   // round(arg0)
		{name: "F32Abs", asm: "F32Abs", argLength: 1, reg: fp32_11, typ: "Float32"},           // abs(arg0)
		{name: "F32Copysign", asm: "F32Copysign", argLength: 2, reg: fp32_21, typ: "Float32"}, // copysign(arg0, arg1)

		{name: "F64Sqrt", asm: "F64Sqrt", argLength: 1, reg: fp64_11, typ: "Float64"},         // sqrt(arg0)
		{name: "F64Trunc", asm: "F64Trunc", argLength: 1, reg: fp64_11, typ: "Float64"},       // trunc(arg0)
		{name: "F64Ceil", asm: "F64Ceil", argLength: 1, reg: fp64_11, typ: "Float64"},         // ceil(arg0)
		{name: "F64Floor", asm: "F64Floor", argLength: 1, reg: fp64_11, typ: "Float64"},       // floor(arg0)
		{name: "F64Nearest", asm: "F64Nearest", argLength: 1, reg: fp64_11, typ: "Float64"},   // round(arg0)
		{name: "F64Abs", asm: "F64Abs", argLength: 1, reg: fp64_11, typ: "Float64"},           // abs(arg0)
		{name: "F64Copysign", asm: "F64Copysign", argLength: 2, reg: fp64_21, typ: "Float64"}, // copysign(arg0, arg1)

		{name: "I64Ctz", asm: "I64Ctz", argLength: 1, reg: gp11, typ: "Int64"},       // ctz(arg0)
		{name: "I64Clz", asm: "I64Clz", argLength: 1, reg: gp11, typ: "Int64"},       // clz(arg0)
		{name: "I32Rotl", asm: "I32Rotl", argLength: 2, reg: gp21, typ: "Int32"},     // rotl(arg0, arg1)
		{name: "I64Rotl", asm: "I64Rotl", argLength: 2, reg: gp21, typ: "Int64"},     // rotl(arg0, arg1)
		{name: "I64Popcnt", asm: "I64Popcnt", argLength: 1, reg: gp11, typ: "Int64"}, // popcnt(arg0)

		{name: "V128Load", asm: "V128Load", argLength: 2, reg: vp128load, typ: "Int128"},
		{name: "V128Load8x8S", asm: "V128Load8x8S", argLength: 2, reg: vp128load, typ: "Int128"},
		{name: "V128Load8x8U", asm: "V128Load8x8U", argLength: 2, reg: vp128load, typ: "Int128"},
		{name: "V128Load16x4S", asm: "V128Load16x4S", argLength: 2, reg: vp128load, typ: "Int128"},
		{name: "V128Load16x4U", asm: "V128Load16x4U", argLength: 2, reg: vp128load, typ: "Int128"},
		{name: "V128Load32x2S", asm: "V128Load32x2S", argLength: 2, reg: vp128load, typ: "Int128"},
		{name: "V128Load32x2U", asm: "V128Load32x2U", argLength: 2, reg: vp128load, typ: "Int128"},
		{name: "V128Load8Splat", asm: "V128Load8Splat", argLength: 2, reg: vp128load, typ: "Int128"},
		{name: "V128Load16Splat", asm: "V128Load16Splat", argLength: 2, reg: vp128load, typ: "Int128"},
		{name: "V128Load32Splat", asm: "V128Load32Splat", argLength: 2, reg: vp128load, typ: "Int128"},
		{name: "V128Load64Splat", asm: "V128Load64Splat", argLength: 2, reg: vp128load, typ: "Int128"},

		{name: "V128Store", asm: "V128Store", argLength: 2, reg: vp128store, typ: "Mem"},

		{name: "V128Const", reg: vp128_01, rematerializeable: true, typ: "Int128"}, // returns the constant integer aux
		// Shuffle
		{name: "I8x16Swizzle", asm: "I8x16Swizzle", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I8x16Splat", reg: regInfo{inputs: []regMask{gp}, outputs: []regMask{vp128}}, typ: "Int128"},
		{name: "I16x8Splat", reg: regInfo{inputs: []regMask{gp}, outputs: []regMask{vp128}}, typ: "Int128"},
		{name: "I32x4Splat", reg: regInfo{inputs: []regMask{gp}, outputs: []regMask{vp128}}, typ: "Int128"},
		{name: "I64x2Splat", reg: regInfo{inputs: []regMask{gp}, outputs: []regMask{vp128}}, typ: "Int128"},
		{name: "F32x4Splat", reg: regInfo{inputs: []regMask{fp32}, outputs: []regMask{vp128}}, typ: "Int128"},
		{name: "F64x2Splat", reg: regInfo{inputs: []regMask{fp64}, outputs: []regMask{vp128}}, typ: "Int128"},
		// I8x16ExtractLaneS
		// I8x16ExtractLaneU
		// I8x16ReplaceLane
		// I16x8ExtractLaneS
		// I16x8ExtractLaneU
		// I16x8ReplaceLane
		// I32x4ExtractLane
		// I32x4ReplaceLane
		// I64x2ExtractLane
		// I64x2ReplaceLane
		// F32x4ExtractLane
		// F32x4ReplaceLane
		// F64x2ExtractLane
		// F64x2ReplaceLane
		{name: "I8x16Eq", asm: "I8x16Eq", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I8x16Ne", asm: "I8x16Ne", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I8x16LtS", asm: "I8x16LtS", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I8x16LtU", asm: "I8x16LtU", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I8x16GtS", asm: "I8x16GtS", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I8x16GtU", asm: "I8x16GtU", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I8x16LeS", asm: "I8x16LeS", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I8x16LeU", asm: "I8x16LeU", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I8x16GeS", asm: "I8x16GeS", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I8x16GeU", asm: "I8x16GeU", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I16x8Eq", asm: "I16x8Eq", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I16x8Ne", asm: "I16x8Ne", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I16x8LtS", asm: "I16x8LtS", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I16x8LtU", asm: "I16x8LtU", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I16x8GtS", asm: "I16x8GtS", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I16x8GtU", asm: "I16x8GtU", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I16x8LeS", asm: "I16x8LeS", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I16x8LeU", asm: "I16x8LeU", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I16x8GeS", asm: "I16x8GeS", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I16x8GeU", asm: "I16x8GeU", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I32x4Eq", asm: "I32x4Eq", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I32x4Ne", asm: "I32x4Ne", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I32x4LtS", asm: "I32x4LtS", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I32x4LtU", asm: "I32x4LtU", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I32x4GtS", asm: "I32x4GtS", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I32x4GtU", asm: "I32x4GtU", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I32x4LeS", asm: "I32x4LeS", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I32x4LeU", asm: "I32x4LeU", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I32x4GeS", asm: "I32x4GeS", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I32x4GeU", asm: "I32x4GeU", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "F32x4Eq", asm: "F32x4Eq", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "F32x4Ne", asm: "F32x4Ne", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "F32x4Lt", asm: "F32x4Lt", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "F32x4Gt", asm: "F32x4Gt", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "F32x4Le", asm: "F32x4Le", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "F32x4Ge", asm: "F32x4Ge", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "F64x2Eq", asm: "F64x2Eq", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "F64x2Ne", asm: "F64x2Ne", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "F64x2Lt", asm: "F64x2Lt", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "F64x2Gt", asm: "F64x2Gt", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "F64x2Le", asm: "F64x2Le", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "F64x2Ge", asm: "F64x2Ge", argLength: 2, reg: vp128_21, typ: "Int128"},

		{name: "V128Not", asm: "V128Not", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "V128And", asm: "V128And", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "V128Andnot", asm: "V128Andnot", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "V128Or", asm: "V128Or", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "V128Xor", asm: "V128Xor", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "V128Bitselect", asm: "V128Bitselect", argLength: 3, reg: vp128_31, typ: "Int128"},
		{name: "V128AnyTrue", asm: "V128AnyTrue", argLength: 1, reg: vp128_11gp, typ: "Int128"},
		// V128Load8Lane
		// V128Load16Lane
		// V128Load32Lane
		// V128Load64Lane
		// V128Store8Lane
		// V128Store16Lane
		// V128Store32Lane
		// V128Store64Lane
		// V128Load32Zero
		// V128Load64Zero
		{name: "F32x4DemoteF64x2Zero", asm: "F32x4DemoteF64x2Zero", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "F64x2PromoteLowF32x4", asm: "F64x2PromoteLowF32x4", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "I8x16Abs", asm: "I8x16Abs", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "I8x16Neg", asm: "I8x16Neg", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "I8x16Popcnt", asm: "I8x16Popcnt", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "I8x16AllTrue", asm: "I8x16AllTrue", argLength: 1, reg: vp128_11gp, typ: "Int128"},
		{name: "I8x16Bitmask", asm: "I8x16Bitmask", argLength: 1, reg: vp128_11gp, typ: "Int128"},
		{name: "I8x16NarrowI16x8S", asm: "I8x16NarrowI16x8S", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I8x16NarrowI16x8U", asm: "I8x16NarrowI16x8U", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "F32x4Ceil", asm: "F32x4Ceil", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "F32x4Floor", asm: "F32x4Floor", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "F32x4Trunc", asm: "F32x4Trunc", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "F32x4Nearest", asm: "F32x4Nearest", argLength: 1, reg: vp128_11, typ: "Int128"},
		// I8x16Shl
		// I8x16ShrS
		// I8x16ShrU
		{name: "I8x16Add", asm: "I8x16Add", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I8x16AddSatS", asm: "I8x16AddSatS", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I8x16AddSatU", asm: "I8x16AddSatU", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I8x16Sub", asm: "I8x16Sub", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I8x16SubSatS", asm: "I8x16SubSatS", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I8x16SubSatU", asm: "I8x16SubSatU", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "F64x2Ceil", asm: "F64x2Ceil", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "F64x2Floor", asm: "F64x2Floor", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "I8x16MinS", asm: "I8x16MinS", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I8x16MinU", asm: "I8x16MinU", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I8x16MaxS", asm: "I8x16MaxS", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I8x16MaxU", asm: "I8x16MaxU", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "F64x2Trunc", asm: "F64x2Trunc", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "I8x16AvgrU", asm: "I8x16AvgrU", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I16x8ExtaddPairwiseI8x16S", asm: "I16x8ExtaddPairwiseI8x16S", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "I16x8ExtaddPairwiseI8x16U", asm: "I16x8ExtaddPairwiseI8x16U", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "I32x4ExtaddPairwiseI16x8S", asm: "I32x4ExtaddPairwiseI16x8S", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "I32x4ExtaddPairwiseI16x8U", asm: "I32x4ExtaddPairwiseI16x8U", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "I16x8Abs", asm: "I16x8Abs", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "I16x8Neg", asm: "I16x8Neg", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "I16x8Q15mulrSatS", asm: "I16x8Q15mulrSatS", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I16x8AllTrue", asm: "I16x8AllTrue", argLength: 1, reg: vp128_11gp, typ: "Int128"},
		{name: "I16x8Bitmask", asm: "I16x8Bitmask", argLength: 1, reg: vp128_11gp, typ: "Int128"},
		{name: "I16x8NarrowI32x4S", asm: "I16x8NarrowI32x4S", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I16x8NarrowI32x4U", asm: "I16x8NarrowI32x4U", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I16x8ExtendLowI8x16S", asm: "I16x8ExtendLowI8x16S", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "I16x8ExtendHighI8x16S", asm: "I16x8ExtendHighI8x16S", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "I16x8ExtendLowI8x16U", asm: "I16x8ExtendLowI8x16U", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "I16x8ExtendHighI8x16U", asm: "I16x8ExtendHighI8x16U", argLength: 1, reg: vp128_11, typ: "Int128"},
		// I16x8Shl
		// I16x8ShrS
		// I16x8ShrU
		{name: "I16x8Add", asm: "I16x8Add", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I16x8AddSatS", asm: "I16x8AddSatS", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I16x8AddSatU", asm: "I16x8AddSatU", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I16x8Sub", asm: "I16x8Sub", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I16x8SubSatS", asm: "I16x8SubSatS", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I16x8SubSatU", asm: "I16x8SubSatU", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "F64x2Nearest", asm: "F64x2Nearest", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "I16x8Mul", asm: "I16x8Mul", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I16x8MinS", asm: "I16x8MinS", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I16x8MinU", asm: "I16x8MinU", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I16x8MaxS", asm: "I16x8MaxS", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I16x8MaxU", asm: "I16x8MaxU", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I16x8AvgrU", asm: "I16x8AvgrU", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I16x8ExtmulLowI8x16S", asm: "I16x8ExtmulLowI8x16S", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I16x8ExtmulHighI8x16S", asm: "I16x8ExtmulHighI8x16S", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I16x8ExtmulLowI8x16U", asm: "I16x8ExtmulLowI8x16U", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I16x8ExtmulHighI8x16U", asm: "I16x8ExtmulHighI8x16U", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I32x4Abs", asm: "I32x4Abs", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "I32x4Neg", asm: "I32x4Neg", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "I32x4AllTrue", asm: "I32x4AllTrue", argLength: 1, reg: vp128_11gp, typ: "Int128"},
		{name: "I32x4Bitmask", asm: "I32x4Bitmask", argLength: 1, reg: vp128_11gp, typ: "Int128"},
		{name: "I32x4ExtendLowI16x8S", asm: "I32x4ExtendLowI16x8S", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "I32x4ExtendHighI16x8S", asm: "I32x4ExtendHighI16x8S", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "I32x4ExtendLowI16x8U", asm: "I32x4ExtendLowI16x8U", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "I32x4ExtendHighI16x8U", asm: "I32x4ExtendHighI16x8U", argLength: 1, reg: vp128_11, typ: "Int128"},
		// I32x4Shl
		// I32x4ShrS
		// I32x4ShrU
		{name: "I32x4Add", asm: "I32x4Add", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I32x4Sub", asm: "I32x4Sub", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I32x4Mul", asm: "I32x4Mul", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I32x4MinS", asm: "I32x4MinS", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I32x4MinU", asm: "I32x4MinU", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I32x4MaxS", asm: "I32x4MaxS", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I32x4MaxU", asm: "I32x4MaxU", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I32x4DotI16x8S", asm: "I32x4DotI16x8S", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I32x4ExtmulLowI16x8S", asm: "I32x4ExtmulLowI16x8S", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I32x4ExtmulHighI16x8S", asm: "I32x4ExtmulHighI16x8S", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I32x4ExtmulLowI16x8U", asm: "I32x4ExtmulLowI16x8U", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I32x4ExtmulHighI16x8U", asm: "I32x4ExtmulHighI16x8U", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I64x2Abs", asm: "I64x2Abs", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "I64x2Neg", asm: "I64x2Neg", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "I64x2AllTrue", asm: "I64x2AllTrue", argLength: 1, reg: vp128_11gp, typ: "Int128"},
		{name: "I64x2Bitmask", asm: "I64x2Bitmask", argLength: 1, reg: vp128_11gp, typ: "Int128"},
		{name: "I64x2ExtendLowI32x4S", asm: "I64x2ExtendLowI32x4S", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "I64x2ExtendHighI32x4S", asm: "I64x2ExtendHighI32x4S", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "I64x2ExtendLowI32x4U", asm: "I64x2ExtendLowI32x4U", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "I64x2ExtendHighI32x4U", asm: "I64x2ExtendHighI32x4U", argLength: 1, reg: vp128_11, typ: "Int128"},
		// I64x2Shl
		// I64x2ShrS
		// I64x2ShrU
		{name: "I64x2Add", asm: "I64x2Add", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I64x2Sub", asm: "I64x2Sub", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I64x2Mul", asm: "I64x2Mul", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I64x2Eq", asm: "I64x2Eq", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I64x2Ne", asm: "I64x2Ne", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I64x2LtS", asm: "I64x2LtS", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I64x2GtS", asm: "I64x2GtS", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I64x2LeS", asm: "I64x2LeS", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I64x2ExtmulLowI32x4S", asm: "I64x2ExtmulLowI32x4S", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I64x2ExtmulHighI32x4S", asm: "I64x2ExtmulHighI32x4S", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I64x2ExtmulLowI32x4U", asm: "I64x2ExtmulLowI32x4U", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I64x2ExtmulHighI32x4U", asm: "I64x2ExtmulHighI32x4U", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "F32x4Abs", asm: "F32x4Abs", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "F32x4Neg", asm: "F32x4Neg", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "F32x4Sqrt", asm: "F32x4Sqrt", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "F32x4Add", asm: "F32x4Add", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "F32x4Sub", asm: "F32x4Sub", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "F32x4Mul", asm: "F32x4Mul", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "F32x4Div", asm: "F32x4Div", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "F32x4Min", asm: "F32x4Min", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "F32x4Max", asm: "F32x4Max", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "F32x4Pmin", asm: "F32x4Pmin", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "F32x4Pmax", asm: "F32x4Pmax", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "F64x2Abs", asm: "F64x2Abs", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "F64x2Neg", asm: "F64x2Neg", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "F64x2Sqrt", asm: "F64x2Sqrt", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "F64x2Add", asm: "F64x2Add", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "F64x2Sub", asm: "F64x2Sub", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "F64x2Mul", asm: "F64x2Mul", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "F64x2Div", asm: "F64x2Div", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "F64x2Min", asm: "F64x2Min", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "F64x2Max", asm: "F64x2Max", argLength: 2, reg: vp128_21, typ: "Int128"},
		{name: "I32x4TruncSatF32x4S", asm: "I32x4TruncSatF32x4S", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "I32x4TruncSatF32x4U", asm: "I32x4TruncSatF32x4U", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "F32x4ConvertI32x4S", asm: "F32x4ConvertI32x4S", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "F32x4ConvertI32x4U", asm: "F32x4ConvertI32x4U", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "I32x4TruncSatF64x2SZero", asm: "I32x4TruncSatF64x2SZero", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "I32x4TruncSatF64x2UZero", asm: "I32x4TruncSatF64x2UZero", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "F64x2ConvertLowI32x4S", asm: "F64x2ConvertLowI32x4S", argLength: 1, reg: vp128_11, typ: "Int128"},
		{name: "F64x2ConvertLowI32x4U", asm: "F64x2ConvertLowI32x4U", argLength: 1, reg: vp128_11, typ: "Int128"},
	}

	archs = append(archs, arch{
		name:            "Wasm",
		pkg:             "cmd/internal/obj/wasm",
		genfile:         "../../wasm/ssa.go",
		ops:             WasmOps,
		blocks:          nil,
		regnames:        regNamesWasm,
		gpregmask:       gp,
		fpregmask:       fp32 | fp64,
		fp32regmask:     fp32,
		fp64regmask:     fp64,
		specialregmask:  vp128,
		framepointerreg: -1, // not used
		linkreg:         -1, // not used
	})
}
