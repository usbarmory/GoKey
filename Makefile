# https://github.com/f-secure-foundry/GoKey
#
# Copyright (c) F-Secure Corporation
# https://foundry.f-secure.com
#
# Use of this source code is governed by the license
# that can be found in the LICENSE file.

BUILD_USER = $(shell whoami)
BUILD_HOST = $(shell hostname)
BUILD_DATE = $(shell /bin/date -u "+%Y-%m-%d %H:%M:%S")
BUILD = ${BUILD_USER}@${BUILD_HOST} on ${BUILD_DATE}
REV = $(shell git rev-parse --short HEAD 2> /dev/null)
PKG = "github.com/f-secure-foundry/GoKey"

APP := gokey
GOENV := GO_EXTLINK_ENABLED=0 CGO_ENABLED=0 GOOS=tamago GOARM=7 GOARCH=arm
TEXT_START := 0x80010000 # ramStart (defined in imx6/imx6ul/memory.go) + 0x10000
GOFLAGS := -ldflags "-s -w -T $(TEXT_START) -E _rt0_arm_tamago -R 0x1000 -X '${PKG}/internal.Build=${BUILD}' -X '${PKG}/internal.Revision=${REV}'"
QEMU ?= qemu-system-arm -machine mcimx6ul-evk -cpu cortex-a7 -m 512M \
        -nographic -monitor none -serial null -serial stdio -net none \
        -semihosting -d unimp

SHELL = /bin/bash
DCD=imx6ul-512mb.cfg

.PHONY: clean qemu qemu-gdb

#### primary targets ####

all: $(APP)

imx: $(APP).imx

imx_signed: $(APP)-signed.imx

elf: $(APP)

#### utilities ####

check_tamago:
	@if [ "${TAMAGO}" == "" ] || [ ! -f "${TAMAGO}" ]; then \
		echo 'You need to set the TAMAGO variable to a compiled version of https://github.com/f-secure-foundry/tamago-go'; \
		exit 1; \
	fi

check_usbarmory_git:
	@if [ "${USBARMORY_GIT}" == "" ]; then \
		echo 'You need to set the USBARMORY_GIT variable to the path of a clone of'; \
		echo '  https://github.com/f-secure-foundry/usbarmory'; \
		exit 1; \
	fi

check_hab_keys:
	@if [ "${KEYS_PATH}" == "" ]; then \
		echo 'You need to set the KEYS_PATH variable to the path of secure/verified boot keys'; \
		echo 'See https://github.com/f-secure-foundry/usbarmory/wiki/Secure-boot-(Mk-II)'; \
		exit 1; \
	fi

check_bundled_keys:
	@if [ "${PGP_SECRET_KEY}" == "" ] || [ ! -f "${PGP_SECRET_KEY}" ]; then \
		echo 'You need to set the PGP_SECRET_KEY variable to the path of a valid PGP secret key'; \
		exit 1; \
	fi

clean:
	rm -f $(APP) gokey_vpcd
	@rm -fr $(APP).bin $(APP).imx $(APP)-signed.imx $(APP).csf

qemu: $(APP)
	$(QEMU) -kernel $(APP)

qemu-gdb: $(APP)
	$(QEMU) -kernel $(APP) -S -s

gokey_vpcd: check_bundled_keys
	cd $(CURDIR) && go generate
	go build -tags vpcd -o $(CURDIR)/gokey_vpcd || (rm -f $(CURDIR)/tmp.go && exit 1)
	rm -f $(CURDIR)/tmp.go

#### dependencies ####

$(APP): check_tamago check_bundled_keys
	cd $(CURDIR) && ${TAMAGO} generate
	$(GOENV) $(TAMAGO) build $(GOFLAGS) -o $(CURDIR)/$(APP) || (rm -f $(CURDIR)/tmp.go && exit 1)
	rm -f $(CURDIR)/tmp.go

$(APP).bin: $(APP)
	$(CROSS_COMPILE)objcopy -j .text -j .rodata -j .shstrtab -j .typelink \
	    -j .itablink -j .gopclntab -j .go.buildinfo -j .noptrdata -j .data \
	    -j .bss --set-section-flags .bss=alloc,load,contents \
	    -j .noptrbss --set-section-flags .noptrbss=alloc,load,contents\
	    $(APP) -O binary $(APP).bin

$(APP).imx: check_usbarmory_git $(APP).bin
	mkimage -n ${USBARMORY_GIT}/software/dcd/$(DCD) -T imximage -e $(TEXT_START) -d $(APP).bin $(APP).imx
	# Copy entry point from ELF file
	dd if=$(APP) of=$(APP).imx bs=1 count=4 skip=24 seek=4 conv=notrunc

#### secure boot ####

$(APP)-signed.imx: check_usbarmory_git check_hab_keys $(APP).imx
	${USBARMORY_GIT}/software/secure_boot/usbarmory_csftool \
		--csf_key ${KEYS_PATH}/CSF_1_key.pem \
		--csf_crt ${KEYS_PATH}/CSF_1_crt.pem \
		--img_key ${KEYS_PATH}/IMG_1_key.pem \
		--img_crt ${KEYS_PATH}/IMG_1_crt.pem \
		--table   ${KEYS_PATH}/SRK_1_2_3_4_table.bin \
		--index   1 \
		--image   $(APP).imx \
		--output  $(APP).csf && \
	cat $(APP).imx $(APP).csf > $(APP)-signed.imx
