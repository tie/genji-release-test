package expr

import (
	"github.com/tie/genji-release-test/database"
	"github.com/tie/genji-release-test/document"
	"github.com/tie/genji-release-test/stringutil"
)

// Environment contains information about the context in which
// the expression is evaluated.
type Environment struct {
	Params []Param
	Vars   *document.FieldBuffer
	Doc    document.Document
	Tx     *database.Transaction

	Outer *Environment
}

func NewEnvironment(d document.Document, params ...Param) *Environment {
	env := Environment{
		Params: params,
		Doc:    d,
	}

	return &env
}

func (e *Environment) Get(path document.Path) (v document.Value, ok bool) {
	if e.Vars != nil {
		v, err := path.GetValueFromDocument(e.Vars)
		if err == nil {
			return v, true
		}
	}

	if e.Outer != nil {
		return e.Outer.Get(path)
	}

	return
}

func (e *Environment) Set(name string, v document.Value) {
	if e.Vars == nil {
		e.Vars = document.NewFieldBuffer()
	}

	e.Vars.Set(document.Path{document.PathFragment{FieldName: name}}, v)
}

func (e *Environment) GetDocument() (document.Document, bool) {
	if e.Doc != nil {
		return e.Doc, true
	}

	if e.Outer != nil {
		return e.Outer.GetDocument()
	}

	return nil, false
}

func (e *Environment) SetDocument(d document.Document) {
	e.Doc = d
}

func (e *Environment) GetParamByName(name string) (v document.Value, err error) {
	if len(e.Params) == 0 {
		if e.Outer != nil {
			return e.Outer.GetParamByName(name)
		}
	}

	for _, nv := range e.Params {
		if nv.Name == name {
			return document.NewValue(nv.Value)
		}
	}

	return document.Value{}, stringutil.Errorf("param %s not found", name)
}

func (e *Environment) GetParamByIndex(pos int) (document.Value, error) {
	if len(e.Params) == 0 {
		if e.Outer != nil {
			return e.Outer.GetParamByIndex(pos)
		}
	}

	idx := int(pos - 1)
	if idx >= len(e.Params) {
		return document.Value{}, stringutil.Errorf("cannot find param number %d", pos)
	}

	return document.NewValue(e.Params[idx].Value)
}

func (e *Environment) GetTx() *database.Transaction {
	if e.Tx != nil {
		return e.Tx
	}

	if e.Outer != nil {
		return e.Outer.GetTx()
	}

	return nil
}

func (e *Environment) Clone() (*Environment, error) {
	newEnv := Environment{
		Params: e.Params,
	}

	newEnv.Tx = e.Tx

	if e.Doc != nil {
		fb := document.NewFieldBuffer()
		err := fb.Copy(e.Doc)
		if err != nil {
			return nil, err
		}

		newEnv.Doc = fb
	}

	if e.Vars != nil {
		fb := document.NewFieldBuffer()
		err := fb.Copy(e.Vars)
		if err != nil {
			return nil, err
		}

		newEnv.Vars = fb
	}

	if e.Outer != nil {
		newOuter, err := e.Outer.Clone()
		if err != nil {
			return nil, err
		}
		newEnv.Outer = newOuter
	}

	return &newEnv, nil
}
