goboots
=======
![version 0.4.0](https://img.shields.io/badge/v-0.4.0-blue.svg)  
  

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
[check yaml config for keys](http://godoc.org/gopkg.in/yaml.v2)

## Benchmarks

`go test -bench=. -benchmem  2>/dev/null`

####v 0.4.0 @ gabstv's iMac i7 3.4 Ghz (Mid 2011)
```
BenchmarkGoboots_Simple	      100000	     13943 ns/op
BenchmarkGoboots_Route15	  100000	     15589 ns/op
BenchmarkGoboots_Route75	  100000	     15249 ns/op
BenchmarkGoboots_Route150	  100000	     14724 ns/op
BenchmarkGoboots_Route300	  100000	     15476 ns/op
BenchmarkGoboots_Route3000	  100000	     15850 ns/op
BenchmarkGoboots_Middleware	  200000	     13594 ns/op
BenchmarkGoboots_Composite	  100000	     13935 ns/op
```

[Comparison with popular Go web frameworks](https://github.com/gabstv/golang-mux-benchmark)