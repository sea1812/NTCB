module BotDaemon

require (
	github.com/eclipse/paho.mqtt.golang v1.5.0
	github.com/gogf/gf/v2 v2.9.0
	github.com/sea1812/NTCB/AuthServer v0.0.0-20250801074826-0716ac4d621f
)

require (
	github.com/BurntSushi/toml v1.5.0 // indirect
	github.com/bwmarrin/snowflake v0.3.0 // indirect
	github.com/clbanning/mxj/v2 v2.7.0 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/grokify/html-strip-tags-go v0.1.0 // indirect
	github.com/magiconair/properties v1.8.10 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/olekukonko/tablewriter v1.0.9 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel v1.37.0 // indirect
	go.opentelemetry.io/otel/metric v1.37.0 // indirect
	go.opentelemetry.io/otel/sdk v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/sync v0.16.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/text v0.27.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/sea1812/NTCB/AuthServer => ../AuthServer

replace github.com/olekukonko/tablewriter => github.com/olekukonko/tablewriter v0.0.5

go 1.23.4
