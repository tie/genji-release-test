package query_test

import (
	"log"

	"github.com/asdine/genji/database"
	"github.com/asdine/genji/query"
	"github.com/asdine/genji/query/expr"
	"github.com/asdine/genji/query/q"
)

var tx *database.Tx

func ExampleSelect() {
	// SELECT Name, Age FROM example WHERE Age >= 18
	res := query.
		Select().
		From(q.Table("example")).
		Where(q.IntField("Age").Gte(18)).
		Exec(tx)

	if err := res.Err(); err != nil {
		log.Fatal(err)
	}
}

func ExampleAnd() {
	// SELECT Name, Age FROM example WHERE Age >= 18 AND Age < 100
	res := query.
		Select().
		From(q.Table("example")).
		Where(
			expr.And(
				q.IntField("Age").Gte(18),
				q.IntField("Age").Lt(100),
			),
		).
		Exec(tx)

	if err := res.Err(); err != nil {
		log.Fatal(err)
	}
}

func ExampleOr() {
	// SELECT Name, Age FROM example WHERE Age >= 18 OR Group = "staff"
	res := query.
		Select().
		From(q.Table("example")).
		Where(
			expr.Or(
				q.IntField("Age").Gte(18),
				q.StringField("Age").Eq("staff"),
			),
		).
		Exec(tx)

	if err := res.Err(); err != nil {
		log.Fatal(err)
	}
}

func ExampleInsert() {
	// INSERT INTO example (Name, Age) VALUES ("foo", 21)
	res := query.
		Insert().
		Into(q.Table("example")).
		Fields("Name", "Age").
		Values(expr.StringValue("foo"), expr.IntValue(21)).
		Exec(tx)

	if err := res.Err(); err != nil {
		log.Fatal(err)
	}
}

func ExampleDelete() {
	// DELETE FROM example (Name, Age) WHERE Age >= 18
	res := query.
		Delete().
		From(q.Table("example")).
		Where(q.IntField("Age").Gte(18)).
		Exec(tx)

	if res.Err() != nil {
		log.Fatal(res.Err())
	}
}
