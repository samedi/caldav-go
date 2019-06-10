package test

func panicerr(err error) {
	if err != nil {
		panic(err)
	}
}
