package go_dremel

import (
  "fmt"
  "strings"
)

type RecordChildren map[interface{}]*Record

type Record struct {
  Name string
  Parent *Record
  Children RecordChildren
  Value interface{}
  Values []interface{}
}
type RepetitionLevel int

func MakeReaders(columns map[string][]Row, fields []ProcessedField, fsm FSM2) []*Reader {
  readers := []*Reader{}

  for _, field := range fields {
    reader := Reader{field, columns[field.Path], fsm[field], 0}
    readers = append(readers, &reader)
  }

  return readers
}

type FieldFSM map[int]ProcessedField

type Reader struct {
  Field ProcessedField
  Rows []Row
  FSM FieldRepetitionLevelTransitions
  CurrentRowIndex int
}

var EmptyReader = &Reader{}

func (reader *Reader) HasData() bool {
  return reader.Field != ProcessedField{}
}

func (reader *Reader) FetchNextRow() Row {
  row := reader.CurrentRow()

  reader.CurrentRowIndex += 1
  return row
}

func (reader *Reader) CurrentRow() Row {
  if reader.CurrentRowIndex < len(reader.Rows) {
    return reader.Rows[reader.CurrentRowIndex]
  } else {
    return Row{}
  }
}

func (reader *Reader) NextRow() Row {
  nextIndex := reader.CurrentRowIndex + 1
  if nextIndex < len(reader.Rows) {
    return reader.Rows[nextIndex]
  } else {
    return Row{}
  }
}

func (reader *Reader) NextRepetionLevel() int {
  nextRow := reader.NextRow()
  return nextRow.RepetitionLevel
}

func findReaderByField(field ProcessedField, readers []*Reader) *Reader {
  reader := EmptyReader
  for i := 0; i < len(readers); i++ {
    r := readers[i]
    if r.Field == field {
      reader = r
    }
  }
  return reader
}


func (reader *Reader) NextReader(readers []*Reader) *Reader {
    destinationField := reader.FSM[reader.NextRepetionLevel()]
    destinationReader := findReaderByField(destinationField, readers)
    return destinationReader
}

func (record *Record) ToMap() {
  if record.Value != nil {
    fmt.Printf("%v-%v\n", record.Name, record.Value)
  }
  if record.Values != nil {
    fmt.Printf("%v-%v\n", record.Name, record.Values)
  }

  for _, r := range record.Children {
    r.ToMap()
  }
}

func countNonEmptyStrings(strings []string) int {
  count := 0
  for i := 0; i < len(strings); i++ {
    if strings[i] != "" {
      count++
    }
  }
  return count
}

func moveToLevel(record *Record, nextReader *Reader, lastReader *Reader, lowestCommonAncestor *Reader) (*Record) {
    commonPath := lowestCommonAncestor.Field.Path
    nextPath := nextReader.Field.Path
    lastPath := lastReader.Field.Path
    commonPaths := strings.Split(commonPath, ".")
    nextPaths := strings.Split(nextPath, ".")
    lastPaths := strings.Split(lastPath, ".")

    // end nested records up to lowest common ancestor of next and last reader
    for index := countNonEmptyStrings(lastPaths); index > countNonEmptyStrings(commonPaths); index-- {
      record = record.Parent
    }

    // start nested records up from lowest common ancestor to nextReader.Path
    for index := countNonEmptyStrings(commonPaths); index < countNonEmptyStrings(nextPaths); index++ {
      name := nextPaths[index]
      record.Children[name] = &Record{Name:name, Children:RecordChildren{}, Parent: record}
      record = record.Children[name]
    }

    // set lastReader to one at newLevel
    lastReader = nextReader

    return record
}

func returnToLevel(record *Record, nextReader *Reader, lastReader *Reader, lowestCommonAncestor *Reader) (*Record) {
    commonPath := lowestCommonAncestor.Field.Path
    //nextPath := nextReader.Field.Path
    lastPath := lastReader.Field.Path
    commonPaths := strings.Split(commonPath, ".")
    //nextPaths := strings.Split(nextPath, ".")
    lastPaths := strings.Split(lastPath, ".")

    // end nested records up to lowest common ancestor of next and last reader
    for index := countNonEmptyStrings(lastPaths); index > countNonEmptyStrings(commonPaths); index-- {
      record = record.Parent
    }

    // set lastReader to one at newLevel
    lastReader = nextReader

    return record
}

func getLowestCommonReaderAncestor(r1 *Reader, r2 *Reader, readers []*Reader) *Reader {
  commonFieldAncestor := GetLowestCommonAncestor(r1.Field, r2.Field)
  return findReaderByField(commonFieldAncestor, readers)
}

func appendValue(record *Record, reader *Reader, value interface{}) {
  if reader.Field.Mode == "repeated" {
    if record.Values == nil {
      record.Values = make([]interface{}, 0, 100)
      record.Values[0] = value
    } else {
      record.Values = append(record.Values, value)
    }
  } else {
    record.Value = value
  }
}

func AssembleRecord(readers []*Reader) *Record {
  rootRecord := &Record{Name: "root", Children:RecordChildren{}}
  record := rootRecord

  rootReader := EmptyReader
  lastReader := rootReader

  counter := 0
  reader := readers[0]

  for reader.HasData() && counter < 20 {
    counter++
    row := reader.FetchNextRow()
    lowestCommonAncestor := getLowestCommonReaderAncestor(reader, lastReader, readers)
    if row.Value != "" {
      record = moveToLevel(record, reader, lastReader, lowestCommonAncestor)
      appendValue(record, reader, row.Value)
    } else {
      // this is still messed up
      record = moveToLevel(record, reader, lastReader, lowestCommonAncestor)
    }
    reader = reader.NextReader(readers)
    if (reader != EmptyReader) {
      lowestCommonAncestor = getLowestCommonReaderAncestor(reader, lastReader, readers)
      record = returnToLevel(record, reader, lastReader, lowestCommonAncestor)
    }
  }
  record = returnToLevel(record, rootReader, lastReader, EmptyReader)
  return rootRecord
}
