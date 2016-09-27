package events

import (
	"bytes"
	"fmt"

	"golang.org/x/net/trace"

	basictracer "github.com/opentracing/basictracer-go"
)

// NetTraceIntegrator can be passed into a basictracer as NewSpanEventListener
// and causes all traces to be registered with the net/trace endpoint.
var NetTraceIntegrator = func() func(basictracer.SpanEvent) {
	var tr trace.Trace
	return func(e basictracer.SpanEvent) {
		switch t := e.(type) {
		case basictracer.EventCreate:
			tr = trace.New("tracing", t.OperationName)
		case basictracer.EventFinish:
			tr.Finish()
		case basictracer.EventTag:
			tr.LazyPrintf("%s:%v", t.Key, t.Value)
		case basictracer.EventLogFields:
			var buf bytes.Buffer

			// Search for an "event" or "error" field for the "main" message.
			// "event" is the key that corresponds to legacy LogEvents.
			// "error" is the key used for error fields (opentracing-go/log.Error).
			mainIdx := -1
			for i, f := range t.Fields {
				key := f.Key()
				if key == "error" || key == "event" {
					mainIdx = i
					if key == "error" {
						buf.WriteString("error: ")
						tr.SetError()
					}
					fmt.Fprint(&buf, f.Value())
					break
				}
			}

			// If we have a "main" message, the format of the message will be:
			//   main message (otherfield1:value, otherfield2:value, otherfield3:value)
			// If we don't, the message will be simply:
			//   field1:value, field2:value, field3:value

			var sep string
			if mainIdx != -1 {
				sep = " ("
			}

			for i, f := range t.Fields {
				if i == mainIdx {
					continue
				}
				buf.WriteString(sep)
				sep = ", "

				key := f.Key()
				if key == "error" {
					tr.SetError()
				}
				fmt.Fprintf(&buf, "%s:%v", key, f.Value())
			}

			if mainIdx != -1 && sep != " (" {
				buf.WriteByte(')')
			}
			tr.LazyPrintf("%s", buf.String())
		case basictracer.EventLog:
			if t.Payload != nil {
				tr.LazyPrintf("%s (payload %v)", t.Event, t.Payload)
			} else {
				tr.LazyPrintf("%s", t.Event)
			}
		}
	}
}
