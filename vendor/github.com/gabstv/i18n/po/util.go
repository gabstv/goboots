package po

// Unmarshal uses a Reader to read raw gettext data (.po .pot)
// and puts into a structured File
func Unmarshal(data []byte, f *File) error {
	rdr := Reader{}
	_, err := rdr.Read(data)
	if err != nil {
		return err
	}
	return rdr.Decode(f)
}
