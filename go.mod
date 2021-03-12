module github.com/f-secure-foundry/GoKey

go 1.15

require (
	github.com/f-secure-foundry/armoryctl v0.0.0-20210203111727-1d302300d6e9
	github.com/f-secure-foundry/tamago v0.0.0-20210217103808-875e533027e3
	github.com/gsora/fidati v0.0.0-20210204160210-1bb65432acad
	github.com/hsanjuan/go-nfctype4 v0.0.1
	github.com/keybase/go-crypto v0.0.0-20200123153347-de78d2cb44f4
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a // indirect
	golang.org/x/sys v0.0.0-20210217090653-ed5674b6da4a // indirect
	golang.org/x/term v0.0.0-20201210144234-2321bbc49cbf // indirect
	golang.org/x/time v0.0.0-20201208040808-7e3f01d25324 // indirect
	gvisor.dev/gvisor v0.0.0-20210213011344-3ef012944d32
	periph.io/x/periph v3.6.7+incompatible // indirect
)

replace gvisor.dev/gvisor => github.com/f-secure-foundry/gvisor v0.0.0-20210201110150-c18d73317e0f
