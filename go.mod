module github.com/f-secure-foundry/GoKey

go 1.15

require (
	github.com/f-secure-foundry/tamago v0.0.0-20200901084418-da45d094d319
	github.com/hsanjuan/go-nfctype4 v0.0.0-20200427083210-bd42ed8410a1
	github.com/keybase/go-crypto v0.0.0-20200123153347-de78d2cb44f4
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a
	golang.org/x/sys v0.0.0-20200821140526-fda516888d29 // indirect
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	gvisor.dev/gvisor v0.0.0-20200821191014-2c422b7f21b3
)

replace gvisor.dev/gvisor => github.com/f-secure-foundry/gvisor v0.0.0-20200812210008-801bb984d4b1
