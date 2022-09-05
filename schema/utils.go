package schema

type noopWriter struct{}

func (*noopWriter) Write(b []byte) (n int, err error) { return }
