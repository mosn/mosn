package grpc

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type wrapper struct {
	grpc.ServerTransportStream
	header, trailer metadata.MD
}

func (w *wrapper) Method() string {
	return w.ServerTransportStream.Method()
}
func (w *wrapper) SetHeader(md metadata.MD) error{
	if err := w.ServerTransportStream.SetHeader(md); err != nil {
		return err
	}
	w.header = md
	return nil
}
func (w *wrapper) SendHeader(md metadata.MD) error{
	if err := w.ServerTransportStream.SendHeader(md); err != nil {
		return err
	}
	w.header = md
	return nil
}
func (w *wrapper) SetTrailer(md metadata.MD) error{
	if err := w.ServerTransportStream.SetTrailer(md); err != nil {
		return err
	}
	w.trailer = md
	return nil
}
