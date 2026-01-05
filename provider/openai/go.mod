module github.com/hlfshell/gotonomy/provider/openai

go 1.24.2

require (
	github.com/hlfshell/gotonomy v0.1.0
	github.com/openai/openai-go/v3 v3.0.0
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/tidwall/gjson v1.14.4 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/hlfshell/gotonomy => ../..
