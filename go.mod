module github.com/f-secure-foundry/GoKey

go 1.14

require (
	github.com/f-secure-foundry/tamago v0.0.0-20200804103143-63eda71c22ec
	github.com/golang/protobuf v1.4.2 // indirect
	github.com/hsanjuan/go-nfctype4 v0.0.0-20200427083210-bd42ed8410a1
	github.com/keybase/go-crypto v0.0.0-20200123153347-de78d2cb44f4
	golang.org/x/crypto v0.0.0-20200728195943-123391ffb6de
	golang.org/x/sys v0.0.0-20200817085935-3ff754bf58a9 // indirect
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	google.golang.org/protobuf v1.25.0 // indirect
	gvisor.dev/gvisor v0.0.0-20200815070629-9a7b5830aa06
)

replace gvisor.dev/gvisor => github.com/f-secure-foundry/gvisor v0.0.0-20191224100818-98827aa91607
