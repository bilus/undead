package htmx

import (
	g "github.com/maragudk/gomponents"
	hx "github.com/maragudk/gomponents-htmx"
	h "github.com/maragudk/gomponents/html"
)

func ActionOn(trigger, event string) g.Node {
	return fragment(h.Method("POST"), hx.Post(""), hx.Boost("true"), hx.Target("body"), hx.Trigger(trigger),
		h.Input(h.Type("hidden"), h.Name("event"), h.Value(event)))
}

func fragment(nodes ...g.Node) g.Node {
	return g.Group(nodes)
}
