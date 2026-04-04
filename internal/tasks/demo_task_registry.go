package tasks

type DemoTaskHandlersRegistry struct {
	handlers map[string]HandlerFunc
}

func NewDemoRegistry() *DemoTaskHandlersRegistry {
	handlers := make(map[string]HandlerFunc)
	handlers["echo"] = Echo
	handlers["send_email"] = SendEmail
	handlers["always_fail"] = AlwaysFails
	handlers["slow_task"] = SlowTask

	return &DemoTaskHandlersRegistry{handlers: handlers}
}

func (r *DemoTaskHandlersRegistry) Get(taskType string) (HandlerFunc, bool) {
	handler, ok := r.handlers[taskType]
	return handler, ok
}
