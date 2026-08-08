package app

import (
	"crypto/rand"
	"encoding/hex"
)

type Harness struct {
	cfg      *Config
	llm      LLM
	guard    *Guardrail
	hitl     *HITL
	tools    *ToolRegistry
	verif    *Verifier
	trace    *TraceRecorder
	ctx      *Context
	OnStep   func(StepEvent)
	taskIdx  int
	redactFn func(string) string
}

func NewHarness(cfg *Config, llm LLM) *Harness {
	return &Harness{
		cfg:   cfg,
		llm:   llm,
		guard: &Guardrail{},
		hitl:  &HITL{},
		tools: NewToolRegistry(),
		verif: &Verifier{},
		ctx:   NewContext(),
	}
}

func (h *Harness) SetTraceRecorder(tr *TraceRecorder) {
	h.trace = tr
	if h.redactFn != nil {
		tr.SetRedactFunc(h.redactFn)
	}
}

func (h *Harness) SetRedactFunc(fn func(string) string) {
	h.redactFn = fn
	h.tools.SetRedactFunc(fn)
	h.verif.SetRedactFunc(fn)
	if h.trace != nil {
		h.trace.SetRedactFunc(fn)
	}
}

func (h *Harness) SetOnStep(fn func(StepEvent)) {
	h.OnStep = fn
}

func (h *Harness) onStep(ev StepEvent) {
	if h.OnStep != nil {
		h.OnStep(ev)
	}
}

func (h *Harness) Reset() {
	h.ctx = NewContext()
}

func (h *Harness) nextTaskID() string {
	h.taskIdx++
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}
