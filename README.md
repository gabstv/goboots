goboots
=======
![version 0.4.6](https://img.shields.io/badge/v-0.4.6-blue.svg)  
  

![goboots](https://s3.amazonaws.com/gabstv-github/goboots.png)

A Web Framework written in Google Go.  
This is not fully ready for production.

## Installation:
run `go get -u github.com/gabstv/goboots/goboots`

## Project Setup
run `goboots new path/to/myprojectname` (e.g. `goboots new github.com/gabstv/mywebsite`)

## Scaffolding:
- Create a new controller:
  - Be at your project's base folder
  - Run `goboots scaff c MyControllerName` 

## Benchmarks

`go test -bench=. -benchmem  2>/dev/null`

####v 0.4.0 @ gabstv's iMac i7 3.4 Ghz (Mid 2011)
```
Simple	      100000	     13807 ns/op	    2269 B/op	      48 allocs/op
Route15	      100000	     15551 ns/op	    2625 B/op	      49 allocs/op
Route75	      100000	     15147 ns/op	    2483 B/op	      49 allocs/op
Route150	  100000	     14708 ns/op	    2766 B/op	      49 allocs/op
Route300	  100000	     15395 ns/op	    2481 B/op	      49 allocs/op
Route3000	  100000	     15842 ns/op	    2486 B/op	      49 allocs/op
Middleware	  200000	     13529 ns/op	    2418 B/op	      48 allocs/op
Composite	  100000	     13814 ns/op	    2783 B/op	      49 allocs/op
```

[Comparison with popular Go web frameworks](https://github.com/gabstv/golang-mux-benchmark)