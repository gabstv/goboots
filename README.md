goboots
=======
![version 0.4.1](https://img.shields.io/badge/v-0.4.1-blue.svg)  
  

![goboots](https://s3.amazonaws.com/gabstv-github/goboots.png)

A Web Framework written in Google Go.  
This is not fully ready for production.

## TODO:
- Change Websocket implementation to [Gorilla](https://github.com/gorilla/websocket)
- Setup documentation
- Setup starter project
- Add example projects

## Deprecation:
-- The old routing method will be removed on version 0.5

## Etc:

[now using Robfig's revel routing method](http://revel.github.io/manual/routing.html)  

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