### 0.10.5
- workaround for bug https://github.com/Joker/jade/issues/16

### 0.10.4
- New app.I18nProvider
- app.GetLangFunc

### 0.10.2
- Set ability to auto close slow connections

### 0.10.1
- Session.DeleteData(key)

### 0.10.0
- improved AppMonitor

### 0.9.9
- in.OutputContentAsJSON()

### 0.9.8
- ```in.BeforeOutput(func(in *goboots.In){})```

### 0.9.7
- Improved Mysql sessions

### 0.9.6
- Graceful restarts using [endless](https://github.com/gabstv/endless) (opt-in)

### 0.9.5
- in.Defer(func())

### 0.9.4
- get temp cert/key raw strings in config

### 0.9.3
- Watch and reload views automatically (experimental)

### 0.9.2
- .donotwatch file (.gitignore format)

### 0.9.0
- Jade/Pug template support

### 0.8.1
- InContent.Del(key)

### 0.8.0
- Static content filters
- No cache filter

### 0.7.1
- Separated access log

### 0.7.0
- Autocert support

### 0.6.8
- Mysql session db driver

### 0.6.1
- Fixed websocket route (use WS instead of *)
- Removed [ENROUTE] messages
