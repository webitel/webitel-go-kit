package sql

// Option allows for managing otelsql configuration using functional options.
type Option interface {
	apply(c *config)
}

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

// config holds configuration of our otelsql tracing middleware.
//
// By default, all options are set to false intentionally when creating a wrapped driver
// and provide the most sensible default with both performance and security in mind.
type config struct {
	tracer MethodTracer

	// allowRoot, if set to true, will allow otelsql to create root spans in absence of existing spans or even context.
	//
	// Default is to not trace otelsql calls if no existing parent span is found in context or when using methods not taking context.
	allowRoot bool

	// ping, if set to true, will enable the creation of spans on Ping requests.
	ping bool

	// rowsNext, if set to true, will enable the creation of spans on RowsNext calls. This can result in many spans.
	rowsNext bool

	// rowsClose, if set to true, will enable the creation of spans on RowsClose calls.
	rowsClose bool

	// rowsAffected, if set to true, will enable the creation of spans on RowsAffected calls.
	rowsAffected bool

	// lastInsertID, if set to true, will enable the creation of spans on LastInsertId calls.
	lastInsertID bool
}

func WithTracer(tracer MethodTracer) Option {
	return optionFunc(func(c *config) {
		c.tracer = tracer
	})
}

// WithAllowRoot allows otelsql to create root spans in absence of existing spans or even context.
//
// Default is to not trace otelsql calls if no existing parent span is found in context or when using methods not taking context.
func WithAllowRoot(allow bool) Option {
	return optionFunc(func(o *config) {
		o.allowRoot = allow
	})
}

// TracePing enables the creation of spans on Ping requests.
func TracePing() Option {
	return optionFunc(func(o *config) {
		o.ping = true
	})
}

// TraceRowsAffected enables the creation of spans on RowsAffected calls.
func TraceRowsAffected() Option {
	return optionFunc(func(o *config) {
		o.rowsAffected = true
	})
}

// TraceLastInsertID enables the creation of spans on LastInsertId calls.
func TraceLastInsertID() Option {
	return optionFunc(func(o *config) {
		o.lastInsertID = true
	})
}

// TraceAll enables the creation of spans on methods.
func TraceAll() Option {
	return optionFunc(func(o *config) {
		o.allowRoot = true
		o.ping = true
		o.rowsNext = true
		o.rowsClose = true
		o.rowsAffected = true
		o.lastInsertID = true
	})
}
