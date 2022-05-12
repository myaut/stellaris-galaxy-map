package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/myaut/stellaris-galaxy-mod/pkg/sgm"
	"github.com/myaut/stellaris-galaxy-mod/pkg/sgmrender"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Invalid number of arguments: 1 is expected")
	}

	fileName := os.Args[1]
	ext := filepath.Ext(fileName)
	if ext != ".sav" {
		log.Fatal("Unexpected file as input, it should have .sav extension")
	}
	outFileName := fileName[:len(fileName)-len(ext)] + ".svg"

	state, err := sgm.LoadGameState(fileName)
	if err != nil {
		log.Fatal(err.Error())
	}

	r := sgmrender.NewRenderer(state)
	r.Render()
	err = r.Write(outFileName)
	if err != nil {
		log.Fatal(err.Error())
	}
}
