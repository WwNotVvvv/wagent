package app

type Harness struct {
	cfg   *Config
	llm   LLM
	guard *Guardrail
	hitl  *HITL
	tools *ToolRegistry
	verif *Verifier
	trace *TraceRecorder
	ctx   *Context
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
}
