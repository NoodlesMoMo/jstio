package adapter

import (
	"fmt"
	"testing"
)

func TestJstioLevelScan(t *testing.T) {
	fileName := `timeout/ok/fsafa/dfkasf`

	level, name := JstioLevelScan(fileName)
	fmt.Println("level:", level, "name:", name)
}
