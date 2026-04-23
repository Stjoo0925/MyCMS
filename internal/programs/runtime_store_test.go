package programs

type fakeRuntimeStore struct {
	state runtimeDocument
}

func (f *fakeRuntimeStore) Load() (runtimeDocument, error) {
	return f.state, nil
}

func (f *fakeRuntimeStore) Save(state runtimeDocument) error {
	f.state = state
	return nil
}

func alwaysMissingProcess(runtimeEntry) bool {
	return false
}
