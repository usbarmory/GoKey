module github.com/f-secure-foundry/GoKey

go 1.14

require (
	github.com/f-secure-foundry/tamago v0.0.0-20200410130929-0c6c15f9fab2
	github.com/golang/protobuf v1.4.0 // indirect
	github.com/hsanjuan/go-nfctype4 v0.0.0-20181103161441-dc2aa9b8a60e
	github.com/keybase/go-crypto v0.0.0-20200123153347-de78d2cb44f4
	golang.org/x/crypto v0.0.0-20200422194213-44a606286825
	golang.org/x/sys v0.0.0-20200420163511-1957bb5e6d1f // indirect
	golang.org/x/time v0.0.0-20200416051211-89c76fbcd5d1 // indirect
	gvisor.dev/gvisor v0.0.0-20191224014503-95108940a01c
)

replace gvisor.dev/gvisor v0.0.0-20191224014503-95108940a01c => github.com/f-secure-foundry/gvisor v0.0.0-20191224100818-98827aa91607
