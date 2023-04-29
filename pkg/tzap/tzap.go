package tzap

import (
	"context"

	"github.com/tzapio/tzap/pkg/config"
	"github.com/tzapio/tzap/pkg/types"
)

var count = 1

func addId(t *Tzap) {
	if t.Id > 0 {
		println("Tzap already has an id", t.Id, t.Name)
		panic(t)
	}
	t.Id = count
	count += 1
	println("Added tzap", t.Id, t.Name)
	GlobalTzaps = append(GlobalTzaps, t)
}

// Tzap is a structure that holds data and methods related to Tzap objects.
type Tzap struct {
	Id      int
	Name    string
	Header  string
	Message types.Message
	Data    types.MappedInterface `json:"-"`
	C       context.Context       `json:"-"`

	types.ITzap[*Tzap] `json:"-"`

	Parent *Tzap

	TG types.TGenerator
}

// NewTzap creates a new Tzap with default values, and returns its pointer.
// Mainly for mocking purposes. Does not have a connector, will likely crash.
func InternalNew() *Tzap {
	t := &Tzap{
		Name:    "ConnectionLess",
		Message: types.Message{},
		Data:    types.MappedInterface{},
		C:       context.Background(),
	}
	addId(t)
	return t
}

func NewWithConnector(connector types.TzapConnector) *Tzap {
	tg, conf := connector()
	t := &Tzap{
		Name:    "Connection",
		Message: types.Message{},
		Data:    types.MappedInterface{},
		C:       config.NewContext(context.Background(), conf),
		TG:      tg,
	}
	addId(t)
	return t
}

// New returns a new Tzap with default values.
func (t *Tzap) New() *Tzap {
	tc := &Tzap{
		Name:    "NewConnection",
		Message: types.Message{},
		Data:    types.MappedInterface{},
		C:       context.Background(),
		TG:      t.TG,
	}
	addId(tc)
	return tc
}

// AppendParentContext assigns the parent's context to the Tzap object, if present.
func (t *Tzap) AppendParentContext() *Tzap {
	if t.Parent != nil {
		t.C = t.Parent.C
		t.TG = t.Parent.TG
	}
	return t
}

// onNewTzap is a helper method that appends the parent's context to the Tzap object.
func (t *Tzap) onNew() *Tzap {

	return t.AppendParentContext()
}

// AddTzap (mostly internal use) initializes and adds a new Tzap child to the current Tzap object.
func (t *Tzap) AddTzap(tc *Tzap) *Tzap {
	Logf(t, "Add tzap (%s)", tc.Name)
	addId(tc)
	tc.Parent = t
	return (tc).onNew()
}

// CloneTzap (mostly internal use) clones a Tzap object and assigns values based on the provided Tzap object.
func (t *Tzap) CloneTzap(tc *Tzap) *Tzap {
	Logf(t, "Clone tzap (%s)", tc.Name)
	tz := &Tzap{
		Parent:  t,
		Name:    t.Name,
		Header:  t.Header,
		Message: t.Message,
		Data:    t.Data,
	}

	if tc.Parent != nil {
		tz.Parent = tc.Parent
	}
	if tc.Name != "" {
		tz.Name = tc.Name
	}
	if tc.Header != "" {
		tz.Header = tc.Header
	}
	if tc.Message.Role != "" {
		tz.Message.Role = tc.Message.Role
	}
	if tc.Message.Content != "" {
		tz.Message.Content = tc.Message.Content
	}

	if len(tc.Data) > 0 {
		tz.Data = tc.Data
	}
	addId(tz)
	return tz.onNew()
}

// HijackTzap (mostly internal use) effectively de-attaches from previous Tzap by changing the own parent to parents parent.
// This can be used AddUserMessage("H").LoadTaskOrRequestNewTask().Hijack() .() Tzap replaces the current Tzap's context and parent with the provided Tzap's context and parent.
func (t *Tzap) HijackTzap(tc *Tzap) *Tzap {
	Logf(t, "Hijack tzap (%s)", tc.Name)
	tc.C = t.C
	tc.TG = t.TG
	tc.Parent = t.Parent
	addId(tc)
	return tc
}
