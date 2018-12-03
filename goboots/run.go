package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/monochromegane/go-gitignore"
	"gopkg.in/fsnotify.v1"
)

var cmdRun = &Command{
	UsageLine: "run [file.go] [[args]]",
	Short:     "Runs a Goboots App.",
	Long: `
Runs a Goboots App with live code reloading.

Flags:
	prebuild
        -prebuild=./prebuildscript.sh | Execute a file before building.
`,
}

func init() {
	cmdRun.Run = runApp
}

func dir_remainder(a string) string {
	sl := filepath.Dir(a)
	aa := strings.Split(sl, string(os.PathSeparator))
	return aa[len(aa)-1]
}

func runApp(args []string) {
	defaultgofile := "main.go"
	prebuildexec := ""
	if len(args) > 0 {
		defaultgofile = args[0]
		print(defaultgofile + "\n")
	}

	if len(args) > 1 {
		fs := flag.NewFlagSet("StaticFlags", flag.ContinueOnError)

		fs.StringVar(&prebuildexec, "prebuild", "", `-prebuild="./prebuildscript.sh"`)
		fs.Parse(args[1:])
	}

	var wcount int
	var wskip int
	w, err := fsnotify.NewWatcher()
	if err != nil {
		errorf("Could not init file watcher: " + err.Error() + "\n")
	}
	defer w.Close()
	wd, _ := os.Getwd()
	w.Add(wd)
	wcount++
	// get ignores!
	donotwatch := make([]func(path string, isDir bool) bool, 0)
	ignoresLoop := func(p string, i os.FileInfo, er error) error {
		if er != nil {
			return nil
		}
		if i.IsDir() {
			if strings.Contains(p, "/.") {
				return filepath.SkipDir
			}
			if strings.Contains(p, "/_") {
				return filepath.SkipDir
			}
			bdir := dir_remainder(p)
			if strings.HasPrefix(bdir, ".") {
				return filepath.SkipDir
			}
			// go to children
			//filepath.Walk(root, walkFn)
			return nil

		}
		//
		_, fn := filepath.Split(p)
		//print("FN: " + fn + "\n")
		if fn == ".donotwatch" {
			// read .donotwatch and add files
			fff, er9 := os.Open(p)
			if er9 != nil {
				print("ERROR READING " + fn + " (" + p + "): " + er9.Error() + "\n")
				return nil
			}
			gig := gitignore.NewGitIgnoreFromReader(p, fff)
			fff.Close()
			donotwatch = append(donotwatch, gig.Match)
			print("donotwatch: " + p + "\n")
		}
		return nil
	}
	filepath.Walk(wd, ignoresLoop)
	//TODO: replace walk function
	filepath.Walk(wd, func(p string, i os.FileInfo, er error) error {
		if er != nil {
			return nil
		}
		if i.IsDir() {
			if strings.Contains(p, "/.") {
				return filepath.SkipDir
			}
			if strings.Contains(p, "/_") {
				return filepath.SkipDir
			}
			bdir := dir_remainder(p)
			if strings.HasPrefix(bdir, ".") {
				return filepath.SkipDir
			}

			for _, func0 := range donotwatch {
				if func0(p, true) {
					return filepath.SkipDir
				}
			}

			w.Add(p)
			wcount++
			print(p + "\n")
		} else {
			//print("FILE: " + p + "\n")
		}
		return nil
	})
	filepath.Walk(wd, func(p string, i os.FileInfo, er error) error {
		if er != nil {
			return nil
		}
		if i.IsDir() {
			return nil
		}
		for _, func0 := range donotwatch {
			if func0(p, true) {
				if wer := w.Remove(p); wer == nil {
					wskip++
				}
				return nil
			}
		}
		return nil
	})
	print(fmt.Sprintln("Watching", wcount, "files"))
	print(fmt.Sprintln("Skipped", wskip, "files"))
	var cm *exec.Cmd
	start := func() {
		os.Remove("_goboots_main_")
		gp := os.Getenv("GOPATH")
		cmbuild := exec.Command("go", "build", "-o", "_goboots_main_", defaultgofile)
		cmbuild.Env = []string{"GOPATH=" + gp}
		cmbuild.Stderr = os.Stderr
		cmbuild.Stdout = os.Stdout
		// run prebuild if any
		if len(prebuildexec) > 0 {
			cmpre := exec.Command(prebuildexec)
			cmbuild.Stderr = os.Stderr
			cmbuild.Stdout = os.Stdout
			fmt.Println("Running prebuild command", prebuildexec)
			cmpre.Run()
		}
		//
		if err := cmbuild.Start(); err != nil {
			print("Could not build the app: " + err.Error() + "\n")
			wwwddd, _ := os.Getwd()
			print(wwwddd + "\n")
			cm = nil
		} else {
			err := cmbuild.Wait()
			if err != nil {
				fmt.Println("Couldnot wait", err)
			}
			time.Sleep(time.Millisecond * 50)
			cm = exec.Command(filepath.Join(wd, "_goboots_main_"))
			cm.Stderr = os.Stderr
			cm.Stdout = os.Stdout
			err = cm.Start()
			if err != nil {
				print("Could not init the app: " + err.Error() + "\n")
			} else {
				if runtime.GOOS == "darwin" {
					cmnot := exec.Command("terminal-notifier", "-message", "Goboots app started!")
					if cmer := cmnot.Run(); cmer != nil {
						print("\nGoboots app started! You can get a notification up if you install terminal-notifier:\n")
						print("brew install terminal-notifier\n")
					}
					print("\n\n")
				}
				//TODO: notification for linux/windows
			}
		}
	}
	stop := func() {
		if cm != nil && cm.Process != nil {
			ok := false
			go func() {
				err := cm.Wait()
				if err != nil {
					print(fmt.Sprintln(err))
				}
				ok = true
			}()
			cm.Process.Kill()
			for !ok {
				time.Sleep(time.Millisecond * 50)
			}
		}
	}
	start()

	//
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		s := <-c
		fmt.Println("Got signal: ", s)
		if cm != nil && cm.Process != nil {
			cm.Process.Kill()
			time.Sleep(time.Millisecond * 10)
		}
		os.Remove("_goboots_main_")
		os.Exit(1)
	}()
	//

	for {
		select {
		case evt := <-w.Events:
			fmt.Printf("File %v %v\n", evt.Name, evt.Op)
			if evt.Op == fsnotify.Write || evt.Op == fsnotify.Create {
				if evt.Op == fsnotify.Create {
					_, fn := filepath.Split(evt.Name)
					if fn == "" {
						// it's a dir
						bdir := dir_remainder(evt.Name)
						if strings.HasPrefix(bdir, ".") {
							break
						}
						skipok := false
						for _, v := range donotwatch {
							if v(evt.Name, true) {
								skipok = true
								break
							}
						}
						if skipok {
							print("donotwatch caught " + evt.Name + "\n")
							break
						}
						w.Add(evt.Name)
					} else {
						if fn == "_goboots_main_" || strings.HasPrefix(fn, ".") {
							break
						}
					}
				}
				if fi, z := os.Stat(evt.Name); z == nil {
					isdir := fi.IsDir()
					skipok := false
					for _, v := range donotwatch {
						if v(evt.Name, isdir) {
							skipok = true
							break
						}
					}
					if skipok {
						print("donotwatch caught " + evt.Name + "\n")
						break
					}
				} else {
					skipok := false
					for _, v := range donotwatch {
						if v(evt.Name, true) {
							skipok = true
							break
						}
						if v(evt.Name, false) {
							skipok = true
							break
						}
					}
					if skipok {
						print("donotwatch caught " + evt.Name + "\n")
						break
					}
				}
				print("Will restart the app.\n")
				if runtime.GOOS == "darwin" {
					cmnot := exec.Command("terminal-notifier", "-message", "Goboots app will restart.")
					if cmer := cmnot.Run(); cmer != nil {
						//print("\nGoboots app started! You can get a notification up if you install terminal-notifier:\n")
						//print("brew install terminal-notifier\n")
					}
					print("\n\n")
				}
				stop()
				go func() {
					for i := 0; i < 1100; i++ {
						select {
						case e := <-w.Events:
							fmt.Print(e.Name)
						default:
							time.Sleep(time.Millisecond)
						}
					}
				}()
				fmt.Print("\n")
				time.Sleep(time.Millisecond * 1500)
				start()
			}
		case er := <-w.Errors:
			print("Error: " + er.Error() + "\n")
		}
	}
}
