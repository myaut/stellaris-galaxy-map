package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/fsnotify/fsnotify"

	"github.com/myaut/stellaris-galaxy-map/pkg/sgm"
	"github.com/myaut/stellaris-galaxy-map/pkg/sgmrender"
)

var opts sgmrender.RenderOptions

func getSaveLocation() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	switch runtime.GOOS {
	case "windows":
		return filepath.Join(homeDir, `Documents\Paradox Interactive\Stellaris\save games`)
	case "linux":
		return filepath.Join(homeDir, ".local/share/Paradox Interactive/Stellaris/save games")
	case "darwin":
		return filepath.Join(homeDir, "Documents/Paradox Interactive/Stellaris/save games/")
	}

	panic("Unsupported platform")
}

func getOutFileName(fileName string) (string, error) {
	ext := filepath.Ext(fileName)
	if ext != ".sav" {
		return "", fmt.Errorf("unexpected file '%s' as input, it should have .sav extension", fileName)
	}

	return fileName[:len(fileName)-len(ext)] + ".svg", nil
}

func renderFile(fileName, outFileName string) error {
	state, err := sgm.LoadGameState(fileName)
	if err != nil {
		return err
	}

	r := sgmrender.NewRenderer(state, opts)
	r.Render()
	return r.Write(outFileName)
}

func main() {
	flag.BoolVar(&opts.ShowSectors, "sectors", false, "show sectors")
	flag.BoolVar(&opts.NoGrid, "no-grid", false, "hide grid")
	flag.BoolVar(&opts.NoInsignificantStars, "no-insignificant-stars", false,
		"hide insignificant stars")
	flag.BoolVar(&opts.NoStarSystems, "no-star-systems", false,
		"hide star systems: planets, starbases")
	flag.BoolVar(&opts.NoHyperLanes, "no-hyperlanes", false, "hide hyperlanes")
	flag.BoolVar(&opts.NoFleets, "no-fleets", false, "hide fleets")
	flag.Parse()
	args := flag.Args()

	if len(args) == 0 {
		runBackground()
		return
	}

	fileName := args[0]
	outFileName, err := getOutFileName(fileName)
	if err != nil {
		log.Fatal(err)
	}

	err = renderFile(fileName, outFileName)
	if err != nil {
		log.Fatal(err)
	}
}

func runBackground() {
	saveDir := askEmpireDir()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	err = watcher.Add(saveDir)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("waiting for save games in %s\n", saveDir)
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			if event.Op&fsnotify.Write == fsnotify.Write {
				ext := filepath.Ext(event.Name)
				if ext == ".svg" {
					continue
				}

				log.Println("got a save game:", event.Name)
				outFileName, err := getOutFileName(event.Name)
				if err != nil {
					log.Println(err)
				}

				err = renderFile(event.Name, outFileName)
				if err != nil {
					log.Println(err)
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("error:", err)
		}
	}
}

func askEmpireDir() string {
	saveRootDir := getSaveLocation()
	empireDirs, err := ioutil.ReadDir(saveRootDir)

	for idx, dir := range empireDirs {
		if !dir.IsDir() {
			continue
		}

		fmt.Printf("[%2d] - %s\n", idx+1, dir.Name())
	}

	fmt.Println("Select an empire:")

	var empireIdx int
	_, err = fmt.Scanf("%d", &empireIdx)
	if err != nil {
		log.Fatal(err.Error())
	}

	empireIdx--
	if empireIdx < 0 || empireIdx > len(empireDirs) {
		log.Fatalf("invalid empire #%d\n", empireIdx+1)
	}

	return filepath.Join(saveRootDir, empireDirs[empireIdx].Name())
}
