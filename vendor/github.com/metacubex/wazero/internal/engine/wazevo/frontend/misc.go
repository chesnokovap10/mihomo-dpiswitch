package frontend

import (
	"github.com/metacubex/wazero/internal/engine/wazevo/ssa"
	"github.com/metacubex/wazero/internal/wasm"
)

func FunctionIndexToFuncRef(idx wasm.Index) ssa.FuncRef {
	return ssa.FuncRef(idx)
}
