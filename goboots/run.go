package main

import (
	"fmt"
	"gopkg.in/fsnotify.v1"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

var cmdRun = &Command{
	UsageLine: "run [file.go]",
	Short:     "Runs a Goboots App.",
	Long: `
Runs a Goboots App with live code reloading.
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
	if len(args) > 0 {
		defaultgofile = args[0]
		print(defaultgofile + "\n")
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		errorf("Could not init file watcher: " + err.Error() + "\n")
	}
	defer w.Close()
	wd, _ := os.Getwd()
	w.Add(wd)
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
			w.Add(p)
			print(p + "\n")
		} else {
			//print("FILE: " + p + "\n")
		}
		return nil
	})
	var cm *exec.Cmd
	start := func() {
		os.Remove("_goboots_main_")
		cmbuild := exec.Command("go", "build", "-o", "_goboots_main_", defaultgofile)
		cmbuild.Stderr = os.Stderr
		cmbuild.Stdout = os.Stdout
		if err := cmbuild.Start(); err != nil {
			print("Could not build the app: " + err.Error() + "\n")
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
						w.Add(evt.Name)
					} else {
						if fn == "_goboots_main_" || strings.HasPrefix(fn, ".") {
							break
						}
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
