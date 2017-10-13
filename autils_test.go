package autils

import "testing"

func Test_PathExist(t *testing.T) {
	if !PathExist("/some/not/exist/path") && PathExist("/") {
		t.Log("right")
	} else {
		t.Error("error")
	}
}

func Test_RandomString(t *testing.T) {
	if len(RandomString(20)) == 20 {
		t.Log("right")
	} else {
		t.Error("error")
	}
}
