module github.com/f-secure-foundry/GoKey

go 1.15

require (
	github.com/albenik/go-serial/v2 v2.1.0 // indirect
	github.com/f-secure-foundry/armoryctl v0.0.0-20201106132543-cbcd84d6ba83
	github.com/f-secure-foundry/tamago v0.0.0-20201107175817-b08870c21469
	github.com/gsora/fidati v0.0.0-20201106184003-29320d613ab3
	github.com/hsanjuan/go-nfctype4 v0.0.1
	github.com/keybase/go-crypto v0.0.0-20200123153347-de78d2cb44f4
	golang.org/x/crypto v0.0.0-20201016220609-9e8e0b390897
	golang.org/x/sys v0.0.0-20201107080550-4d91cf3a1aaf // indirect
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	gvisor.dev/gvisor v0.0.0-20201107072220-78cce3a46b95
)

replace gvisor.dev/gvisor => github.com/f-secure-foundry/gvisor v0.0.0-20200812210008-801bb984d4b1
