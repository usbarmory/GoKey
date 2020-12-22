module github.com/f-secure-foundry/GoKey

go 1.15

require (
	github.com/f-secure-foundry/armoryctl v0.0.0-20201222082058-32dbba313ad2
	github.com/f-secure-foundry/tamago v0.0.0-20201222080844-0c719196f07d
	github.com/gsora/fidati v0.0.0-20201209091741-dd85cfe0480e
	github.com/hsanjuan/go-nfctype4 v0.0.1
	github.com/keybase/go-crypto v0.0.0-20200123153347-de78d2cb44f4
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a // indirect
	golang.org/x/sys v0.0.0-20201221093633-bc327ba9c2f0 // indirect
	golang.org/x/term v0.0.0-20201210144234-2321bbc49cbf // indirect
	golang.org/x/time v0.0.0-20201208040808-7e3f01d25324 // indirect
	gvisor.dev/gvisor v0.0.0-20201222062610-620de250a48a
	periph.io/x/periph v3.6.5+incompatible // indirect
)

replace gvisor.dev/gvisor => github.com/f-secure-foundry/gvisor v0.0.0-20200812210008-801bb984d4b1
