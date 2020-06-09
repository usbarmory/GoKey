module github.com/f-secure-foundry/GoKey

go 1.14

require (
	github.com/f-secure-foundry/tamago v0.0.0-20200609085001-1cf499249137
	github.com/golang/protobuf v1.4.2 // indirect
	github.com/hsanjuan/go-nfctype4 v0.0.0-20200427083210-bd42ed8410a1
	github.com/keybase/go-crypto v0.0.0-20200123153347-de78d2cb44f4
	golang.org/x/crypto v0.0.0-20200604202706-70a84ac30bf9
	golang.org/x/sys v0.0.0-20200602225109-6fdc65e7d980 // indirect
	golang.org/x/time v0.0.0-20200416051211-89c76fbcd5d1 // indirect
	google.golang.org/protobuf v1.24.0 // indirect
	gvisor.dev/gvisor v0.0.0-20200609060823-4e96b9491563
)

replace gvisor.dev/gvisor => github.com/f-secure-foundry/gvisor v0.0.0-20191224100818-98827aa91607
