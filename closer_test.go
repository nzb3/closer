package closer

import (
	"fmt"
	"testing"
)

func TestCloser_CloseAll(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			t.Fatal(err)
		}
	}()

	cf := func() error {
		fmt.Println("closeFunc called")
		return nil
	}

	Add(cf)

	CloseAll()

	Wait()

}
