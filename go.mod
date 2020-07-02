module github.com/f-secure-foundry/GoKey

go 1.14

require (
	github.com/f-secure-foundry/tamago v0.0.0-20200702080126-71cb40f0a220
	github.com/golang/protobuf v1.4.2 // indirect
	github.com/hsanjuan/go-nfctype4 v0.0.0-20200427083210-bd42ed8410a1
	github.com/keybase/go-crypto v0.0.0-20200123153347-de78d2cb44f4
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	golang.org/x/sys v0.0.0-20200625212154-ddb9806d33ae // indirect
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	google.golang.org/protobuf v1.25.0 // indirect
	gvisor.dev/gvisor v0.0.0-20200702050442-6c099d830091
)

replace gvisor.dev/gvisor => github.com/f-secure-foundry/gvisor v0.0.0-20191224100818-98827aa91607
