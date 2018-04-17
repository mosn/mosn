package javaio

// Request : mesh request
type Serialize interface {
	WriteObject(stream *OutputObjectStream, obj *MiddleClass)

	ReadObject(stream *InputObjectStream, obj *MiddleClass)
}
