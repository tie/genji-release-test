package database

import (
	"strings"

	"github.com/tie/genji-release-test/document"
	"github.com/tie/genji-release-test/stringutil"
)

// FieldConstraint describes constraints on a particular field.
type FieldConstraint struct {
	Path         document.Path
	Type         document.ValueType
	IsPrimaryKey bool
	IsNotNull    bool
	IsUnique     bool // not stored, only set during table creation
	DefaultValue document.Value
	IsInferred   bool
	InferredBy   []document.Path
}

// IsEqual compares f with other member by member.
// Inference is not compared.
func (f *FieldConstraint) IsEqual(other *FieldConstraint) (bool, error) {
	if !f.Path.IsEqual(other.Path) {
		return false, nil
	}

	if f.Type != other.Type {
		return false, nil
	}

	if f.IsPrimaryKey != other.IsPrimaryKey {
		return false, nil
	}

	if f.IsNotNull != other.IsNotNull {
		return false, nil
	}

	if f.HasDefaultValue() != other.HasDefaultValue() {
		return false, nil
	}

	if f.HasDefaultValue() {
		if ok, err := f.DefaultValue.IsEqual(other.DefaultValue); !ok || err != nil {
			return ok, err
		}
	}

	return true, nil
}

func (f *FieldConstraint) String() string {
	var s strings.Builder

	s.WriteString(f.Path.String())
	s.WriteString(" ")
	s.WriteString(f.Type.String())
	if f.IsNotNull {
		s.WriteString(" NOT NULL")
	}
	if f.IsPrimaryKey {
		s.WriteString(" PRIMARY KEY")
	}

	if f.HasDefaultValue() {
		s.WriteString(" DEFAULT ")
		s.WriteString(f.DefaultValue.String())
	}

	return s.String()
}

// MergeInferred adds the other.InferredBy to f.InferredBy and ensures there are no duplicates.
func (f *FieldConstraint) MergeInferred(other *FieldConstraint) {
	for _, by := range other.InferredBy {
		duplicate := false
		for _, fby := range f.InferredBy {
			if fby.IsEqual(by) {
				duplicate = true
				break
			}
		}

		if !duplicate {
			f.InferredBy = append(f.InferredBy, by)
		}
	}
}

// HasDefaultValue returns this field contains a default value constraint.
func (f *FieldConstraint) HasDefaultValue() bool {
	return f.DefaultValue.Type != 0
}

// ToDocument returns a document from f.
func (f *FieldConstraint) ToDocument() document.Document {
	buf := document.NewFieldBuffer()

	buf.Add("path", document.NewArrayValue(pathToArray(f.Path)))
	buf.Add("type", document.NewIntegerValue(int64(f.Type)))
	buf.Add("is_primary_key", document.NewBoolValue(f.IsPrimaryKey))
	buf.Add("is_not_null", document.NewBoolValue(f.IsNotNull))
	if f.HasDefaultValue() {
		buf.Add("default_value", f.DefaultValue)
	}
	buf.Add("is_inferred", document.NewBoolValue(f.IsInferred))
	if f.IsInferred {
		vb := document.NewValueBuffer()
		for _, by := range f.InferredBy {
			vb = vb.Append(document.NewArrayValue(pathToArray(by)))
		}
		buf.Add("inferred_by", document.NewArrayValue(vb))
	}
	return buf
}

// ScanDocument implements the document.Scanner interface.
func (f *FieldConstraint) ScanDocument(d document.Document) error {
	v, err := d.GetByField("path")
	if err != nil {
		return err
	}
	f.Path, err = arrayToPath(v.V.(document.Array))
	if err != nil {
		return err
	}

	v, err = d.GetByField("type")
	if err != nil {
		return err
	}
	tp := v.V.(int64)
	f.Type = document.ValueType(tp)

	v, err = d.GetByField("is_primary_key")
	if err != nil {
		return err
	}
	f.IsPrimaryKey = v.V.(bool)

	v, err = d.GetByField("is_not_null")
	if err != nil {
		return err
	}
	f.IsNotNull = v.V.(bool)

	v, err = d.GetByField("default_value")
	if err != nil && err != document.ErrFieldNotFound {
		return err
	}
	if err == nil {
		f.DefaultValue = v
	}

	v, err = d.GetByField("is_inferred")
	if err != nil && err != document.ErrFieldNotFound {
		return err
	}
	if err == nil {
		f.IsInferred = v.V.(bool)
	}

	v, err = d.GetByField("inferred_by")
	if err != nil && err != document.ErrFieldNotFound {
		return err
	}
	if err == nil {
		v.V.(document.Array).Iterate(func(i int, value document.Value) error {
			by, err := arrayToPath(value.V.(document.Array))
			if err != nil {
				return err
			}
			f.InferredBy = append(f.InferredBy, by)
			return nil
		})
	}

	return nil
}

// FieldConstraints is a list of field constraints.
type FieldConstraints []*FieldConstraint

// NewFieldConstraints takes user-defined field constraints, validates them, infers additional
// constraints if needed, and returns a valid FieldConstraints type that can be assigned to a table.
func NewFieldConstraints(userConstraints []*FieldConstraint) (FieldConstraints, error) {
	return FieldConstraints(userConstraints).Infer()
}

// Get a field constraint by path. Returns nil if not found.
func (f FieldConstraints) Get(path document.Path) *FieldConstraint {
	for _, fc := range f {
		if fc.Path.IsEqual(path) {
			return fc
		}
	}

	return nil
}

// Infer additional constraints based on user defined ones.
// For example, given the following table:
//   CREATE TABLE foo (a.b[0] TEXT)
// this function will return a TableInfo that behaves as if the table
// had been created like this:
//   CREATE TABLE foo(
//      a DOCUMENT
//      a.b ARRAY
//      a.b[0] TEXT
//   )
func (f FieldConstraints) Infer() (FieldConstraints, error) {
	newConstraints := make(FieldConstraints, 0, len(f))

	for _, fc := range f {
		// loop over all the path fragments and
		// create intermediary inferred constraints.
		if len(fc.Path) > 1 {
			for i := range fc.Path {
				// stop before reaching the last fragment
				// which will be added outside of this loop
				if i+1 == len(fc.Path) {
					break
				}

				newFc := FieldConstraint{
					Path:       fc.Path[:i+1],
					IsInferred: true,
					InferredBy: []document.Path{fc.Path},
				}
				if fc.Path[i+1].FieldName != "" {
					newFc.Type = document.DocumentValue
				} else {
					newFc.Type = document.ArrayValue
				}

				err := newConstraints.Add(&newFc)
				if err != nil {
					return nil, err
				}
			}
		}

		// add the non inferred path to the list
		// and ensure there are no conflicts with
		// existing ones.
		err := newConstraints.Add(fc)
		if err != nil {
			return nil, err
		}
	}

	return newConstraints, nil
}

// Add a field constraint to the list. If another constraint exists for the same path
// and they are equal, newFc will be ignored. Otherwise an error will be returned.
// If newFc has been inferred by another constraint and another constraint exists with the same
// path, their InferredBy member will be merged.
func (f *FieldConstraints) Add(newFc *FieldConstraint) error {
	for i, c := range *f {
		if c.Path.IsEqual(newFc.Path) {
			// if both non inferred, they are duplicate
			if !newFc.IsInferred && !c.IsInferred {
				return stringutil.Errorf("conflicting constraints: %q and %q", c.String(), newFc.String())
			}

			// determine which one is inferred
			inferredFc, nonInferredFc := c, newFc
			if newFc.IsInferred {
				inferredFc, nonInferredFc = nonInferredFc, inferredFc
			}

			// the inferred one may have less constraints that the user-defined one
			inferredFc.DefaultValue = nonInferredFc.DefaultValue
			inferredFc.IsNotNull = nonInferredFc.IsNotNull
			inferredFc.IsPrimaryKey = nonInferredFc.IsPrimaryKey

			// safe-guard in case we add more fields to the struct
			ok, err := c.IsEqual(newFc)
			if err != nil {
				return err
			}

			// if constraints are different
			if !ok {
				return stringutil.Errorf("conflicting constraints: %q and %q", c.String(), newFc.String())
			}

			// if both inferred, merge the InferredBy member
			if newFc.IsInferred && c.IsInferred {
				c.MergeInferred(newFc)
				return nil
			}

			// if existing one is not inferred, ignore newFc
			if newFc.IsInferred && !c.IsInferred {
				return nil
			}

			// if existing one is inferred, and newFc is not,
			// replace it
			(*f)[i] = newFc
			return nil
		}

		// ensure we don't have duplicate primary keys
		if c.IsPrimaryKey && newFc.IsPrimaryKey {
			return stringutil.Errorf(
				"multiple primary keys are not allowed (%q is primary key)",
				c.Path.String(),
			)
		}
	}

	// convert default values to the right types
	targetType := newFc.Type

	// if there is no type constraint, numbers must be converted to double
	if newFc.DefaultValue.Type == document.IntegerValue && newFc.Type == 0 {
		targetType = document.DoubleValue
	}
	if newFc.DefaultValue.Type != 0 && targetType != 0 {
		var err error
		newFc.DefaultValue, err = newFc.DefaultValue.CastAs(targetType)
		if err != nil {
			return err
		}
	}

	*f = append(*f, newFc)
	return nil
}

// ValidateDocument calls Convert then ensures the document validates against the field constraints.
func (f FieldConstraints) ValidateDocument(d document.Document) (*document.FieldBuffer, error) {
	fb, err := f.ConvertDocument(d)
	if err != nil {
		return nil, err
	}

	// ensure no field is missing
	for _, fc := range f {
		v, err := fc.Path.GetValueFromDocument(fb)
		if err == nil {
			// if field is found, it has already been converted
			// to the right type above.
			// check if it is required but null.
			if v.Type == document.NullValue && fc.IsNotNull {
				return nil, stringutil.Errorf("field %q is required and must be not null", fc.Path)
			}

			continue
		}

		if err != document.ErrFieldNotFound {
			return nil, err
		}

		// if field is not found
		// check if there is a default value
		if fc.DefaultValue.Type != 0 {
			err = fb.Set(fc.Path, fc.DefaultValue)
			if err != nil {
				return nil, err
			}
			// if there is no default value
			// check if field is required
		} else if fc.IsNotNull {
			return nil, stringutil.Errorf("field %q is required and must be not null", fc.Path)
		}
	}

	return fb, nil
}

// ConvertDocument the document using the field constraints.
// It converts any path that has a field constraint on it into the specified type using CAST.
// If there is no constraint on an integer field or value, it converts it into a double.
// Default values on missing fields are not applied.
func (f FieldConstraints) ConvertDocument(d document.Document) (*document.FieldBuffer, error) {
	return f.convertDocumentAtPath(nil, d, CastConversion)
}

// ConversionFunc is called when the type of a value is different than the expected type
// and the value needs to be converted.
type ConversionFunc func(v document.Value, path document.Path, targetType document.ValueType) (document.Value, error)

// CastConversion is a ConversionFunc that casts the value to the target type.
func CastConversion(v document.Value, path document.Path, targetType document.ValueType) (document.Value, error) {
	newV, err := v.CastAs(targetType)
	if err != nil {
		return v, stringutil.Errorf("field %q must be of type %q, got %q", path, targetType, v.Type)
	}

	return newV, nil
}

// LosslessConversion is a ConversionFunc that only converts numbers if there is no precision loss in the process.
func LosslessNumbersConversion(v document.Value, path document.Path, targetType document.ValueType) (document.Value, error) {
	if v.Type == document.IntegerValue && targetType == document.DoubleValue {
		return v.CastAsDouble()
	}

	if v.Type == document.DoubleValue && targetType == document.IntegerValue {
		f := v.V.(float64)
		if float64(int64(f)) == f {
			return v.CastAsInteger()
		}
	}

	return v, nil
}

// ConvertValueAtPath converts the value using the field constraints that are applicable
// at the given path.
func (f FieldConstraints) ConvertValueAtPath(path document.Path, v document.Value, conversionFn ConversionFunc) (document.Value, error) {
	switch v.Type {
	case document.ArrayValue:
		vb, err := f.convertArrayAtPath(path, v.V.(document.Array), conversionFn)
		return document.NewArrayValue(vb), err
	case document.DocumentValue:
		fb, err := f.convertDocumentAtPath(path, v.V.(document.Document), conversionFn)
		return document.NewDocumentValue(fb), err
	}
	return f.convertScalarAtPath(path, v, conversionFn)
}

// convert the value using field constraints type information.
// if there is a type constraint on a path, apply it.
// if a value is an integer and has no constraint, convert it to double.
func (f FieldConstraints) convertScalarAtPath(path document.Path, v document.Value, conversionFn ConversionFunc) (document.Value, error) {
	for _, fc := range f {
		if !fc.Path.IsEqual(path) {
			continue
		}

		// check if the constraint enforce a particular type
		// and if so convert the value to the new type.
		if fc.Type != 0 {
			newV, err := conversionFn(v, fc.Path, fc.Type)
			if err != nil {
				return v, err
			}

			return newV, nil
		}
		break
	}

	// no constraint have been found for this path.
	// check if this is an integer and convert it to double.
	if v.Type == document.IntegerValue {
		newV, _ := v.CastAsDouble()
		return newV, nil
	}

	return v, nil
}

func (f FieldConstraints) convertDocumentAtPath(path document.Path, d document.Document, conversionFn ConversionFunc) (*document.FieldBuffer, error) {
	fb := document.NewFieldBuffer()
	err := fb.Copy(d)
	if err != nil {
		return nil, err
	}

	err = fb.Apply(func(p document.Path, v document.Value) (document.Value, error) {
		return f.convertScalarAtPath(append(path, p...), v, conversionFn)
	})

	return fb, err
}

func (f FieldConstraints) convertArrayAtPath(path document.Path, a document.Array, conversionFn ConversionFunc) (*document.ValueBuffer, error) {
	vb := document.NewValueBuffer()
	err := vb.Copy(a)
	if err != nil {
		return nil, err
	}

	err = vb.Apply(func(p document.Path, v document.Value) (document.Value, error) {
		return f.convertScalarAtPath(append(path, p...), v, conversionFn)
	})

	return vb, err
}
