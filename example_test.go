package velr_test

import (
	"fmt"
	"log"
	"unsafe"

	velr "github.com/velr-ai/velr-go-driver"
)

func ExampleOpenInMemory() {
	db, err := velr.OpenInMemory()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.RunWithParams("CREATE (:Person {name: $name, born: $born})", velr.Params{
		"name": "Keanu Reeves",
		"born": 1964,
	}); err != nil {
		log.Fatal(err)
	}

	rows, err := db.Query("MATCH (p:Person) RETURN p.name AS name, p.born AS born")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(rows[0]["name"], rows[0]["born"])
}

func ExampleOpenReadonly() {
	db, err := velr.OpenReadonly("graph.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query("MATCH (n) RETURN count(n) AS nodes")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(rows[0]["nodes"])
}

func ExampleDB_Transaction() {
	db, err := velr.OpenInMemory()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Transaction(func(tx *velr.Tx) error {
		if err := tx.Run("CREATE (:Account {name: 'checking', balance: 100})"); err != nil {
			return err
		}
		return tx.Run("CREATE (:Account {name: 'savings', balance: 50})")
	})
	if err != nil {
		log.Fatal(err)
	}
}

func ExampleTx_Savepoint() {
	db, err := velr.OpenInMemory()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	tx, err := db.BeginTx()
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Close()

	if err := tx.Run("CREATE (:Item {name: 'kept'})"); err != nil {
		log.Fatal(err)
	}
	sp, err := tx.Savepoint()
	if err != nil {
		log.Fatal(err)
	}
	if err := tx.Run("CREATE (:Item {name: 'discarded'})"); err != nil {
		log.Fatal(err)
	}
	if err := sp.Rollback(); err != nil {
		log.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}
}

func ExampleRows_NextInto() {
	db, err := velr.OpenInMemory()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	table, err := db.ExecOne("RETURN 42 AS answer, 'Velr' AS name")
	if err != nil {
		log.Fatal(err)
	}
	defer table.Close()

	rows, err := table.Rows()
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var answer int64
	var name string
	ok, err := rows.NextInto(&answer, &name)
	if err != nil {
		log.Fatal(err)
	}
	if ok {
		fmt.Println(answer, name)
	}
}

func ExampleCell_AsProperty() {
	db, err := velr.OpenInMemory()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	table, err := db.ExecOne("CREATE (p:Person {name: 'Ada'}) RETURN p")
	if err != nil {
		log.Fatal(err)
	}
	defer table.Close()

	cells, err := table.Collect()
	if err != nil {
		log.Fatal(err)
	}
	value, err := cells[0][0].AsProperty()
	if err != nil {
		log.Fatal(err)
	}
	node := value.(velr.Node)
	fmt.Println(node.Labels, node.Properties["name"].GoValue())
}

func ExampleDB_RegisterVectorEmbedder() {
	db, err := velr.OpenInMemory()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.RegisterVectorEmbedder("demo", func(inputs []velr.VectorEmbeddingInput) ([][]float32, error) {
		vectors := make([][]float32, len(inputs))
		for i, input := range inputs {
			vector := make([]float32, input.Dimensions)
			if input.Text() != "" && len(vector) > 0 {
				vector[0] = 1
			}
			vectors[i] = vector
		}
		return vectors, nil
	})
	if err != nil {
		log.Fatal(err)
	}

	err = db.Run(`
		CREATE (:Paper {title: 'Graph retrieval'})
		CREATE VECTOR INDEX paperEmbedding IF NOT EXISTS
		FOR (p:Paper) ON EACH [p.title]
		OPTIONS {indexConfig: {dimensions: 3, embedder: 'demo'}}
	`)
	if err != nil {
		log.Fatal(err)
	}
}

func ExampleDB_BindArrowIPC() {
	db, err := velr.OpenInMemory()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var ipcFileBytes []byte
	if err := db.BindArrowIPC("people", ipcFileBytes); err != nil {
		log.Fatal(err)
	}

	rows, err := db.Query("UNWIND BIND('people') AS row RETURN row.name AS name")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(rows)
}

func ExampleDB_BindArrow() {
	db, err := velr.OpenInMemory()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var schemaPtr unsafe.Pointer
	var arrayPtr unsafe.Pointer
	err = db.BindArrow("people", []velr.ArrowColumn{{
		Name:   "name",
		Schema: schemaPtr,
		Array:  arrayPtr,
	}})
	if err != nil {
		log.Fatal(err)
	}
}

func ExampleDB_Explain() {
	db, err := velr.OpenInMemory()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	trace, err := db.Explain("MATCH (p:Person) RETURN p.name")
	if err != nil {
		log.Fatal(err)
	}
	defer trace.Close()

	text, err := trace.CompactString()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(text)
}
